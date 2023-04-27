package collector_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/collector/pushFunc"
	"wanggj.com/abyss/module"
)

type testCollectFunc struct{}

func (tc *testCollectFunc) SetDuration(d time.Duration) {}

func (tc *testCollectFunc) Push(ch chan<- *pushFunc.DataPair, ctx context.Context) {
	dur, err := time.ParseDuration("0.2s")
	if err != nil {
		return
	}
	timer := time.NewTicker(dur)
	value := float64(1)
	for {
		select {
		case t := <-timer.C:
			dp := pushFunc.NewDataPair(value, t)
			value++
			ch <- dp
		case <-ctx.Done():
			close(ch)
			fmt.Println("================== the pusher is closed =======")
			return
		}
	}
}

type testAna struct {
	desc *collector.Desc
}

func NewTestAna() *testAna {
	desc := collector.NewDesc(
		"testAna",
		"testAna",
		collector.LevelLog,
		nil,
		collector.Labels{"a": "a"},
	)
	res := &testAna{desc: desc}
	return res
}

func (ta *testAna) Describe(ch chan<- *collector.Desc) {
	ch <- ta.desc
}
func (ta *testAna) Analyze(data []*pushFunc.DataPair, ch chan<- collector.Metric) {
	for _, dp := range data {
		// TODO: write a function that generate ConstMetric, and transipoint
		// dp into ConstMetric
		cm, err := collector.NewConstMetric(
			ta.desc,
			collector.CounterValue,
			dp.Value,
		)
		if err != nil {
			fmt.Println("Analyze get an error.")
		}
		ch <- collector.NewTimeStampMetric(dp.Timestamp, cm)
	}
}

func TestPusher(t *testing.T) {
	desc := collector.NewDesc(
		"test",
		"test",
		collector.LevelInfo,
		nil,
		collector.Labels{"a": "a"},
	)
	tna := NewTestAna()
	dur, _ := time.ParseDuration("1m")
	tp := collector.NewPusher(
		desc,
		true,
		collector.GaugeValue,
		&testCollectFunc{},
		nil,
		[]collector.StatelessAnalyzer{tna},
		dur,
	)

	t.Run("Parallel test basic", func(t *testing.T) {
		t.Parallel()
		tp.Start()

		// test describe function
		t.Run("Run pb.Describe", func(t *testing.T) {
			descCh := make(chan *collector.Desc, 2)
			go func() {
				tp.Describe(descCh)
				close(descCh)
			}()
			result := make([]*collector.Desc, 0)
			for d := range descCh {
				result = append(result, d)
			}
			if result[0] != tna.desc {
				t.Errorf(
					"pb.Describe error: expected %v, get %v.",
					*(tna.desc),
					*(result[0]),
				)
			}
		})

		testCollect := func(pb *collector.Pusher) []*module.Metric {
			colCh := make(chan collector.Metric, 10)
			go func() {
				pb.Collect(colCh)
				t.Log("pb.Collect execution end.")
				close(colCh)
			}()
			result := make([]*module.Metric, 0)
			for m := range colCh {
				pm, _ := m.Write()
				result = append(result, pm)
			}
			return result
		}

		oneSecond, _ := time.ParseDuration("1s")
		time.Sleep(oneSecond)
		t.Run("Run pb.Collect first", func(t *testing.T) {
			result := testCollect(tp)
			t.Log(result)
		})
		time.Sleep(oneSecond)
		t.Run("Run pb.Collect second", func(t *testing.T) {
			result := testCollect(tp)
			t.Log(result)
		})

		tp.Stop()
		t.Log("the test end.")
	})
}

func TestPusherOOT(t *testing.T) {
	desc := collector.NewDesc(
		"test",
		"test",
		collector.LevelInfo,
		nil,
		collector.Labels{"a": "a"},
	)
	tna := NewTestAna()
	dur, _ := time.ParseDuration("1m")
	tp := collector.NewPusher(
		desc,
		false, // test Pusher.Data, dose not need the self collect
		collector.GaugeValue,
		&testCollectFunc{},
		nil,
		[]collector.StatelessAnalyzer{tna},
		dur,
	)

	t.Run("Parallel test out of time", func(t *testing.T) {
		t.Parallel()
		tp.Start()

		testCollect := func(pb *collector.Pusher) []*module.Metric {
			colCh := make(chan collector.Metric, 10)
			go func() {
				pb.Collect(colCh)
				close(colCh)
			}()
			result := make([]*module.Metric, 0)
			for m := range colCh {
				pm, _ := m.Write()
				result = append(result, pm)
			}
			return result
		}

		oneSecond, _ := time.ParseDuration("1s")
		time.Sleep(oneSecond)
		result1 := make([]*module.Metric, 0, 0)
		result2 := make([]*module.Metric, 0, 0)
		t.Run("Run pb.Collect first", func(t *testing.T) {
			result1 = testCollect(tp)
		})
		time.Sleep(time.Second * 2)
		t.Run("Run pb.Collect second", func(t *testing.T) {
			result2 = testCollect(tp)
		})

		if result1[0].Counter.Value == result2[0].Counter.Value && result1[0].Timestamp == result2[0].Timestamp {
			t.Errorf("Some metrics in result1 must clear in result2.\n\tresult1: %v\n\tresult2: %v", result1, result2)
		} else {
			t.Logf("result1: %v\n\tresult2: %v", result1, result2)
		}
		tp.Stop()
		t.Log("the test end.")
	})
}

var pusherYaml = `
desc:
  name: pusher
  help: this is a pusher
  level: 2
  constLabels:
    aaa: aaa
    bbb: bbb
selfcol: true
valuetype: 2
inv: 10s
pushFunc: procinfo:cpuUsage
pfinv: 300ms
`

func TestPusherFromYaml(t *testing.T) {
	var po collector.PusherOpts
	err := yaml.Unmarshal([]byte(pusherYaml), &po)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(po)

	pu, err := collector.NewPusherFromOpts(uint32(os.Getpid()), po, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	descChan := make(chan *collector.Desc, 5)
	mtcChan := make(chan collector.Metric, 5)

	pu.Start()

	go func() {
		pu.Describe(descChan)
		close(descChan)
	}()
	for desc := range descChan {
		fmt.Println(desc)
	}

	action := 0
	go func() {
		for i := 0; i < 10000; i++ {
			action++
		}
	}()

	time.Sleep(time.Duration(1) * time.Second)
	go func() {
		pu.Collect(mtcChan)
		close(mtcChan)
	}()

	count := 0
	for m := range mtcChan {
		count++
		mm, err := m.Write()
		if err != nil {
			t.Error(err)
		}
		fmt.Println(mm.String())
	}
	if count != 3 {
		t.Errorf("Expect %d metrics, got %d.", 2, count)
	}

	pu.Stop()
}
