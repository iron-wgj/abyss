package analyzer_test

import (
	"sync"
	"testing"
	"time"

	analyzer "wanggj.com/abyss/analyzers"
	"wanggj.com/abyss/collector"
)

var (
	testCfg = map[string]*analyzer.AggregationOpts{
		"max": &analyzer.AggregationOpts{
			Opts: collector.Opts{
				Name:  "maxAgg",
				Help:  "this is max",
				Level: collector.LevelInfo,
				ConstLabels: collector.Labels{
					"a": "a",
					"b": "b",
				},
			},
			Duration: time.Second * 5,
			Type:     "max",
		},
		"min": &analyzer.AggregationOpts{
			Opts: collector.Opts{
				Name: "minAgg",
				Help: "this is max",
				ConstLabels: collector.Labels{
					"a": "a",
					"b": "b",
				},
			},
			Duration: time.Second * 5,
			Type:     "min",
		},
	}

	testData   = []float64{0.1, 0.2, 0.3, 0.4}
	testResult = map[string]float64{
		"max": 0.4,
		"min": 0.1,
	}
)

func generateTestData() []*collector.DataPair {
	result := []*collector.DataPair{}
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 500)
		result = append(result, &collector.DataPair{Value: testData[i%4], Timestamp: time.Now()})
		//fmt.Printf("testData %d", i)
	}
	return result
}

func TestAggConfig(t *testing.T) {
	for tp, cfg := range testCfg {
		ana, err := cfg.NewStatelessAna()
		if err != nil {
			t.Errorf("%sAna generate error: %s.", tp, err.Error())
		}
		if ana, ok := ana.(*analyzer.Aggregation); !ok {
			t.Errorf("%s Aggregation get wrong type", tp)
		} else {
			t.Log(ana.Desc.String())
		}
	}
}
func TestAggAnalyze(t *testing.T) {
	for tp, cfg := range testCfg {
		t.Logf("============test %s==============", tp)
		sla, err := cfg.NewStatelessAna()
		ana, ok := sla.(*analyzer.Aggregation)
		if !ok {
			t.Errorf("%s Aggregation get wrong type", tp)
		}
		data := generateTestData()
		ch := make(chan collector.Metric)
		go func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func() {
				ana.Analyze(data, ch)
				wg.Done()
			}()
			ana.Analyze(data, ch)
			wg.Wait()
			close(ch)
		}()
		result, ok := <-ch
		if !ok {
			t.Errorf("%s Aggregation didn't analyze.", tp)
			return
		}
		m, err := result.Write()
		if err != nil {
			t.Fatalf("%s Aggregation generate Metric unwritable.", tp)
		}
		if m.Gauge == nil {
			t.Fatalf("%s Aggregation didn't generate Gauge.", tp)
		}
		if m.Gauge.GetValue() != testResult[tp] {
			t.Fatalf("%s Aggregation result wrong.", tp)
		}
		if _, ok := <-ch; ok {
			t.Fatalf("%s Aggregation analyze when not timeout.", tp)
		}
		t.Logf("%s Aggregation analyze result right:%g", tp, m.Gauge.GetValue())
	}
}
