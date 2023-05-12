package pushFunc

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

var (
	targetPath = ""
	symbol     = ""
)

func TestNewPushFunc(t *testing.T) {
	testPfName := []string{"procinfo:cpuUsage", "procinfo:memUsage",
		fmt.Sprintf("UfuncCnt:%s", symbol),
	}
	testPfOpts := []*PfOpts{
		nil,
		nil,
		{targetPath, symbol},
	}
	pid := os.Getpid()
	testDuration := time.Duration(200) * time.Millisecond
	testInterval := time.Second
	for idx, name := range testPfName {
		pf, err := NewPushFunc(uint32(pid), name, testDuration, testPfOpts[idx])
		if err != nil {
			t.Fatal(err.Error())
		}

		ctx, cancel := context.WithCancel(context.Background())
		receiver := make(chan *DataPair, 5)
		result := make([]*DataPair, 0, 5)
		go pf.Push(receiver, ctx)
		go func() {
			for d := range receiver {
				result = append(result, d)
			}
		}()

		time.Sleep(testInterval)
		cancel()

		if len(result) != int(testInterval/testDuration)-1 {
			t.Error("Data not enough.")
		}
		fmt.Printf("==================%s================\n", name)
		for _, d := range result {
			fmt.Printf("value: %2f, time: %s.\n", d.Value, d.Timestamp.String())
		}
	}
}
