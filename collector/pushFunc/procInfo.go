package pushFunc

import (
	"context"
	"fmt"
	"time"

	glog "github.com/golang/glog"
	"github.com/shirou/gopsutil/v3/process"
)

var ProcInfoFields = map[string]struct{}{
	"cpuUsage": struct{}{},
	"memUsage": struct{}{},
}

func (p *ProcInfo) getProcStat() (float64, error) {
	exists, err := process.PidExists(int32(p.pid))
	if !exists {
		return 0, fmt.Errorf("Pid %d dose not exists.", p.pid)
	}
	if err != nil {
		return 0, err
	}

	proc, err := process.NewProcess(int32(p.pid))
	if err != nil {
		return 0, err
	}
	switch p.field {
	case "cpuUsage":
		usage, err := proc.CPUPercent()
		if err != nil {
			return 0, err
		}
		return float64(usage), nil
	case "memUsage":
		usage, err := proc.MemoryPercent()
		if err != nil {
			return 0, err
		}
		return float64(usage), nil
	default:
		return 0, fmt.Errorf("Unsported field for ProcInfo")
	}
}

type ProcInfo struct {
	pid      uint32
	duration time.Duration
	field    string
}

func NewProcInfo(pid uint32, field string) *ProcInfo {
	return &ProcInfo{
		pid:   pid,
		field: field,
	}
}

func (p *ProcInfo) SetDuration(d time.Duration) {
	p.duration = d
}

func (p *ProcInfo) Push(ch chan<- *DataPair, ctx context.Context) {
	ticker := time.NewTicker(p.duration)
	for {
		select {
		case <-ticker.C:
			value, err := p.getProcStat()
			if err != nil {
				glog.Error(err)
			} else {
				ch <- &DataPair{
					Value:     value,
					Timestamp: time.Now(),
				}
			}
		case <-ctx.Done():
			close(ch)
			return
		}

	}
}
