package newProcTracing

import (
	"context"
	"encoding/binary"
	"fmt"

	bpf "github.com/aquasecurity/libbpfgo"
	glog "github.com/golang/glog"
	errors "github.com/pkg/errors"
)

const (
	// const variant for  execve message
	MaxFileNameLen int = 127
	MaxArgvNum     int = 19
	MaxArgvLen     int = 127
	NewProcMsgSize int = 8 + MaxFileNameLen + MaxArgvLen*MaxArgvNum

	// const variant for exit message
	ExitProcMsgSize int = 4 * 3
)

// message received from eBPF program attached to
// tracepoint sys_enter_execve, must sync to
// struct process in newProcess.h
type NewProcMsg struct {
	Pid      uint32
	Ppid     uint32
	Filename string
	Argv     []string
}

type ExitProcMsg struct {
	Pid       uint32
	Ppid      uint32
	ErrorCode int
}

// object loaded from ePBF program, keeped unitl process exit
type NewProcBPFObjs struct {
	module   *bpf.Module
	programs []*bpf.BPFProg
	bpfMaps  []*bpf.RingBuffer
}

// decode bytes into struct NewProcMsg, len of bytes must equal to NewProcMsgSize
func DecodeToNewProcMsg(msg []byte) (*NewProcMsg, error) {
	if len(msg) < NewProcMsgSize {
		return nil, fmt.Errorf("New process message can not fit into struct NewProcMsg, size of message is %d, expect %d", len(msg), NewProcMsgSize)
	}
	res := &NewProcMsg{
		Pid:      binary.LittleEndian.Uint32(msg[:4]),
		Ppid:     binary.LittleEndian.Uint32(msg[4:8]),
		Filename: string(msg[8 : 8+MaxFileNameLen]),
		Argv:     make([]string, MaxArgvNum),
	}
	for i := 0; i < MaxArgvNum; i++ {
		startOff := 8 + MaxFileNameLen + i*MaxArgvLen
		res.Argv[i] = string(msg[startOff : startOff+MaxArgvLen])
	}
	return res, nil
}

// decode bytes into struct ExitProcMsg, len of bytes must equal to ExitProcMsgSize
func DecodeToExitProcMsg(msg []byte) (*ExitProcMsg, error) {
	if len(msg) < ExitProcMsgSize {
		return nil, fmt.Errorf("Exit process message unfit to struct ExitProcMsg, size of message is %d, expect %d.", len(msg), ExitProcMsgSize)
	}
	res := &ExitProcMsg{
		Pid:       binary.LittleEndian.Uint32(msg[:4]),
		Ppid:      binary.LittleEndian.Uint32(msg[4:8]),
		ErrorCode: int(binary.LittleEndian.Uint32(msg[8:])),
	}
	return res, nil
}

