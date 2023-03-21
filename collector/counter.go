package collector

import (
	"errors"
	"sync"

	"wanggj.com/abyss/module"
)

type Counter interface {
	Metric
	Collector

	// Inc increments the counter by 1.
	Inc()
	// Add adds the given value to the counter.The value must be non-negative
	Add(float64)
}

// CounterOpts is an alias for Opts.
type CounterOpts Opts

type counter struct {
	value uint64

	desc *Desc

	labelPairs []*module.LabelPair
	mtx        sync.RWMutex // mtx used to concurrently collect and write
}

func NewCounter(opts CounterOpts) Counter {
	desc := NewDesc(
		opts.Name,
		opts.Help,
		opts.Level,
		nil,
		opts.ConstLabels,
	)
	result := &counter{
		desc:       desc,
		labelPairs: desc.constLabelPairs,
	}
	return result
}

func (c *counter) Desc() *Desc {
	return c.desc
}

func (c *counter) Write() (*module.Metric, error) {
	result := &module.Metric{}
	c.mtx.RLock()
	err := populateMetric(CounterValue, float64(c.value), c.labelPairs, result)
	c.mtx.RUnlock()
	return result, err
}

func (c *counter) Add(v float64) {
	if v < 0 {
		panic(errors.New("counter cannot decrease in value"))
	}

	ival := uint64(v)
	c.mtx.Lock()
	c.value += ival
	c.mtx.Unlock()
}

func (c *counter) Inc() {
	c.mtx.Lock()
	c.value++
	c.mtx.Unlock()
}

func (c *counter) Describe(ch chan<- *Desc) {
	ch <- c.desc
}

func (c *counter) Collect(ch chan<- Metric) {
	ch <- c
}
