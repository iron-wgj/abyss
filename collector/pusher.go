package collector

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	dataReceiverLen = 10
)

// DataPair is used to storage data collected by Pusher
type DataPair struct {
	Value     float64
	Timestamp time.Time
}

func NewDataPair(v float64, t time.Time) *DataPair {
	return &DataPair{Value: v, Timestamp: t}
}

// CollectFunc will be used to create a goroutine in Pusher, it receives a channel
// and a context.
//
// The channel is used to receive data and received data
// will be storaged into dataBuf. Each time Collect func is called,
// dataBuf will be flushed into Data and data out of time will be cleared.
//
// context is used to close goroutine, like context.WithCancel
type PushFunc interface {
	Push(chan<- *DataPair, context.Context)
}

// Analyzer is used to analysis Data, an Analyzer must implement Collector
type StatefulAnalyzer interface {
	Collector

	// Observe is used to receive new data
	Observe(*DataPair)
}

// StatelessAnalyzer is used to analysis data in a time range
type StatelessAnalyzer interface {
	// The same as Describe function of Collector
	Describe(chan<- *Desc)
	// Analyze receive data series and send Metrics into ch, the implementation
	// must insure data series not changed
	Analyze([]*DataPair, chan<- Metric)
}

// Pusher is a collector that can create goroutine to collect data initiactivly.
// It is used to perform analies on data collected to data aggregation or
// alarms, which can shrink the amount of data neeeded to transition.
type Pusher struct {
	// Data is the data collected has been analysised
	Data []*DataPair
	mtx  sync.Mutex

	// dataBuf storaged data that collected from last time Collect called
	dataBuf []*DataPair
	bufMtx  sync.Mutex

	// data receiver is a channel used to collect data from cf, its length
	// is dataReceiverLen
	receiver chan *DataPair

	// CollectFunc will be used in Start
	pf     PushFunc
	cancel context.CancelFunc

	// stateful analyzer need to observe new data each time a new DataPair
	// is pushed.
	StatefulAna []StatefulAnalyzer
	// stateless analyzer analyze Data each time Collecte func is referenced
	StatelessAna []StatelessAnalyzer

	// Pusher only keep data inside timeRange before time.Now()
	TimeRange time.Duration

	// closed is used to prevent Collect continue after Pusher close
	closed bool
}

func NewPusher(
	pf PushFunc,
	statefulAnas []StatefulAnalyzer,
	statelessAnas []StatelessAnalyzer,
	tr time.Duration,
) *Pusher {
	if tr < 0 {
		return nil
	}

	result := &Pusher{
		Data:         make([]*DataPair, 0),
		dataBuf:      make([]*DataPair, 0),
		pf:           pf,
		StatefulAna:  statefulAnas,
		StatelessAna: statelessAnas,
		TimeRange:    tr,
		closed:       true,
	}
	return result
}

func (p *Pusher) receive() {
	for d := range p.receiver {
		p.bufMtx.Lock()
		p.dataBuf = append(p.dataBuf, d)
		p.bufMtx.Unlock()
		for _, a := range p.StatefulAna {
			a.Observe(d)
		}
	}
}

func (p *Pusher) Start() {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if !p.closed {
		return
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	p.cancel = cancelFunc
	p.receiver = make(chan *DataPair, dataReceiverLen)
	go p.pf.Push(p.receiver, ctx)

	go p.receive()

	p.closed = false
}

func (p *Pusher) Stop() {
	p.mtx.Lock()
	fmt.Println("======= Stop the pusher ======")
	defer p.mtx.Unlock()

	if p.closed {
		return
	}

	p.cancel()

	p.closed = true
	close(p.receiver)
	for range p.receiver {
	}
}

func (p *Pusher) Describe(ch chan<- *Desc) {
	for _, a := range p.StatefulAna {
		a.Describe(ch)
	}
	for _, a := range p.StatelessAna {
		a.Describe(ch)
	}
}

func (p *Pusher) Collect(ch chan<- Metric) {
	p.mtx.Lock()
	if p.closed {
		p.mtx.Unlock()
		return
	}

	wg := new(sync.WaitGroup)

	// can start collect StatefulAnalyzer now
	for _, a := range p.StatefulAna {
		wg.Add(1)
		go func(a StatefulAnalyzer) {
			a.Collect(ch)
			wg.Done()
		}(a)
	}

	p.bufMtx.Lock()
	tmp := p.dataBuf
	p.dataBuf = make([]*DataPair, 0, len(tmp))
	p.bufMtx.Unlock()

	timeUpBound := time.Now().Add(-p.TimeRange)
	ot := -1
	for i, d := range p.Data {
		if d.Timestamp.Before(timeUpBound) {
			ot = i
		}
	}
	p.Data = p.Data[ot+1:]
	p.Data = append(p.Data, tmp...)

	// start StatelessAnalyzer now
	for _, a := range p.StatelessAna {
		wg.Add(1)
		go func(a StatelessAnalyzer) {
			a.Analyze(p.Data, ch)
			fmt.Println("=============Analyze end========")
			wg.Done()
		}(a)
	}

	wg.Wait()
	p.mtx.Unlock()
	return
}
