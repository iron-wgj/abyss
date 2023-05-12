package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/module"
)

var pusherYaml = `
pusher:
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
slana:
- type: "aggregation"
  opt:
    desc:
      name: max
      help: this is a simple aggregation of max
      level: 3
      constLabels:
    duration: 3s
    type: "max"
sfana:
- type: "quantile"
  opt:
    desc:
      name: quantile_test
      help: this is a quantile analyzer test
      level: 2
      constLabels:
    targets:
    - 0.5
    - 0.9
    - 0.99
`

func TestNewPusherFromConfig(t *testing.T) {
	var pucfg PusherConfig
	err := yaml.Unmarshal([]byte(pusherYaml), &pucfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("stateful analyzer: %v", pucfg.SfAna)

	p, err := NewPusherFromConfig(111, &pucfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(p)
	t.Log(p.StatefulAna)
	t.Log(p.StatelessAna)
}

func TestPusherWithAnalyzer(t *testing.T) {
	var pucfg PusherConfig
	err := yaml.Unmarshal([]byte(pusherYaml), &pucfg)
	if err != nil {
		t.Fatal(err)
	}

	p, err := NewPusherFromConfig(uint32(os.Getpid()), &pucfg)
	if err != nil {
		t.Fatal(err)
	}

	descChan := make(chan *collector.Desc, 5)
	mtcChan := make(chan collector.Metric, 5)

	p.Start()

	go func() {
		p.Describe(descChan)
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

	time.Sleep(time.Duration(3) * time.Second)
	go func() {
		p.Collect(mtcChan)
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
	p.Stop()
}

var regyaml = `
pushercfg:
- pusher:
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
  slana:
  - type: "aggregation"
    opt:
      desc:
        name: max
        help: this is a simple aggregation of max
        level: 3
        constLabels:
      duration: 3s
      type: "max"
  sfana:
  - type: "quantile"
    opt:
      desc:
        name: quantile_test
        help: this is a quantile analyzer test
        level: 2
        constLabels:
      targets:
      - 0.5
      - 0.9
      - 0.99
`

func printMetricFamily(mfs []*module.MetricFamily) {
	for _, mf := range mfs {
		for _, m := range mf.Metric {
			fmt.Println(m)
		}
	}
}

// test ProcReg generate、 Registry and Gather
func TestProcReg(t *testing.T) {
	var regCfg ProcConfig
	err := yaml.Unmarshal([]byte(regyaml), &regCfg)
	if err != nil {
		t.Fatal(err)
	}

	reg, err := NewProcRegFromConfig(uint32(os.Getpid()), &regCfg)
	errs := err.(collector.MultiError)
	if len(errs) > 0 {
		t.Fatal(err)
	}

	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	reg.Start()
	defer reg.Stop()

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-stopper:
			return
		case <-ticker.C:
			mf, err := reg.Gather()
			errs = err.(collector.MultiError)
			if len(errs) > 0 {
				t.Error(err.Error())
				return
			}
			printMetricFamily(mf[0])
		}
	}

}
