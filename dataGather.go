package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/module"
)

// dataGather is used to generate and destory registry and collectors and from
// collectors and write into bytes

var TargetProc map[uint32]*ProcRegistry = map[uint32]*ProcRegistry{}

// NewProcErr is used for errors when
type NewProcErr struct {
	np  *MonitorProc
	err error
}

func (n *NewProcErr) Error() string {
	return fmt.Sprintf(
		"#New process message error#: %s,\n %v.",
		n.err.Error(),
		n.np,
	)
}

func NewProcError(np *MonitorProc, err error) error {
	return &NewProcErr{
		np:  np,
		err: err,
	}
}

func DataGather(logger *log.Logger, gatherInv time.Duration) error {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		newProcCh chan *MonitorProc
		exitCh    chan *ExitProc
		errorCh   chan error
		dataCh    chan map[int][]*module.MetricFamily
		ticker    time.Ticker
	)
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	newProcCh = make(chan *MonitorProc, 128)
	exitCh = make(chan *ExitProc)

	errorCh = make(chan error, 10)
	dataCh = make(chan map[int][]*module.MetricFamily, 10)
	ticker = *time.NewTicker(gatherInv)

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
				cfg, err := ioutil.ReadFile(n.Configpath)
				if err != nil {
					logger.Println(NewProcError(n, err))
					continue
				}
				//fmt.Println(cfg)
				proccfg := new(ProcConfig)
				err = yaml.Unmarshal(cfg, &proccfg)
				if err != nil {
					logger.Println(NewProcError(n, err))
					continue
				}
				//fmt.Println(proccfg)
				reg, err := NewProcRegFromConfig(n.Pid, proccfg)
				errs := err.(collector.MultiError)
				if len(errs) > 0 {
					logger.Println(NewProcError(n, err))
					continue
				}

				if _, ok := TargetProc[n.Pid]; ok {
					err = fmt.Errorf("Duplicated Pid: %d.", n.Pid)
					logger.Fatal(NewProcError(n, err))
					continue
				}

				TargetProc[n.Pid] = reg
				//fmt.Println(reg)

				reg.Start()
			case e := <-exitCh:
				if reg, ok := TargetProc[e.Pid]; ok {
					// TODO: destory registry
					reg.Stop()
					delete(TargetProc, e.Pid)
				}
			case <-ctx.Done():
				for _, reg := range TargetProc {
					reg.Stop()
				}
				close(dataCh)
				close(errorCh)
				return
			case <-ticker.C:
				data := map[int][]*module.MetricFamily{}
				for _, reg := range TargetProc {
					mfs, err := reg.Gather()
					if err != nil {
						errs := err.(collector.MultiError)
						if len(errs) > 0 {
							logger.Println(errors.WithStack(errs))
						}
					}

					for l, d := range mfs {
						if _, ok := data[l]; !ok {
							data[l] = []*module.MetricFamily{}
						}
						data[l] = append(data[l], d...)
					}
				}
				dataCh <- data
			}
		}
	}()

	// data Write
	go InfluxWrite(logger, dataCh)
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errorCh:
		return err
	case <-stopper:
		return nil
	}
}
