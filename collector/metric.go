package collector

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"wanggj.com/abyss/module"
)

type Metric interface {
	// Desc returns the descriptor for the Metric. This method returns the same descriptor
	// throughout the lifetime of the Metric. The returned descriptor is immutable. A
	// Metric ubable to describe it self must return an invalid descriptor (created
	// with NewInvalidDesc).
	Desc() *Desc
	// Write encodes the Metric into a "Metric" Protocol Buffer data transmission object.
	//
	// Metric implementations must observe concurrency safety as reads of this metric may
	// occur at the expense of total performance of rendering all registered metrics.
	//
	// While populating module.Metric, it is the responsibility of the impelmentation to
	// ensure validity of the Metric protobuf. It is recommended to sort labels
	// lexicographically. Callers of Write should still make sure of sorting if they depend
	// on it.
	Write() (*module.Metric, error)
}

// Opts bundles the options for creating most Metric types. Each metric
// implementation XXX has its own XXXOpts type, but in most cases, it is just
// an alias of this type (which might change when the requirement arises.)
//
// It is mandatory to set Name to a non-empty string. All other fields are
// optional and can safely be left at their zero value, although it is strongly
// encouraged to set a Help string.
type Opts struct {
	Name        string      `yaml:"name"`
	Help        string      `yaml:"help"`
	ConstLabels Labels      `yaml:"constLabels"`
	Level       MetricLevel `yaml:"level"`
	Priority    uint16      `yaml:"priority"`
}

type timeStampMetric struct {
	Metric
	t time.Time
}

func NewTimeStampMetric(t time.Time, m Metric) Metric {
	return &timeStampMetric{
		Metric: m,
		t:      t,
	}
}

func (t *timeStampMetric) Write() (*module.Metric, error) {
	m, err := t.Metric.Write()
	if err != nil {
		return nil, err
	}
	m.Timestamp = timestamppb.New(t.t)
	return m, nil
}
