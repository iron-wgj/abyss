package analyzer

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/module"
)

type AlertOp int

const (
	NoneOp AlertOp = iota
	Bigger
	Smaller
)

var Str2Op = map[string]AlertOp{
	"none":    NoneOp,
	"bigger":  Bigger,
	"smaller": Smaller,
}

// Alert is a Metric of type Event, which is used to alert
// Metric that dosen't fullfill the alert rule
type Alert struct {
	desc         *collector.Desc
	op           AlertOp
	compareValue float64
	metricValue  float64
	timestamp    time.Time
	mtx          sync.Mutex
}

// use opt form analyzer and labels specified by analyzer,
// str must be format OP:compareValue:Level
func NewAlertFromStr(
	mopt *collector.Opts,
	labels collector.Labels,
	str string,
) (*Alert, error) {
	if str == "none" || str == "" {
		return nil, nil
	}

	name := mopt.Name + "(alert)"
	help := fmt.Sprintf("Alert of metric %s.", mopt.Name)
	constLabel := make(collector.Labels)
	for k, v := range mopt.ConstLabels {
		constLabel[k] = v
	}
	for k, v := range labels {
		constLabel[k] = v
	}
	constLabel["rules"] = str

	cfg := strings.Split(str, ":")
	if len(cfg) != 3 {
		return nil, errors.WithStack(
			fmt.Errorf("Alert config must have 3 fields, got %d", len(cfg)),
		)
	}
	op, ok := Str2Op[cfg[0]]
	if !ok {
		return nil, errors.WithStack(
			fmt.Errorf("Generate Alert error: unsupported op %s.", cfg[0]),
		)
	}
	cv, err := strconv.ParseFloat(cfg[1], 64)
	if err != nil {
		return nil, errors.WithStack(
			fmt.Errorf("Generate Alert error: %s.", err.Error()),
		)
	}
	level, err := strconv.Atoi(cfg[2])
	if err != nil || level < 1 || level > 4 {
		return nil, errors.WithStack(
			fmt.Errorf(
				"Generate Alert error: level must between 0-4, got %s.",
				cfg[2],
			),
		)
	}
	desc := collector.NewDesc(
		name,
		help,
		collector.MetricLevel(level),
		mopt.Priority,
		nil,
		constLabel,
	)

	return &Alert{
		desc:         desc,
		compareValue: cv,
		op:           op,
	}, nil
}

// compare is used to compare CompareValue and metric value
// if result is true, Alert should be collected
func (a *Alert) compare(value float64, tp time.Time) bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.metricValue = value
	a.timestamp = tp
	switch a.op {
	case Bigger:
		return value > a.compareValue
	case Smaller:
		return value < a.compareValue
	default:
		return false
	}
}

func (a *Alert) Desc() *collector.Desc {
	return a.desc
}

func (a *Alert) Write() (*module.Metric, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	result := &module.Metric{}
	result.Event = &module.Event{
		Value:     &a.metricValue,
		Timestamp: timestamppb.New(a.timestamp),
	}
	result.Label = collector.MakeLabelPairs(a.desc)
	result.Priority = proto.Uint32(a.desc.GetPriority())
	return result, nil
}
