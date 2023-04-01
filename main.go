package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	monCh := make(chan *MonitorProc, 10)
	exitCh := make(chan *ExitProc, 10)

	wg := new(sync.WaitGroup)

	go func() {
		wg.Add(1)
		err := gatherMonitorProc(ctx, monCh, exitCh)
		if err != nil {
			fmt.Println(err)
		}
		wg.Done()
	}()
	go func() {
		for {
			select {
			case m := <-monCh:
				fmt.Printf("Monitor Proc: %+v\n", *m)
			case <-ctx.Done():
				for range monCh {
				}
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case m := <-exitCh:
				fmt.Printf("Exit Proc: %+v\n", *m)
			case <-ctx.Done():
				for range exitCh {
				}
				return
			}
		}
	}()

	time.Sleep(time.Second * 10)
	cancel()
	wg.Wait()
	close(monCh)
	close(exitCh)
}
