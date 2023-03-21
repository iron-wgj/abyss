package collector

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cespare/xxhash/v2"
	"google.golang.org/protobuf/proto"
	"wanggj.com/abyss/module"
)

// level Enum is the possible level number of metrics
type MetricLevel int

const (
	LevelFault MetricLevel = 0
	LevelError MetricLevel = 1
	LevelLog   MetricLevel = 2
	LevelInfo  MetricLevel = 3
)

// Desc is the descriptor used by every Metric. It is essentially
// the immutable meta-data of a Metric. The normal Metric implementations
// included in this package manage their Desc under the hood.
//
// The collecotr is impelmented hierarchally, for every process and metric
// belongs to it.Each has a Desc struct, which describe the static message
// of the metric. When write to bytes, metrics may add variable labels,
// such as when an error happends.
type Desc struct {
	// fqName has been built from Namespace, Subsystem, and Name.
	name string
	// help provides some helpful information about this metric.
	help string
	// constLabelPairs contains precalculated DTO label pairs based on
	// the constant labels.
	constLabelPairs []*module.LabelPair
	// variableLabels contains names of labels and normalization function
	// for which the metric maintains variable values
	variableLabels Labels
	// id is a hash of the values of the ConstLabels and Name. This
	// must be unique among all registered descriptors and can therefore be
	// used as an identifier of the descriptor.
	id uint64
	// level represents the importance of the Metric, Its possible value is among
	// MetricLevel.
	level MetricLevel
	// err is an error that occurred during construction. It is reported on
	// registration time.
	err error
}

// NewDesc return a new struct Desc. Desc contains constLabels, which is unchangable
// during collection, and variableLabels whose value is changable. The label name of
// both type of labels shoud not duplicate.
func NewDesc(name, help string, level MetricLevel, variableLabels Labels, constLabels Labels) *Desc {
	d := &Desc{
		name:           name,
		help:           help,
		variableLabels: variableLabels,
	}

	// check level is legal
	if level < 0 || level > 3 {
		d.err = fmt.Errorf("New Desc get illegal level, which is %d.", level)
	}

	// labelValues contains the label values of const labels (in order of
	// their sorted label names) plus the name.
	labelValues := make([]string, 1, len(constLabels)+1)
	labelValues[0] = name
	labelNames := make([]string, 0, len(constLabels)+len(variableLabels))
	labelNameSet := map[string]struct{}{}
	// First add only the const label names and sort them
	// if has duplicated labelNames, return err
	for labelName := range constLabels {
		labelNames = append(labelNames, labelName)
		labelNameSet[labelName] = struct{}{}
	}
	sort.Strings(labelNames)
	for _, labelName := range labelNames {
		labelValues = append(labelValues, constLabels[labelName])
	}
	// Now add variable label names, but prefix them with something that
	// cannot be in a regular label names.
	for labelName := range variableLabels {
		labelNames = append(labelNames, "$"+labelName)
		labelNameSet[labelName] = struct{}{}
	}
	if len(labelNames) != len(labelNameSet) {
		d.err = fmt.Errorf("Duplicate label names in constant and variable labels for metric %q", name)
		return d
	}

	xxh := xxhash.New()
	for _, val := range labelValues {
		xxh.WriteString(val)
	}

	d.id = xxh.Sum64()

	// wirte constLabels into data module's labelPari
	d.constLabelPairs = make([]*module.LabelPair, 0, len(constLabels))
	for n, v := range constLabels {
		d.constLabelPairs = append(d.constLabelPairs, &module.LabelPair{
			Name:  proto.String(n),
			Value: proto.String(v),
		})
	}
	return d
}

func NewInvalidDesc(err error) *Desc {
	return &Desc{
		err: err,
	}
}

func (d *Desc) String() string {
	lpStrings := make([]string, 0, len(d.constLabelPairs))
	for _, lp := range d.constLabelPairs {
		lpStrings = append(
			lpStrings,
			fmt.Sprintf("%s=%q", lp.GetName(), lp.GetValue()),
		)
	}
	return fmt.Sprintf(
		"Desc{name: %q, help %q, constLabels: {%s}, variableLabels: %v}",
		d.name,
		d.help,
		strings.Join(lpStrings, ","),
		d.variableLabels,
	)
}
