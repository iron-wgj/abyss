package internal

import "wanggj.com/abyss/module"

func NormalizeMetricFamilies(
	metricFamiliesByName map[string]*module.MetricFamily,
) map[int][]*module.MetricFamily {
	result := make(map[int][]*module.MetricFamily)
	result[0] = make([]*module.MetricFamily, 0, len(metricFamiliesByName))
	for _, mfs := range metricFamiliesByName {
		result[0] = append(result[0], mfs)
	}
	return result
}
