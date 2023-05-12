package main

import (
	"context"

	"wanggj.com/abyss/newProcTracing"
)

const (
	bpfMonitorFlag       = "-bpfMonitor"
	bpfMonitorConfigFlag = "-bpfMonConfig"
)

type MonitorProc struct {
	Pid        uint32
	Ppid       uint32
	Filename   string
	Configpath string
}

type ExitProc = newProcTracing.ExitProcMsg

// gatherMonitorProc gathers processes that with arg "-bpfMonitor" and arg like
// "-bpfMonConfig=xxx", which represent that the process want to be monitored by abyss
// and path of config file is xxx. Config file will be parsed to generate collectors
// to collect metrics.
func gatherMonitorProc(
	ctx context.Context,
	mCh chan<- *MonitorProc,
	eCh chan<- *ExitProc,
) error {
	execCh, exitCh := make(chan *newProcTracing.NewProcMsg, 10), make(chan *newProcTracing.ExitProcMsg, 10)

	obj, err := newProcTracing.LoadBpfProgram(ctx, execCh, exitCh)
	if err != nil {
		return err
	}
	defer func() {
		newProcTracing.CloseBpfObject(obj)
	}()

	for {
		select {
		case m := <-execCh:
			flag := false
			for _, arg := range m.Argv {
				if []byte(arg)[0] == 0 {
					break
				}
				str := string(arg[:len(bpfMonitorFlag)])
				if str == bpfMonitorFlag {
					flag = true
					break
				}
			}
			if !flag {
				continue
			}
			if msg := parseExecMsg(m); msg != nil {
				mCh <- msg
			}
		case m := <-exitCh:
			eCh <- m
		case <-ctx.Done():
			return nil
		}
	}
}

func parseExecMsg(msg *newProcTracing.NewProcMsg) *MonitorProc {
	if msg == nil {
		return nil
	}
	configPath := ""
	for _, strbytes := range msg.Argv {
		if strbytes[0] == 0 {
			break
		}
		idx := 0
		for idx < len(strbytes) && strbytes[idx] != 0 {
			idx++
		}
		if string(strbytes[:len(bpfMonitorConfigFlag)]) == bpfMonitorConfigFlag {
			configPath = string(strbytes[len(bpfMonitorConfigFlag)+1 : idx])
			//fmt.Println("aaa", configPath)
		}
	}
	if configPath == "" {
		return nil
	}
	return &MonitorProc{
		Pid:        msg.Pid,
		Ppid:       msg.Ppid,
		Filename:   msg.Filename,
		Configpath: configPath,
	}
}
