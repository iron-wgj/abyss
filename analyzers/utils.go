package analyzer

import (
	"fmt"
	"time"

	"wanggj.com/abyss/collector"
)

func checkOptLabels(labels collector.Labels, illegalNames []string) error {
	for _, name := range illegalNames {
		if _, ok := labels[name]; ok {
			return fmt.Errorf("Label name \"%s\" is illegal.", name)
		}
	}
	return nil
}

func newConstTimeMetric(d *collector.Desc, value float64) (collector.Metric, error) {
	cm, err := collector.NewConstMetric(
		d,
		collector.GaugeValue,
		value,
	)
	if err != nil {
		return nil, err
	}
	return collector.NewTimeStampMetric(time.Now(), cm), nil
}
