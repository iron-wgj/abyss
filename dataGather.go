package main

import (
	"context"

	"wanggj.com/abyss/collector"
)

// dataGather is used to generate and destory registry and collectors and from
// collectors and write into bytes

var ProcRegistry map[uint32]collector.Registry

func DataGather() error {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		newProcCh chan *MonitorProc
		exitCh    chan *ExitProc
		errorCh   chan error
	)
	ctx, cancel = context.WithCancel(context.Background())
	newProcCh = make(chan *MonitorProc, 128)
	exitCh = make(chan *ExitProc)

	go func() {
		if err := gatherMonitorProc(ctx, newProcCh, exitCh); err != nil {
			errorCh <- err
		}
	}()
	go func() {
		for {
			select {
			case n := <-newProcCh:
				// TODO: creat new registry and add it into ProcRegistry
			case e := <-exitCh:
				if _, ok := ProcRegistry[e.Pid]; ok {
					// TODO: destory registry
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case err := <-errorCh:
		return err
	}

	cancel()
	return nil
}

type ProcessConfig struct {
}
