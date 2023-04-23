package collector

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"

	"google.golang.org/protobuf/proto"
	"wanggj.com/abyss/collector/internal"
	"wanggj.com/abyss/module"
)

const (
	// capacity for channel to collect metrics and descriptors.
	capMetricChan = 100
	capDescChan   = 10
)

// Registerer is the interface for the part of a registry in charge of registering and
// unregistering. Users of custom registries should use Registerer as type for registration
// purposses (rather than the Registry type directly).
type Registerer interface {
	// Register registers a new Collector to be included in metrics collection.
	// It returns an error if the descriptors provided by the Collector are invalid.
	//
	// If the provided Collecotr is equal to a Collector already registered, the
	// returned error is an instance of AlreadyRegisteredError, which contains the
	// Previously registered Collector.
	Register(Collector) error
	// Unregister unregisters the Collector that equals the Collector passed in as an
	// argument. (Two Collectors are considered equal if their Describe method yields
	// the same set of descriptors.) The function returns whether a Collector
	// was unregistered.
	Unregister(Collector)
}

// Gather is the interface for the part of a registry in charge of gathering the collected
// the collected metrics into a number of MetricFamilies. The Gatherer interface comes with
// the same general implication as described for the Registrterer.
type Gatherer interface {
	// Gather calls the Collect method of the registered Collectors and then gathers
	// the collected metrics into a lexicographically sorted slice of uniquely named
	// MetricFamily protobufs. Gather ensures that that the returned slice is valid
	// and self-consistent so that it can be used for valid exposition.
	//
	// Even if an error occurs, Gather attempts to gather as many metrics as possible.
	// Hence, if a non-nil error is returned, the returned MetricFamily slice could
	// be nil or contain a number of MetricFamily protobufs, some of which might be
	// incomplete, and some might be missing altogether. The returned error explains the
	// details. Note that this is mostly useful for debugging purposes.
	//
	// The result is a map, used for classify metrics
	Gather() (map[int][]*module.MetricFamily, error)
}

// AlreadyRegisteredError is returned by the Register method if the Collector to
// be registered has already been registered before, or a different Collector
// that collects the same metrics has been registered before. Registration fails
// in that case, but you can detect from the kind of error what has
// happened. The error contains fields for the existing Collector and the
// (rejected) new Collector that equals the existing one. This can be used to
// find out if an equal Collector has been registered before and switch over to
// using the old one, as demonstrated in the example.
type AlreadyRegisteredError struct {
	ExistingCollector, NewCollector Collector
}

func (err AlreadyRegisteredError) Error() string {
	return "Duplicate metrics collector registration attempted."
}

// MultiError is a slice of errors implementing the error interface. It is used by Gatherer
// to report multiple errors during MetricFamily gathering.
type MultiError []error

func (errs MultiError) Error() string {
	if len(errs) == 0 {
		return ""
	}
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%d error(s) occurred:", len(errs))
	for _, err := range errs {
		fmt.Fprintf(buf, "\n\t* %s", err)
	}
	return buf.String()
}

func (errs *MultiError) Append(err error) {
	if err != nil {
		*errs = append(*errs, err)
	}
}

// Registry registers prometheus collectors, collects their metrics, and gathers them
// into MetricFamilies for exposition. It implements Registerer, Gatherer, and Collector.
// The zero value is not usable. Create instances with NewRegistry.
//
// Registry implements Collector to allow it to be used for creating groups of metrics.
// See the Grouping example for how this can be down.
type Registry struct {
	mtx            sync.RWMutex
	pushersByID    map[uint64]Pusher
	collectorsByID map[uint64]Collector // ID is a hash of the descIDs
	descIDs        map[uint64]struct{}
}

// Registry implements Registerer.
func (r *Registry) Register(c Collector) error {
	var (
		descChan         = make(chan *Desc, capDescChan)
		newDescIDs       = map[uint64]struct{}{}
		collectorID      uint64 // All desc IDs XOR'd together
		duplicateDescErr error
	)
	go func() {
		c.Describe(descChan)
		close(descChan)
	}()
	r.mtx.Lock()
	defer func() {
		// Drain channel in case of premature return to not leak a goroutine.
		for range descChan {
		}
		r.mtx.Unlock()
	}()
	// Conduct various tests...
	for desc := range descChan {
		// 1.Is the descriptor valid?
		if desc.err != nil {
			return fmt.Errorf("descriptor %s is invalid: %v", desc, desc.err)
		}

		// 2.Is the DescID unique in registry?
		// (i.e. name + constLabel combination unique for all DescID in registry)
		if _, exists := r.descIDs[desc.id]; exists {
			duplicateDescErr = fmt.Errorf(
				"descriptor %s already exists with the same name and const label values",
				desc,
			)
			break
		}

		// If it is not a duplicate desc in this collector, XOR it to the
		// collectorID.
		if _, exists := newDescIDs[desc.id]; !exists {
			newDescIDs[desc.id] = struct{}{}
			collectorID ^= desc.id
		}
	}
	// if collector already exists, return AlreadyRegisteredError
	if e, exists := r.collectorsByID[collectorID]; exists {
		return AlreadyRegisteredError{
			ExistingCollector: e,
			NewCollector:      c,
		}
	}

	// duplicate collector is more important than duplicate Desc
	if duplicateDescErr != nil {
		return duplicateDescErr
	}

	// Add new collector to Registry
	r.collectorsByID[collectorID] = c
	for hash := range newDescIDs {
		r.descIDs[hash] = struct{}{}
	}
	return nil
}

