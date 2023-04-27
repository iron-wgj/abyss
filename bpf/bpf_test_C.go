package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"time"

	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/aquasecurity/libbpfgo/helpers"
)

var userFuncCountBPF = "./userFuncCount.bpf.o"
var userFuncExecTimeBPF = "./userFuncExecTime.bpf.o"
var targetBinary = "./test/test"

func testFuncCount(binaryPath, symbol string, dur time.Duration) {
	fmt.Println("Start test the func count.")
	module, err := bpf.NewModuleFromFile(userFuncCountBPF)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer module.Close()

	if err = module.BPFLoadObject(); err != nil {
		fmt.Println(err)
		return
	}

	prog, err := module.GetProgram("uprobe__func_call")
	if err != nil {
		fmt.Println(err)
		return
	}

	offset, err := helpers.SymbolToOffset(
		binaryPath,
		symbol,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	cmd := exec.Command(binaryPath)
	err = cmd.Start()
	if err != nil {
		fmt.Println("can't start test binary.")
		return
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	_, err = prog.AttachUprobe(pid, binaryPath, offset)
	if err != nil {
		fmt.Println(err)
		return
	}

	eventsChannel := make(chan []byte)
	lostChannel := make(chan uint64)
	pb, err := module.InitPerfBuf("events", eventsChannel, lostChannel, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	pb.Start()
	defer func() {
		pb.Stop()
		pb.Close()
	}()

	stopChannel := make(chan bool)
	go func() {
		for {
			select {
			case e := <-eventsChannel:
				pid := binary.LittleEndian.Uint32(e[0:4])
				tgid := binary.LittleEndian.Uint32(e[4:8])
				ts := binary.LittleEndian.Uint64(e[8:16])
				fmt.Printf(
					"Got a func call, pid:%d, tgid:%d, ts:%d.\n",
					pid, tgid, ts,
				)
			case e := <-lostChannel:
				fmt.Printf("lost %d events.\n", e)
			case <-stopChannel:
				return
			}
		}
	}()

	time.Sleep(dur)
	stopChannel <- false
}

func testFuncDuration(binaryPath, symbol string, dur time.Duration) {
	fmt.Println("Start test the func count.")
	module, err := bpf.NewModuleFromFile(userFuncExecTimeBPF)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer module.Close()

	fret, err := module.GetMap("func_ret")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("max entries: ", fret.GetMaxEntries())
	if err = fret.Resize(32); err != nil {
		fmt.Println(err)
		return
	}

	if err = module.BPFLoadObject(); err != nil {
		fmt.Println(err)
		return
	}

	fentry, err := module.GetProgram("uprobe__func_entry")
	if err != nil {
		fmt.Println(err)
		return
	}
	fexit, err := module.GetProgram("uprobe__func_exit")
	if err != nil {
		fmt.Println(err)
		return
	}

	offset, err := helpers.SymbolToOffset(
		binaryPath,
		symbol,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	eventsChannel := make(chan []byte, 10)
	lostChannel := make(chan uint64)
	pb, err := module.InitPerfBuf("func_ret", eventsChannel, lostChannel, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	pb.Start()
	defer func() {
		pb.Stop()
		pb.Close()
	}()

	cmd := exec.Command(binaryPath)
	err = cmd.Start()
	if err != nil {
		fmt.Println("can't start test binary.")
		return
	}
	defer func() {
		out, err := cmd.Output()
		if err == nil {
			fmt.Println(string(out))
		}
		cmd.Process.Kill()
	}()

	pid := cmd.Process.Pid
	_, err = fentry.AttachUprobe(pid, binaryPath, offset)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = fexit.AttachURetprobe(pid, binaryPath, offset)
	if err != nil {
		fmt.Println(err)
		return
	}

	stopChannel := make(chan bool)
	go func() {
		for {
			select {
			case e := <-eventsChannel:
				pid := binary.LittleEndian.Uint32(e[0:4])
				tgid := binary.LittleEndian.Uint32(e[4:8])
				ts := binary.LittleEndian.Uint64(e[8:16])
				fmt.Printf(
					"Got a func call, pid:%d, tgid:%d, duration in ns:%d.\n",
					pid, tgid, ts,
				)
			case e := <-lostChannel:
				fmt.Printf("lost %d events.\n", e)
			case <-stopChannel:
				return
			}
		}
	}()

	time.Sleep(dur)
	stopChannel <- false
}

func main() {
	binaryPath, symbol := targetBinary, "main.uprobeTarget"
	duration := "2s"
	monitorType := "count"
	if len(os.Args) >= 2 && os.Args[1] == "duration" {
		monitorType = "duration"
	}
	if len(os.Args) >= 3 {
		duration = os.Args[2]
	}
	if len(os.Args) >= 5 {
		binaryPath, symbol = os.Args[3], os.Args[4]
	}

	dur, err := time.ParseDuration(duration)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch monitorType {
	case "count":
		testFuncCount(binaryPath, symbol, dur)
	case "duration":
		testFuncDuration(binaryPath, symbol, dur)
	default:
		panic(fmt.Errorf("Monitor type %s is unsupported.", monitorType))
	}
}
