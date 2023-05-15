package analyzer

import (
	"fmt"
	"testing"
	"time"

	"wanggj.com/abyss/collector"
)

func TestNewAlertFromStr(t *testing.T) {
	desc := collector.Opts{
		Name: "test",
		Help: "this is help",
		ConstLabels: collector.Labels{
			"aaa": "aaa",
			"PID": "222",
		},
		Level:    collector.LevelInfo,
		Priority: 222,
	}

	labels := collector.Labels{
		"quantile_rank": "0.8",
	}

	cfgs := []string{
		"bigger:6.8:3",
		"bigg:5.8:3",
		"smaller:aaa:4",
		"smaller:8.7:8",
	}
	succ := []bool{true, false, false, false}

	for k, cfg := range cfgs {
		fmt.Printf("============ cfg %d ==========\n", k+1)
		a, err := NewAlertFromStr(&desc, labels, cfg)
		if (succ[k] && err != nil) || (!succ[k] && err == nil) {
			t.Errorf("Expected error: %t, got %v.", succ[k], err)
		}
		t.Log(a)
		if a != nil {
			t.Log(a.Desc().String())
		}
	}

}

func TestAlertComp(t *testing.T) {
	desc := collector.Opts{
		Name: "test",
		Help: "this is help",
		ConstLabels: collector.Labels{
			"aaa": "aaa",
			"PID": "222",
		},
		Level:    collector.LevelInfo,
		Priority: 222,
	}

	labels := collector.Labels{
		"quantile_rank": "0.8",
	}

	cfgs := []string{
		"bigger:6.8:3",
		"smaller:8.7:4",
	}
	for k, cfg := range cfgs {
		fmt.Printf("============ cfg %d ==========\n", k+1)
		a, err := NewAlertFromStr(&desc, labels, cfg)
		if err != nil {
			t.Error(err)
		}
		t.Log(a.compare(22.8, time.Now()))
		t.Log(a.metricValue)
	}
}
