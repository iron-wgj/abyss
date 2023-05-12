package collector_test

import (
	"testing"

	"wanggj.com/abyss/collector"
)

func TestConstSummary(t *testing.T) {
	desc := collector.NewDesc(
		"cs",
		"this is test for constsummary",
		collector.LevelError,
		234,
		nil,
		collector.Labels{
			"pid":  "aaaa",
			"type": "quantile",
		},
	)
	cs := collector.NewConstSummary(
		desc,
		100,
		14234.8,
		map[float64]float64{
			0.5: 22.2,
			0.9: 343.2,
		},
	)
	if cs == nil {
		t.Fatal("Got nil when generate const summary")
	}
	t.Logf("Const summary describe: %s", cs.Desc().String())
	m, err := cs.Write()
	if err != nil {
		t.Errorf("Got error when write, error:%s", err.Error())
	}

	t.Logf("Const summary write result: %s", m.String())
}

func TestConstNullDescSummary(t *testing.T) {
	cs := collector.NewConstSummary(nil, 100, 12334, nil)
	if cs != nil {
		t.Fatalf("Expected nil, but got %v.", *cs)
	}
}
