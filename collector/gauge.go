package collector

import (
	"sync"

	"wanggj.com/abyss/module"
)

type Gauge interface {
	Metric
	Collector

	// Set sets the Gauge to an arbitry value
	Set(float64)
	// Add adds the given value to the Gauge.(The value can be negative,
	// resulting in a decrease of the Gauge.)
	Add(float64)
	// Sub subtracts the given value form the Gauge.(The value can be
	// negative i, resulting in an increase of the Gauge.)
	Sub(float64)
}

type GaugeOpts Opts

// NewGauge creates a new Gauge based on the provided GaugeOpts.
//
// The returned implementation is optimized for a fast Set method. If you have a
// choice for managing the value of a Gauge via Set vs. Inc/Dec/Add/Sub, pick
// the former. For example, the Inc method of the returned Gauge is slower than
// the Inc method of a Counter returned by NewCounter. This matches the typical
// scenarios for Gauges and Counters, where the former tends to be Set-heavy and
// the latter Inc-heavy.
func NewGauge(opts GaugeOpts) Gauge {
	desc := NewDesc(
		opts.Name,
		opts.Help,
		opts.Level,
		nil,
		opts.ConstLabels,
	)
	result := &gauge{
		value:      0,
		desc:       desc,
		labelPairs: desc.constLabelPairs,
	}
	return result
}

type gauge struct {
	value      float64
	desc       *Desc
	labelPairs []*module.LabelPair

	mtx sync.RWMutex
}

func (g *gauge) Desc() *Desc {
	return g.desc
}

func (g *gauge) Write() (*module.Metric, error) {
	result := &module.Metric{}
	g.mtx.RLock()
	defer g.mtx.RUnlock()
	if err := populateMetric(GaugeValue, float64(g.value), g.labelPairs, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (g *gauge) Set(v float64) {
	g.mtx.Lock()
	g.value = v
	g.mtx.Unlock()
}

func (g *gauge) Add(v float64) {
	g.mtx.Lock()
	g.value += v
	g.mtx.Unlock()
}

func (g *gauge) Sub(v float64) {
	g.mtx.Lock()
	g.value -= v
	g.mtx.Unlock()
}

func (g *gauge) Describe(ch chan<- *Desc) {
	ch <- g.desc
}

func (g *gauge) Collect(ch chan<- Metric) {
	ch <- g
}
