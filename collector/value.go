package collector

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"wanggj.com/abyss/module"
)

type ValueType int

// Possible values for the ValueType enum.
const (
	_ ValueType = iota
	CounterValue
	GaugeValue
)

var (
	CounterMetricTypePtr = func() *module.MetricType {
		d := module.MetricType_COUNTER
		return &d
	}()
	GaugeMetricTypePtr = func() *module.MetricType {
		d := module.MetricType_GAUGE
		return &d
	}()
)

func (v ValueType) ToModule() *module.MetricType {
	switch v {
	case CounterValue:
		return CounterMetricTypePtr
	case GaugeValue:
		return GaugeMetricTypePtr
	default:
		return nil
	}
}

func populateMetric(
	t ValueType,
	v float64,
	labelPairs []*module.LabelPair,
	m *module.Metric,
) error {
	m.Label = labelPairs
	switch t {
	case CounterValue:
		m.Counter = &module.Counter{Value: proto.Float64(v)}
	case GaugeValue:
		m.Gauge = &module.Gauge{Value: proto.Float64(v)}
	default:
		return fmt.Errorf("func populateMetric encountered unknow type %v", t)
	}
	return nil
}

// NewConstMetric returns a metric with one fixed value that cannot be changed.
// When implementing some Collectors, it is useful as a throw-asay metric that
// is generated on the fly to send it to Registry in the Collect method.
// NewConstMEtric returns an error if Desc is invalid.
func NewConstMetric(
	desc *Desc,
	valueType ValueType,
	value float64,
) (Metric, error) {
	if desc.err != nil {
		return nil, desc.err
	}

	metric := &module.Metric{}
	if err := populateMetric(
		valueType,
		value,
		MakeLabelPairs(desc),
		metric,
	); err != nil {
		return nil, err
	}
	return &ConstMetric{
		desc:   desc,
		metric: metric,
	}, nil
}

// ConstMetric is used to generate a fixed metric that cannot changed
type ConstMetric struct {
	desc   *Desc
	metric *module.Metric
}

func (c *ConstMetric) Desc() *Desc {
	return c.desc
}
func (c *ConstMetric) Write() (*module.Metric, error) {
	return c.metric, nil
}

// MakeLabelPairs now is used to copy constLabelPairs in desc, it may
// be used to involve variable labelpairs in the future
func MakeLabelPairs(desc *Desc) []*module.LabelPair {
	result := make([]*module.LabelPair, 0, len(desc.constLabelPairs))
	result = append(result, desc.constLabelPairs...)
	return result
}