// Unregister implements Registerer
func (r *Registry) Unregister(c Collector) {
	var (
		descChan    = make(chan *Desc, capDescChan)
		descIDs     = map[uint64]struct{}{}
		collectorID uint64
	)
	go func() {
		c.Describe(descChan)
		close(descChan)
	}()
	for desc := range descChan {
		if _, exist := descIDs[desc.id]; !exist {
			collectorID ^= desc.id
			descIDs[desc.id] = struct{}{}
		}
	}

	r.mtx.RLock()
	if _, exists := r.collectorsByID[collectorID]; !exists {
		r.mtx.RUnlock()
		return
	}
	r.mtx.RUnlock()

	r.mtx.Lock()
	defer r.mtx.Unlock()

	delete(r.collectorsByID, collectorID)
	for id := range descIDs {
		delete(r.descIDs, id)
	}
	return
}

func (r *Registry) Gather() (map[int][]*module.MetricFamily, error) {
	r.mtx.RLock()

	if len(r.collectorsByID) == 0 {
		r.mtx.RUnlock()
		return nil, nil
	}

	var (
		metricChan = make(chan Metric, capMetricChan)
		wg         sync.WaitGroup
		errs       MultiError
	)

	goroutineBudget := len(r.collectorsByID)
	metricFamiliesByName := make(map[string]*module.MetricFamily, len(r.descIDs))
	collectors := make(chan Collector, len(r.collectorsByID))
	for _, collector := range r.collectorsByID {
		collectors <- collector
	}
	r.mtx.RUnlock()

	wg.Add(goroutineBudget)

	collectWorker := func() {
		for {
			select {
			case collector := <-collectors:
				collector.Collect(metricChan)
			default:
				return
			}
			wg.Done()
		}
	}

	// Start the first worker now to make sure at least one is running
	go collectWorker()
	goroutineBudget--

	// Close metricChan once all collectors are closed
	go func() {
		wg.Wait()
		close(metricChan)
	}()

	// Drain metricChan in case of premature return
	defer func() {
		if metricChan != nil {
			for range metricChan {
			}
		}
	}()

	// Copy the channel references so when if the channel has been Drained,
	// we can nil it out
	mc := metricChan

	for {
		select {
		case metric, ok := <-mc:
			if !ok {
				mc = nil
				break
			}
			errs.Append(processMetric(metric, metricFamiliesByName))

		default:
			if goroutineBudget <= 0 || len(collectors) == 0 {
				select {
				case metric, ok := <-mc:
					if !ok {
						mc = nil
						break
					}
					errs.Append(processMetric(metric, metricFamiliesByName))
				}
				break
			}

			// start more collectorWorkers
			go collectWorker()
			goroutineBudget--
			runtime.Gosched()
		}
		// Once metricChan has drained, mc will bi nil
		if mc == nil {
			break
		}
	}
	return internal.NormalizeMetricFamilies(metricFamiliesByName), errs
}

func processMetric(
	metric Metric,
	metricFamiliesByName map[string]*module.MetricFamily,
) error {
	desc := metric.Desc()
	// Wrapped metrics collected by an unchecked Collector can have an invalid Desc.
	if desc.err != nil {
		return desc.err
	}

	mdlMetric, err := metric.Write()
	if err != nil {
		return fmt.Errorf("error collecting metric %v: %w", desc, err)
	}
	metricFamily, ok := metricFamiliesByName[desc.name]
	if ok {
		// this metric desc has existed
		switch metricFamily.GetType() {
		case module.MetricType_COUNTER:
			if mdlMetric.Counter == nil {
				return fmt.Errorf(
					"collected metric %s %s should be a Counter",
					desc.name, mdlMetric,
				)
			}
		case module.MetricType_GAUGE:
			if mdlMetric.Gauge == nil {
				return fmt.Errorf(
					"collected metric %s %s should be a Counter",
					desc.name, mdlMetric,
				)
			}
		case module.MetricType_EVENT:
			if mdlMetric.Event == nil {
				return fmt.Errorf(
					"collected metric %s %s should be a Counter",
					desc.name, mdlMetric,
				)
			}
		case module.MetricType_SUMMARY:
			if mdlMetric.Summary == nil {
				return fmt.Errorf(
					"collected metric %s %s should be a Counter",
					desc.name, mdlMetric,
				)
			}
		case module.MetricType_HISTOGRAM:
			if mdlMetric.Histogram == nil {
				return fmt.Errorf(
					"collected metric %s %s should be a Counter",
					desc.name, mdlMetric,
				)
			}
		default:
			panic("encouraged MetricFamily with invalid type")
		}
	} else {
		// get a new name
		metricFamily = &module.MetricFamily{}
		metricFamily.Name = proto.String(desc.name)
		switch {
		case mdlMetric.Gauge != nil:
			metricFamily.Type = module.MetricType_GAUGE.Enum()
		case mdlMetric.Counter != nil:
			metricFamily.Type = module.MetricType_COUNTER.Enum()
		case mdlMetric.Event != nil:
			metricFamily.Type = module.MetricType_EVENT.Enum()
		case mdlMetric.Histogram != nil:
			metricFamily.Type = module.MetricType_HISTOGRAM.Enum()
		case mdlMetric.Summary != nil:
			metricFamily.Type = module.MetricType_SUMMARY.Enum()
		default:
			return fmt.Errorf("empty metric collected: %s", mdlMetric)
		}
	}
	metricFamily.Metric = append(metricFamily.Metric, mdlMetric)
	return nil
}
