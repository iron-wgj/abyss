package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"wanggj.com/abyss/collector"
)

var AggregationFunc = map[string]aggFunc{
	"max": max,
	"min": min,
}

type aggFunc func(*collector.Desc, []*collector.DataPair, chan<- collector.Metric)

// AnaMax is a stateless analyzer, it find the maximal value
// in past Duration.
type Aggregation struct {
	Desc        *collector.Desc
	Duration    time.Duration
	lastAnalyze time.Time
	mtx         sync.Mutex
	aggFunc     aggFunc
}

func (a *Aggregation) Describe(ch chan<- *collector.Desc) {
	ch <- a.Desc
}

func (a *Aggregation) Analyze(data []*collector.DataPair, ch chan<- collector.Metric) {
	if len(data) <= 0 {
		return
	}
	a.mtx.Lock()
	oldestTime := time.Now().Add(a.Duration)
	if oldestTime.Before(a.lastAnalyze) {
		a.mtx.Unlock()
		return
	}
	a.lastAnalyze = time.Now()
	a.mtx.Unlock()

	start := len(data)
	for ; start > 0 && data[start-1].Timestamp.After(oldestTime); start-- {
	}
	if start < len(data) && a.aggFunc != nil {
		a.aggFunc(a.Desc, data[start:], ch)
	}
}

// type AnaMaxOpt is used to initialize the AnaMax in func NewAnaMax
type AggregationOpts struct {
	collector.Opts
	Duration time.Duration `yaml:"duration"`
	Type     string
}

func NewAggregation(opt *AggregationOpts) (*Aggregation, error) {
	if opt.Duration.Abs() > time.Minute*10 {
		return nil, fmt.Errorf("Duration of Opt should no longer than 10min.")
	}
	if _, ok := AggregationFunc[opt.Type]; !ok {
		return nil, fmt.Errorf("Aggregation not exists.")
	}

	if err := checkOptLabels(
		opt.ConstLabels,
		[]string{"analyzer", "analyzer_duration"},
	); err != nil {
		return nil, err
	}
	newLabels := make(collector.Labels)
	for n, v := range opt.ConstLabels {
		newLabels[n] = v
	}
	newLabels["analyzer"] = opt.Type
	newLabels["analyzer_duration"] = opt.Duration.Abs().String()
	desc := collector.NewDesc(opt.Name, opt.Help, opt.Level, nil, opt.ConstLabels)
	return &Aggregation{
		Desc:        desc,
		Duration:    opt.Duration,
		lastAnalyze: time.Now(),
	}, nil
}

func (a *AggregationOpts) NewStatelessAna() (collector.StatelessAnalyzer, error) {
	return NewAggregation(a)
}

func max(desc *collector.Desc, data []*collector.DataPair, ch chan<- collector.Metric) {
	idx := len(data) - 1
	max := data[idx]
	idx--

	for ; idx >= 0; idx-- {
		if data[idx].Value > max.Value {
			max = data[idx]
		}
	}
	cm, err := collector.NewConstMetric(
		desc,
		collector.GaugeValue,
		max.Value,
	)
	if err != nil {
		glog.Error(err)
		return
	}
	ch <- collector.NewTimeStampMetric(
		time.Now(),
		cm,
	)
}

func min(desc *collector.Desc, data []*collector.DataPair, ch chan<- collector.Metric) {
	idx := len(data) - 1
	max := data[idx]
	idx--

	for ; idx >= 0; idx-- {
		if data[idx].Value < max.Value {
			max = data[idx]
		}
	}
	cm, err := collector.NewConstMetric(
		desc,
		collector.GaugeValue,
		max.Value,
	)
	if err != nil {
		glog.Error(err)
		return
	}
	ch <- collector.NewTimeStampMetric(
		time.Now(),
		cm,
	)
}
