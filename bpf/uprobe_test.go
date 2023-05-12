package bpf

import (
	"encoding/binary"
	"fmt"
	"os/exec"
	"testing"
	"time"

	ebh "github.com/DataDog/ebpfbench"
	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/aquasecurity/libbpfgo/helpers"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go userFuncCount userFuncCount.bpf.c -- -I../include

var (
	binaryPath = "./test/test"
	symbol     = "main.uprobeTarget"
)

func BenchmarkUprobe(b *testing.B) {
	eb := ebh.NewEBPFBenchmark(b)
	defer eb.Close()

	// setup ebpf kprobe
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
	_, err = prog.AttachUprobe(-1, binaryPath, offset)
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

	// profile the ebpf program
	eb.ProfileProgram(prog.GetFd(), "userFuncCount")
	eb.Run(func(b *testing.B) {
		cmd := exec.Command(binaryPath)
		err = cmd.Start()
		if err != nil {
			fmt.Println("can't start test binary.")
			return
		}
		defer func() {
			cmd.Process.Kill()
			stopChannel <- true
		}()
		time.Sleep(time.Second * 5)
	})
}

func BenchmarkUprobeFuncDur(b *testing.B) {
	eb := ebh.NewEBPFBenchmark(b)
	defer eb.Close()

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
	_, err = fentry.AttachUprobe(-1, binaryPath, offset)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = fexit.AttachURetprobe(-1, binaryPath, offset)
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

	eb.ProfileProgram(fentry.GetFd(), "uprobeDurEntry")
	eb.ProfileProgram(fexit.GetFd(), "uprobeDurExit")
	eb.Run(func(b *testing.B) {
		cmd := exec.Command(binaryPath)
		err = cmd.Start()
		if err != nil {
			fmt.Println("can't start test binary.")
			return
		}
		defer func() {
			time.Sleep(time.Duration(5) * time.Second)
			out, err := cmd.Output()
			if err == nil {
				fmt.Println(string(out))
			}
			stopChannel <- true
			cmd.Process.Kill()
		}()

	})
	stopChannel <- false
}

//func TestFuncCountBpf2go(t *testing.T){
//	stopper := make(chan os.Signal, 1)
//	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)
//
//	if err := rlimit.RemoveMemlock(); err != nil {
//		t.Error(err)
//	}
//
//	// load compiled programs and maps into the kernel
//	obj := userFuncCountObjects{}
//	if err := loadUserFuncCountObjects(&obj, nil); err != nil {
//		t.Error(err)
//	}
//
//	// Open target binary file and read its symbol
//	ex, err := link.OpenExecutable(binaryPath)
//	if err != nil {
//		t.Error(err)
//	}
//
//	// attach bpf program to uprobe
//	up, err := ex.Uprobe(symbol, obj.UprobeFuncCall, &link.UprobeOptions{PID: pid})
//	if err != nil {
//		t.Error(err)
//	}
//	defer up.Close()
//
//	rd, err := perf.NewReader(obj.Events, os.Getpagesize())
//	if err != nil {
//		t.Error(err)
//	}
//	defer rd.Close()
//
//	go func() {
//		<-stopper
//		log.Println("Received signal, exiting program..")
//
//		if err := rd.Close(); err != nil {
//			log.Fatalf("closing perf event reader: %s", err)
//		}
//	}()
//
//	t.Log("Listening to events...")
//
//	var event userFuncCountMaps