// load eBPF program to tracepoint sys_enter_execve and sys_enter_exit
// need two channel to receive message from eBPF program
func LoadBpfProgram(ctx context.Context, execMsgCh chan<- *NewProcMsg, exitMsgCh chan<- *ExitProcMsg) (*NewProcBPFObjs, error) {
	blo := &NewProcBPFObjs{}
	var (
		err                    error
		bpfModule              *bpf.Module
		prog_exec, prog_exit   *bpf.BPFProg
		execByteCh, exitByteCh chan []byte
		execRingbuf            *bpf.RingBuffer
		exitRingbuf            *bpf.RingBuffer
	)

	bpfModule, err = bpf.NewModuleFromFile("/home/ivic/program/abyss/newProcTracing/newProcess.bpf.o")
	if err != nil {
		return nil, err
	}
	blo.module = bpfModule

	if err = bpfModule.BPFLoadObject(); err != nil {
		goto moduleAndProgErr
	}

	glog.Info("Start load and attach program")

	prog_exec, err = bpfModule.GetProgram("handle_exec")
	if err != nil {
		err = errors.WithMessage(err, "Can not load eBPF program \"handle_exec\".")
		goto moduleAndProgErr
	}
	blo.programs = append(blo.programs, prog_exec)
	prog_exit, err = bpfModule.GetProgram("handle_exit")
	if err != nil {
		err = errors.WithMessage(err, "Can not load eBPF program \"handle_exit\".")
		goto moduleAndProgErr
	}
	blo.programs = append(blo.programs, prog_exit)

	if _, err = prog_exec.AttachTracepoint("syscalls", "sys_enter_execve"); err != nil {
		err = errors.WithMessage(err, "Can not attach program \"handle_exit\" to tracepoint.")
		goto moduleAndProgErr
	}
	if _, err = prog_exit.AttachTracepoint("sched", "sched_process_exit"); err != nil {
		err = errors.WithMessage(err, "Can not attach program \"handle_exit\" to tracepoint.")
		goto moduleAndProgErr
	}

	fmt.Println("Start create map")

	execByteCh = make(chan []byte)
	execRingbuf, err = bpfModule.InitRingBuf("new_proc", execByteCh)
	if err != nil {
		goto initExecBufErr
	}
	exitByteCh = make(chan []byte)
	exitRingbuf, err = bpfModule.InitRingBuf("exit_proc", exitByteCh)
	if err != nil {
		goto initExitBufErr
	}
	blo.bpfMaps = append(blo.bpfMaps, []*bpf.RingBuffer{execRingbuf, exitRingbuf}...)
	execRingbuf.Start()
	exitRingbuf.Start()

	go receiveExecMsg(ctx, execByteCh, execMsgCh)
	go receiveExitMsg(ctx, exitByteCh, exitMsgCh)

	fmt.Println("Load succeed.")
	return blo, nil

initExitBufErr:
	exitRingbuf.Stop()
	exitRingbuf.Close()
initExecBufErr:
	execRingbuf.Stop()
	execRingbuf.Close()
moduleAndProgErr:
	bpfModule.Close()
	return nil, err
}

func CloseBpfObject(obj *NewProcBPFObjs) {
	obj.module.Close()
	for _, m := range obj.bpfMaps {
		m.Stop()
		m.Close()
	}
}

func receiveExecMsg(ctx context.Context, mapCh chan []byte, msgCh chan<- *NewProcMsg) {
	for {
		select {
		case p := <-mapCh:
			msg, err := DecodeToNewProcMsg(p)
			if err != nil {
				glog.Warning(err.Error())
				continue
			}
			msgCh <- msg
		case <-ctx.Done():
			glog.Info("Routine \"receiveExecMsg\" exit.\n")
			return
		}
	}
}

func receiveExitMsg(ctx context.Context, mapCh chan []byte, msgCh chan<- *ExitProcMsg) {
	for {
		select {
		case p := <-mapCh:
			msg, err := DecodeToExitProcMsg(p)
			if err != nil {
				glog.Warning(err.Error())
				continue
			}
			glog.Infof("Received an exit message, %p\n", msg)
			msgCh <- msg
		case <-ctx.Done():
			glog.Info("Routine \"receiveExitMsg\" exit.\n")
			return
		}
	}
}

//func main() {
//	flag.Parse()
//
//	ctx, cancel := context.WithCancel(context.Background())
//	execCh, exitCh := make(chan *NewProcMsg, 10), make(chan *ExitProcMsg, 10)
//
//	obj, err := LoadBpfProgram(ctx, execCh, exitCh)
//	if err != nil {
//		glog.Error(err.Error())
//	}
//	defer func() {
//		fmt.Println("Exit the program.")
//		CloseBpfObject(obj)
//		glog.Flush()
//	}()
//
//	sig := make(chan bool)
//	go func() {
//		time.Sleep(time.Second * 10)
//		sig <- true
//	}()
//	for {
//		select {
//		case m := <-execCh:
//			flag := false
//			for _, arg := range m.Argv {
//				str := arg[:11]
//				if str == "-bpfMonitor" {
//					flag = true
//					break
//				}
//			}
//			if flag {
//				fmt.Println("execve:", m)
//				for _, str := range m.Argv {
//					if []byte(str)[0] == 0 {
//						break
//					}
//					fmt.Println("\t", []byte(str))
//				}
//			}
//		case m := <-exitCh:
//			fmt.Println("exit:", m)
//		case _ = <-sig:
//			cancel()
//			fmt.Println("Monitor Process Exit.")
//			glog.Info("Monitor Process Exit!")
//			return
//		}
//	}
//}
