package pushFunc

import (
	"context"
	"math/rand"
	"time"
)

type UfuncCnt struct {
	pid        uint32
	targetPath string
	symbol     string
	duration   time.Duration
	count      uint64
}

func NewUfuncCnt(pid uint32, targetPath, symbol string) *UfuncCnt {
	return &UfuncCnt{
		pid:        pid,
		targetPath: targetPath,
		symbol:     symbol,
	}
}

func (u *UfuncCnt) SetDuration(d time.Duration) {
	u.duration = d
}

func (u *UfuncCnt) Push(ch chan<- *DataPair, ctx context.Context) {
	ticker := time.NewTicker(u.duration)
	// TODO: attach uprobe to the target binary file
	for {
		select {
		case <-ticker.C:
			u.count += uint64(rand.Uint32()) % 5
			ch <- &DataPair{
				Value:     float64(u.count),
				Timestamp: time.Now(),
			}
		case <-ctx.Done():
			close(ch)
			return
		}
	}
}
