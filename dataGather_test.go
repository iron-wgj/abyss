package main

import (
	"log"
	"os"
	"testing"
	"time"
)

func TestDataGather(t *testing.T) {
	logger := log.New(os.Stdout, "[Test] ", log.Ltime|log.Lshortfile)
	err := DataGather(logger, time.Second*2)
	if err != nil {
		t.Error(err)
	}
}
