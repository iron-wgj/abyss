package collector

import "fmt"

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
	NewStatefulAna() (StatefulAnalyzer, error)
}

type StatelessAnaOpt interface {
	NewStatelessAna() (StatelessAnalyzer, error)
}

type AnaConfig struct {
	Type string     `yamal:"name"`
	Opt  RawMessage `yaml:"opt"`
}

// GetAnaOptFromConfig is is used to parse AnaConfig into AnaOpt, which will be used
// to generate Analyzer. Every Analyzer should sign in here.
func GetAnaOptFromConfig(config AnaConfig) (interface{}, error) {
	switch config.Type {
	// all Analyzer should get install here to allow config
	// the code is like:
	//
	// opt := anaOPt{} // correct type
	// if err != nil {
	// 	return nil, err
	// }
	// return opt, nil
	default:
		err := fmt.Errorf("Unrecongnized config type %q", config.Type)
		return nil, err
	}
}
