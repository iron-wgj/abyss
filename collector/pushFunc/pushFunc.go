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

type PfOpts struct {
	// TargetPath is used by user process monitor
	TargetPath string
	// symbol specify the func to trace
	Symbol string
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
func NewPushFunc(
	pid uint32,
	name string,
	duration time.Duration,
	opt *PfOpts,
) (PushFunc, error) {
	fields := strings.Split(name, ":")
	//if len(fields) < 2 {
	//	return nil, fmt.Errorf("PushFunc name must have format \"PfType:fieldName\".")
	//}
	var (
		pf  PushFunc = nil
		err error    = nil
	)
	switch fields[0] {
	case "procinfo":
		if _, ok := ProcInfoFields[fields[1]]; !ok {
			return nil, fmt.Errorf(
				"ProcInfo dose not support fieldName \"%s\".",
				fields[1],
			)
		}
		pf = NewProcInfo(pid, fields[1])
	case "UfuncCnt":
		if len(fields) < 2 {
			pf, err = nil, fmt.Errorf("UFuncCnt PushFunc must have two fields with format \"UFuncCnt:Symbol\".")
			break
		}
		pf = NewUfuncCnt(pid, opt.TargetPath, fields[1])
	default:
		err = fmt.Errorf("PushFunc dose not support pfType \"%s\".", fields[0])
	}

	if pf != nil {
		pf.SetDuration(duration)
	}
	return pf, err
}
