package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/bmizerany/perks/quantile"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/collector/pushFunc"
)

// QuatileAnalyzer can be either stateful or stateless, depending on the
// option, but cannot be both at the same time, otherwise it will get
// incorrect result when Collect and Analyze
type QuantileAnalyzer struct {
	Ranks     map[float64]*Alert
	Desc      *collector.Desc
	targetNum int

	count  uint64
	sum    float64
	stream *quantile.Stream
	mtx    sync.Mutex
}

func (q *QuantileAnalyzer) getResults() []collector.Metric {
	qs := map[float64]float64{}
	result := []collector.Metric{}
	timestamp := time.Now()
	for k := range q.Ranks {
		qs[k] = q.stream.Query(k)
	}
	sum := collector.NewConstSummary(
		q.Desc,
		q.count,
		q.sum,
		qs,
	)
	result = append(result, collector.NewTimeStampMetric(timestamp, sum))

	for k, mv := range qs {
		if q.Ranks[k] == nil {
			continue
		}
		if q.Ranks[k].compare(mv, timestamp) {
			result = append(result, q.Ranks[k])
		}
	}

	return result
}

func (q *QuantileAnalyzer) Describe(ch chan<- *collector.Desc) {
	ch <- q.Desc
}

func (q *QuantileAnalyzer) collectMetric(reset bool, ch chan<- collector.Metric) {
	result := q.getResults()
	if reset {
		q.stream.Reset()
		q.count = 0
		q.sum = 0
	}

	for _, v := range result {
		ch <- v
	}
}

func (q *QuantileAnalyzer) insert(value float64) {
	q.stream.Insert(value)
	q.count++
	q.sum += value
}

func (q *QuantileAnalyzer) Collect(ch chan<- collector.Metric) {
	if q.targetNum <= 0 {
		return
	}
	q.mtx.Lock()
	q.collectMetric(false, ch)
	q.mtx.Unlock()
}

func (q *QuantileAnalyzer) Observe(data *pushFunc.DataPair) {
	if q.targetNum <= 0 {
		return
	}
	q.mtx.Lock()
	q.insert(data.Value)
	q.mtx.Unlock()
}

func (q *QuantileAnalyzer) Analyze(data []*pushFunc.DataPair, ch chan<- collector.Metric) {
	if q.targetNum <= 0 {
		return
	}
	q.mtx.Lock()
	for _, d := range data {
		q.insert(d.Value)
	}
	q.collectMetric(true, ch)
	q.mtx.Unlock()

}

// Opts used to generate QuantileAnalyzer, Ranks is the predefined targets when analyze
// ConstLabels must not contain "analyzer" and "quatileTarget".
//
// Quatile implements interface StatefulAnaOpt and StatelessAnaOpt
type QuantileOpts struct {
	collector.Opts `yaml:"desc"`
	Ranks          map[float64]string `yaml:"targets"`
}

func NewQuatileAna(q *QuantileOpts) (*QuantileAnalyzer, error) {
	if err := checkOptLabels(q.ConstLabels, []string{"analyzer", "quatileTarget"}); err != nil {
		return nil, err
	}
	newLabels := collector.Labels{}
	for n, v := range q.ConstLabels {
		newLabels[n] = v
	}
	newLabels["analyzer"] = "Quatile"

	rankWithAlert := map[float64]*Alert{}
	ranks := make([]float64, 0, len(q.Ranks))
	desc := collector.NewDesc(
		q.Name,
		q.Help,
		q.Level,
		q.Priority,
		nil,
		newLabels,
	)
	// descs := make([]*collector.Desc, 0, len(q.Ranks))
	// for _, t := range q.Ranks {
	// 	newLabels["quatileTarget"] = fmt.Sprint(t)
	// 	desc := collector.NewDesc(
	// 		q.Name,
	// 		q.Help,
	// 		q.Level,
	// 		nil,
	// 		newLabels,
	// 	)
	// 	descs = append(descs, desc)
	// }
	for k, v := range q.Ranks {
		r, err := NewAlertFromStr(
			&q.Opts,
			collector.Labels{
				"quantile_rank": fmt.Sprint(k),
			},
			v,
		)
		if err != nil {
			return nil, err
		}
		rankWithAlert[k] = r
		ranks = append(ranks, k)
	}
	stream := quantile.NewTargeted(ranks...)

	return &QuantileAnalyzer{
		Ranks:     rankWithAlert,
		Desc:      desc,
		targetNum: len(q.Ranks),
		stream:    stream,
	}, nil
}

func (qo *QuantileOpts) NewStatelessAna() (collector.StatelessAnalyzer, error) {
	return NewQuatileAna(qo)
}

func (qo *QuantileOpts) NewStatefulAna() (collector.StatefulAnalyzer, error) {
	return NewQuatileAna(qo)
}
