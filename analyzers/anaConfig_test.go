package analyzer_test

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"
	analyzer "wanggj.com/abyss/analyzers"
)

// Test all Analyzer config file resolvation

// construct a struct that contains a list of struct AnaConfig
type TestAnaConfigs struct {
	AnaConfigs []analyzer.AnaConfig `yaml:"analyzers"`
}

// the test data
var yamlAggregation = `
analyzers:
- type: "aggregation"
  opt:
    desc:
      name: max
      help: this is a simple aggregation of max
      level: 3
      constLabels:
        pid: 34567
        func: aaaa
    duration: 100ms
    type: "max"
- type: "aggregation"
  opt:
    desc:
      name: min
      help: this is a simple aggregation of min
      level: 3
      constLabels:
        pid: 2345545
        func: bbbb
    duration: 100ms
    type: "min"
`
var yamlQuantile = `
analyzers:
- type: "quantile"
  opt:
    desc:
      name: quantile_test
      help: this is a quantile analyzer test
      level: 2
      constLabels:
        pid: 2222
        name: 3333
    targets:
    - 0.5
    - 0.9
    - 0.99
- type: "quantile"
  opt:
    desc:
      name: "quantile_test"
      help: "this is a quantile analyzer test"
      level: 2
      constLabels:
        pid: 2222
        name: 3333
    targets: []
`

// Test aggregation
func TestAggregationConfig(t *testing.T) {
	var cfgs TestAnaConfigs

	err := yaml.Unmarshal([]byte(yamlAggregation), &cfgs)
	if err != nil {
		t.Fatalf("simple aggregation config unmarshal got error: %s", err.Error())
	}

	t.Logf("cfgs got %d cfg.", len(cfgs.AnaConfigs))

	for idx, cfg := range cfgs.AnaConfigs {
		ram, err := analyzer.GetAnaOptFromConfig(cfg)
		if err != nil {
			t.Errorf(
				"Got error when resolve %dth config, type is %s, error: %s",
				idx,
				cfg.Type,
				err.Error(),
			)
		}

		fmt.Println(ram)
		opt, ok := ram.(analyzer.AggregationOpts)
		if !ok {
			t.Errorf("When resolve %dth aggregation config, didn't got right type.", idx)
		}

		agg, err := analyzer.NewAggregation(&opt)
		if err != nil {
			t.Errorf("Can't get aggregation from opt, error: %s", err.Error())
		}
		t.Log(agg)
	}
}

func TestQuantileConfig(t *testing.T) {
	var cfgs TestAnaConfigs

	err := yaml.Unmarshal([]byte(yamlQuantile), &cfgs)
	if err != nil {
		t.Fatalf("simple Quantile config unmarshal got error: %s", err.Error())
	}

	t.Logf("cfgs got %d cfg.", len(cfgs.AnaConfigs))

	for idx, cfg := range cfgs.AnaConfigs {
		fmt.Println(cfg)
		ram, err := analyzer.GetAnaOptFromConfig(cfg)
		if err != nil {
			t.Errorf(
				"Got error when resolve %dth config, type is %s, error: %s",
				idx,
				cfg.Type,
				err.Error(),
			)
		}

		opt, ok := ram.(analyzer.QuantileOpts)
		if !ok {
			t.Errorf("When resolve %dth quantile config, didn't got right type.", idx)
		}
		fmt.Println(opt)

		agg, err := analyzer.NewQuatileAna(&opt)
		if err != nil {
			t.Errorf("Can't get quantile analyzer from opt, error: %s", err.Error())
		}
		t.Log(agg)
	}
}
