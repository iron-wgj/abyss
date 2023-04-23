package collector

import (
	"google.golang.org/protobuf/proto"
	"wanggj.com/abyss/module"
)

type ConstSummary struct {
	desc       *Desc
	count      uint64
	sum        float64
	objectives map[float64]float64
}

func (cs *ConstSummary) Desc() *Desc {
	return cs.desc
}

func (cs *ConstSummary) Write() (*module.Metric, error) {
	sum := &module.Summary{}
	qs := make([]*module.Quantile, 0, len(cs.objectives))

	sum.SampleCount = proto.Uint64(cs.count)
	sum.SampleSum = proto.Float64(cs.sum)

	for k, v := range cs.objectives {
		qs = append(qs, &module.Quantile{
			Quantile: proto.Float64(k),
			Value:    proto.Float64(v),
		})
	}

	sum.Quantile = qs

	return &module.Metric{
		Label:   cs.desc.constLabelPairs,
		Summary: sum,
	}, nil
}

func NewConstSummary(
	desc *Desc,
	count uint64,
	sum float64,
	objectives map[float64]float64,
) *ConstSummary {
	if desc == nil {
		return nil
	}
	return &ConstSummary{
		desc:       desc,
		count:      count,
		sum:        sum,
		objectives: objectives,
	}
}
