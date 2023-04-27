package pushFunc

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProcInfo(t *testing.T) {
	testFields := []string{"cpuUsage", "memUsage"}
	testInterval := time.Duration(2) * time.Second
	pfDuration := time.Duration(100) * time.Millisecond
	for _, field := range testFields {
		pf := NewProcInfo(uint32(os.Getpid()), field)
		pf.SetDuration(pfDuration)

		receiver := make(chan *DataPair, 5)
		ctx, cancel := context.WithCancel(context.Background())

		result := make([]*DataPair, 0)
		go pf.Push(receiver, ctx)
		go func() {
			for {
				select {
				case dp, ok := <-receiver:
					if !ok {
						return
					}
					result = append(result, dp)
				case <-ctx.Done():
					return
				}
			}
		}()

		time.Sleep(testInterval)
		cancel()
		if len(result) != int(testInterval/pfDuration)-1 {
			t.Logf(
				"The ProcInfo %s dose got enough data, wanted %d, got %d.",
				field,
				int(testInterval/pfDuration),
				len(result),
			)
		}

		for _, dp := range result {
			fmt.Printf(
				"value: %2f, time: %s.",
				dp.Value,
				dp.Timestamp.String(),
			)
		}
	}
}
