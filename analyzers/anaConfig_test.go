package analyzer_test

import (
	"fmt"
	"reflect"
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
      priority: 222
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
      priority: 222
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
      priority: 222
      constLabels:
        pid: 2222
        name: 3333
    targets:
      0.5: "bigger:9.8:3"
      0.9: "bigger:9.8:3"
      0.99: "bigger:9.8:3"
- type: "quantile"
  opt:
    desc:
      name: "quantile_test"
      help: "this is a quantile analyzer test"
      level: 2
      priority: 222
      constLabels:
        pid: 2222
        name: 3333
    targets:
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
		ram, err := analyzer.GetAnaOptFromConfig(111, cfg)
		if err != nil {
			t.Errorf(
				"Got error when resolve %dth config, type is %s, error: %s",
				idx,
				cfg.Type,
				err.Error(),
			)
		}

		fmt.Println(ram)
		opt, ok := ram.(*analyzer.AggregationOpts)
		if !ok {
			t.Errorf("When resolve %dth aggregation config, didn't got right type: %v.", idx, reflect.TypeOf(ram))
		}

		agg, err := analyzer.NewAggregation(opt)
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
		ram, err := analyzer.GetAnaOptFromConfig(111, cfg)
		if err != nil {
			t.Errorf(
				"Got error when resolve %dth config, type is %s, error: %s",
				idx,
				cfg.Type,
				err.Error(),
			)
		}

		opt, ok := ram.(*analyzer.QuantileOpts)
		if !ok {
			t.Errorf("When resolve %dth quantile config, didn't got right type.", idx)
		}
		fmt.Println(opt)

		agg, err := analyzer.NewQuatileAna(opt)
		if err != nil {
			t.Errorf("Can't get quantile analyzer from opt, error: %s", err.Error())
		}
		t.Log(agg)
	}
}

func TestStatefulAnaGen(t *testing.T) {
	var cfgs TestAnaConfigs
	var tmpcfgs TestAnaConfigs

	testConfigs := []string{yamlQuantile}
	for _, str := range testConfigs {
		err := yaml.Unmarshal([]byte(str), &tmpcfgs)
		if err != nil {
			t.Fatalf("simple Quantile config unmarshal got error: %s", err.Error())
		}
		cfgs.AnaConfigs = append(cfgs.AnaConfigs, tmpcfgs.AnaConfigs...)
	}

	t.Logf("cfgs got %d cfg.", len(cfgs.AnaConfigs))

	for idx, cfg := range cfgs.AnaConfigs {
		fmt.Printf("================= %dth config =========\n", idx+1)
		ram, err := analyzer.GetAnaOptFromConfig(111, cfg)
		if err != nil {
			t.Errorf(
				"Got error when resolve %dth config, type is %s, error: %s",
				idx,
				cfg.Type,
				err.Error(),
			)
		}
		t.Log(ram)

		// sfat and ramt can be used to make sure that ram implements StatefulAnaOpt
		//sfat := reflect.TypeOf((*analyzer.StatefulAnaOpt)(nil)).Elem()
		//ramt := reflect.TypeOf(ram)
		//fmt.Println(ramt)
		//fmt.Println(sfat)
		//fmt.Println(ramt.ConvertibleTo(sfat))
		//fmt.Println(ramt.Implements(sfat))
		opt, ok := ram.(analyzer.StatefulAnaOpt)
		if !ok {
			t.Errorf("When resolve %dth quantile config, didn't got right type.", idx+1)
		}
		fmt.Println(opt)

		agg, err := opt.NewStatefulAna()
		if err != nil {
			t.Errorf("Can't get stateful analyzer from opt, error: %s", err.Error())
		}
		t.Log(agg)
	}
}

// test stateless analyzer generation from configs
func TestStatelessAnaGen(t *testing.T) {
	var cfgs TestAnaConfigs
	var tmpcfgs TestAnaConfigs

	testConfigs := []string{yamlQuantile, yamlAggregation}
	for _, str := range testConfigs {
		err := yaml.Unmarshal([]byte(str), &tmpcfgs)
		if err != nil {
			t.Fatalf("simple Quantile config unmarshal got error: %s", err.Error())
		}
		cfgs.AnaConfigs = append(cfgs.AnaConfigs, tmpcfgs.AnaConfigs...)
	}

	t.Logf("cfgs got %d cfg.", len(cfgs.AnaConfigs))

	for idx, cfg := range cfgs.AnaConfigs {
		fmt.Printf("================= %dth config =========\n", idx+1)
		ram, err := analyzer.GetAnaOptFromConfig(111, cfg)
		if err != nil {
			t.Errorf(
				"Got error when resolve %dth config, type is %s, error: %s",
				idx,
				cfg.Type,
				err.Error(),
			)
		}
		t.Log(ram)

		// sfat and ramt can be used to make sure that ram implements StatefulAnaOpt
		//sfat := reflect.TypeOf((*analyzer.StatefulAnaOpt)(nil)).Elem()
		//ramt := reflect.TypeOf(ram)
		//fmt.Println(ramt)
		//fmt.Println(sfat)
		//fmt.Println(ramt.ConvertibleTo(sfat))
		//fmt.Println(ramt.Implements(sfat))
		opt, ok := ram.(analyzer.StatelessAnaOpt)
		if !ok {
			t.Errorf("When resolve %dth quantile config, didn't got right type.", idx+1)
		}
		fmt.Println(opt)

		agg, err := opt.NewStatelessAna()
		if err != nil {
			t.Errorf("Can't get stateful analyzer from opt, error: %s", err.Error())
		}
		t.Log(agg)
	}
}
