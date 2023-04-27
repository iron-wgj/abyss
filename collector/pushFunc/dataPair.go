package pushFunc

import "time"

type DataPair struct {
	Value     float64
	Timestamp time.Time
}

func NewDataPair(v float64, t time.Time) *DataPair {
	return &DataPair{Value: v, Timestamp: t}
}
