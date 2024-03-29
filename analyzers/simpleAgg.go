package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/collector/pushFunc"
)

var AggregationFunc = map[string]aggFunc{
	"max": max,
	"min": min,
}

type aggFunc func(*Aggregation, []*pushFunc.DataPair) float64

// AnaMax is a stateless analyzer, it find the maximal value
// in past Duration.
type Aggregation struct {
	Desc        *collector.Desc
	Duration    time.Duration
	lastAnalyze time.Time
	mtx         sync.Mutex
	aggFunc     aggFunc
	alert       *Alert
}

func (a *Aggregation) Describe(ch chan<- *collector.Desc) {
	ch <- a.Desc
}

func (a *Aggregation) Analyze(data []*pushFunc.DataPair, ch chan<- collector.Metric) {
	if len(data) <= 0 {
		return
	}
	a.mtx.Lock()
	oldestTime := time.Now().Add(-a.Duration)
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
		result := a.aggFunc(a, data[start:])
		tp := time.Now()
		cm, err := collector.NewConstMetric(
			a.Desc,
			collector.GaugeValue,
			result,
		)
		if err != nil {
			glog.Error(err)
			return
		}
		ch <- collector.NewTimeStampMetric(tp, cm)

		if a.alert != nil && a.alert.compare(result, tp) {
			ch <- a.alert
		}
	}
}

// type AnaMaxOpt is used to initialize the AnaMax in func NewAnaMax
type AggregationOpts struct {
	collector.Opts `yaml:"desc"`
	Duration       time.Duration `yaml:"duration"`
	Type           string        `yaml:"type"`
	Alert          string        `yaml:"alert,omitempty"`
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
	desc := collector.NewDesc(
		opt.Name,
		opt.Help,
		opt.Level,
		opt.Priority,
		nil,
		newLabels,
	)

	alert, err := NewAlertFromStr(
		&opt.Opts,
		collector.Labels{"analyzer": opt.Type},
		opt.Alert,
	)
	if err != nil {
		return nil, err
	}
	return &Aggregation{
		Desc:        desc,
		Duration:    opt.Duration,
		lastAnalyze: time.Now(),
		aggFunc:     AggregationFunc[opt.Type],
		alert:       alert,
	}, nil
}

func (a *AggregationOpts) NewStatelessAna() (collector.StatelessAnalyzer, error) {
	return NewAggregation(a)
}

func max(a *Aggregation, data []*pushFunc.DataPair) float64 {
	idx := len(data) - 1
	max := data[idx]
	idx--

	for ; idx >= 0; idx-- {
		if data[idx].Value > max.Value {
			max = data[idx]
		}
	}
	return max.Value
}

func min(a *Aggregation, data []*pushFunc.DataPair) float64 {
	idx := len(data) - 1
	min := data[idx]
	idx--

	for ; idx >= 0; idx-- {
		if data[idx].Value < min.Value {
			min = data[idx]
		}
	}
	return min.Value
}

//func quantile(a *Aggregation, data []*collector.DataPair, ch chan<- collector.Metric) {
//	if len(a.QuaTarget) == 0 || len(data) == 0 {
//		return
//	}
//	q := qua.NewTargeted(a.QuaTarget...)
//	for _, d := range data {
//		q.Insert(d.Value)
//	}
//	for _, tar := range a.QuaTarget {
//		cm, err := collector.NewConstMetric(
//			a.Desc,
//			collector.GaugeValue,
//			q.Query(tar),
//		)
//		if err != nil {
//			glog.Error(err)
//		}
//		ch <- collector.NewTimeStampMetric(time.Now(), cm)
//	}
//}
