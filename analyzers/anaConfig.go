package analyzer

import (
	"fmt"

	"wanggj.com/abyss/collector"
)

type RawMessage struct {
	unmarshal func(interface{}) error
}

func (msg *RawMessage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	msg.unmarshal = unmarshal
	return nil
}

func (msg *RawMessage) Unmarshal(v interface{}) error {
	return msg.unmarshal(v)
}

// the StateXXXAnaOpt is used to generate analyzer
type StatefulAnaOpt interface {
	NewStatefulAna() (collector.StatefulAnalyzer, error)
}

type StatelessAnaOpt interface {
	NewStatelessAna() (collector.StatelessAnalyzer, error)
}

type AnaConfig struct {
	Type string     `yaml:"type"`
	Opt  RawMessage `yaml:"opt"`
}

// GetAnaOptFromConfig is is used to parse AnaConfig into AnaOpt, which will be used
// to generate Analyzer. Every Analyzer should sign in here.
func GetAnaOptFromConfig(pid uint32, config AnaConfig) (interface{}, error) {
	var (
		result interface{}
		err    error
	)
	switch config.Type {
	// all Analyzer should get install here to allow config
	// the code is like:
	//
	// opt := anaOPt{} // correct type
	// err := config.Opt.Unmarshal(&opt)
	// if err != nil {
	// 	return nil, err
	// }
	// set PID in ConstLabels
	// return opt, nil
	case "aggregation":
		aggcfg := AggregationOpts{}
		err = config.Opt.Unmarshal(&aggcfg)
		if err != nil {
			result = nil
			break
		}
		if aggcfg.ConstLabels == nil {
			aggcfg.ConstLabels = collector.Labels{}
		}
		aggcfg.ConstLabels["PID"] = fmt.Sprint(pid)
		result, err = &aggcfg, nil
	case "quantile":
		quancfg := QuantileOpts{}
		err := config.Opt.Unmarshal(&quancfg)
		if err != nil {
			result = nil
			break
		}
		if quancfg.ConstLabels == nil {
			quancfg.ConstLabels = collector.Labels{}
		}
		quancfg.ConstLabels["PID"] = fmt.Sprint(pid)
		result, err = &quancfg, nil
	default:
		err = fmt.Errorf("Unrecongnized config type %q", config.Type)
		result = nil
	}
	return result, err
}

func GetSfaFromConfig(pid uint32, config AnaConfig) (collector.StatefulAnalyzer, error) {
	cfg, err := GetAnaOptFromConfig(pid, config)
	if err != nil {
		return nil, err
	}

	opt, ok := cfg.(StatefulAnaOpt)
	if !ok {
		return nil, fmt.Errorf("Config is not a StatefulAnaOpt: %v.", cfg)
	}

	return opt.NewStatefulAna()
}
func GetSlaFromConfig(pid uint32, config AnaConfig) (collector.StatelessAnalyzer, error) {
	cfg, err := GetAnaOptFromConfig(pid, config)
	if err != nil {
		return nil, err
	}

	opt, ok := cfg.(StatelessAnaOpt)
	if !ok {
		return nil, fmt.Errorf("Config is not a StatefulAnaOpt: %v.", cfg)
	}

	return opt.NewStatelessAna()
}
