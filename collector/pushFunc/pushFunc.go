package pushFunc

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CollectFunc will be used to create a goroutine in Pusher, it receives a channel
// and a context.
//
// The channel is used to receive data and received data
// will be storaged into dataBuf. Each time Collect func is called,
// dataBuf will be flushed into Data and data out of time will be cleared.
//
// context is used to close goroutine, like context.WithCancel
type PushFunc interface {
	SetDuration(time.Duration)
	Push(chan<- *DataPair, context.Context)
}

// NewPushFunc need two arguments to generate a PushFunc
//
// parameters:
//
//	Name: string[type:field]
//	Duration: time.Duration[100ms-5s]
//
// return:
//
//	pointer of the pushfunc
func NewPushFunc(pid uint32, name string, duration time.Duration) (PushFunc, error) {
	fields := strings.Split(name, ":")
	if len(fields) < 2 {
		return nil, fmt.Errorf("PushFunc name must have format \"PfType:fieldName\".")
	}

	switch fields[0] {
	case "procinfo":
		if _, ok := ProcInfoFields[fields[1]]; !ok {
			return nil, fmt.Errorf(
				"ProcInfo dose not support fieldName \"%s\".",
				fields[1],
			)
		}
		pf := NewProcInfo(pid, fields[1])
		pf.SetDuration(duration)
		return pf, nil
	default:
		return nil, fmt.Errorf("PushFunc dose not support pfType \"%s\".", fields[0])
	}
}
