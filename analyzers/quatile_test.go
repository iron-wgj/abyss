package analyzer_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	analyzer "wanggj.com/abyss/analyzers"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/collector/pushFunc"
)

var (
	testOpt = &analyzer.QuantileOpts{
		Opts: collector.Opts{
			Name:  "quatile",
			Help:  "this is quantile",
			Level: collector.LevelInfo,
			ConstLabels: collector.Labels{
				"a": "a",
				"b": "b",
			},
		},
		Ranks: map[float64]string{
			0.9: "bigger:0.1:3",
		},
	}
)

func generateRandomData(min, max float64, length int) []*pushFunc.DataPair {
	min, max = math.Abs(min), math.Abs(max)
	if min > max {
		min, max = max, min
	}
	result := make([]*pushFunc.DataPair, length, length)
	for i := 0; i < length; i++ {
		rf := min + rand.Float64()*(max-min)
		result[i] = &pushFunc.DataPair{
			Value:     rf,
			Timestamp: time.Now(),
		}
	}
	return result
}

func TestQuantile(t *testing.T) {
	//sfat := reflect.TypeOf((*analyzer.StatefulAnaOpt)(nil)).Elem()
	//fmt.Println(reflect.TypeOf(testOpt).ConvertibleTo(sfat))
	slg, err := testOpt.NewStatelessAna()
	if err != nil {
		t.Fatal("Cannot generate quantile.")
	}
	slana, ok := slg.(*analyzer.QuantileAnalyzer)
	if !ok {
		t.Fatal("Analyzer is not the Quantile.")
	}
	if len(slana.Ranks) != len(testOpt.Ranks) {
		t.Fatalf("Stateless Quantile expected %d targets, got %d.", len(testOpt.Ranks), len(slana.Ranks))
	}

	sfg, err := testOpt.NewStatefulAna()
	if err != nil {
		t.Fatal("Cannot generate quantile.")
	}
	sfana, ok := sfg.(*analyzer.QuantileAnalyzer)
	if !ok {
		t.Fatal("Analyzer is not the Quantile.")
	}

	testData := generateRandomData(0, 10, 20)
	t.Logf("test data: %v", testData)
	for _, d := range testData {
		sfana.Observe(d)
	}

	slch := make(chan collector.Metric, 10)
	sfch := make(chan collector.Metric, 10)
	sfMetric := make([]collector.Metric, 0)
	slMetric := make([]collector.Metric, 0)
	go func() {
		slana.Analyze(testData, slch)
		close(slch)
		sfana.Collect(sfch)
		close(sfch)
	}()
	for m := range slch {
		slMetric = append(slMetric, m)
	}
	for m := range sfch {
		sfMetric = append(sfMetric, m)
	}

	if len(slMetric) != 1 {
		t.Errorf("Stateless Quantile expect %d results, got %d.", len(testOpt.Ranks), 1)
	}
	if len(sfMetric) != 1 {
		t.Errorf("Stateful Quantile expect %d results, got %d.", len(testOpt.Ranks), 1)
	}

	for _, m := range slMetric {
		_, ok := m.(*collector.ConstSummary)
		md, err := m.Write()
		if err != nil {
			t.Errorf("Metric cannot Write: %s.", err.Error())
		}
		if ok && md.Summary == nil {
			t.Fatal("Want Summary but Metric's Summary is null.")
		}
		if ok && len(md.Summary.Quantile) != len(testOpt.Ranks) {
			t.Errorf("Not enough quantiles, expected %d, got %d.", len(md.Summary.Quantile), len(testOpt.Ranks))
		}
		fmt.Println(md.String())
	}
	for _, m := range sfMetric {
		_, ok := m.(*collector.ConstSummary)
		md, err := m.Write()
		if err != nil {
			t.Errorf("Metric cannot Write: %s.", err.Error())
		}
		if ok && md.Summary == nil {
			t.Fatal("Want Summary but Metric's Summary is null.")
		}
		if ok && len(md.Summary.Quantile) != len(testOpt.Ranks) {
			t.Errorf("Not enough quantiles, expected %d, got %d.", len(md.Summary.Quantile), len(testOpt.Ranks))
		}
		fmt.Println(md.String())
	}
}
