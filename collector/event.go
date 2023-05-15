package collector

import "time"

type Event interface {
	Metric
	Collector

	// Set set value the event keep
	Set(float64)

	// SetTime set timestamp when event happend
	SetTime(time.Time)
}
