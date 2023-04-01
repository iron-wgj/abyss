package analyzer

import (
	"fmt"

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
