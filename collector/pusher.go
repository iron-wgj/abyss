package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"wanggj.com/abyss/collector/pushFunc"
)

const (
	dataReceiverLen = 10
)

// Analyzer is used to analysis Data, an Analyzer must implement Collector
type StatefulAnalyzer interface {
	Collector

	// Observe is used to receive new data
	Observe(*pushFunc.DataPair)
}

// StatelessAnalyzer is used to analysis data in a time range
type StatelessAnalyzer interface {
	// The same as Describe function of Collector
	Describe(chan<- *Desc)
	// Analyze receive data series and send Metrics into ch, the implementation
	// must insure data series not changed
	Analyze([]*pushFunc.DataPair, chan<- Metric)
}

// Pusher is a collector that can create goroutine to collect data initiactivly.
// It is used to perform analies on data collected to data aggregation or
// alarms, which can shrink the amount of data neeeded to transition.
type Pusher struct {
	// Desc is used to identify Pusher, if Pusher value need to
	// collect, Desc would be useful
	Desc      *Desc
	selfCol   bool      // true if Data need to be collected
	valueType ValueType // metric type the data default chenged to
	// Data is the data collected has been analysised
	Data []*pushFunc.DataPair
	mtx  sync.Mutex

	// dataBuf storaged data that collected from last time Collect called
	dataBuf []*pushFunc.DataPair
	bufMtx  sync.Mutex

	// data receiver is a channel used to collect data from cf, its length
	// is dataReceiverLen
	receiver chan *pushFunc.DataPair

	// CollectFunc will be used in Start
	pf     pushFunc.PushFunc
	cancel context.CancelFunc

	// stateful analyzer need to observe new data each time a new pushFunc.DataPair
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
	desc *Desc,
	selfCol bool,
	valueType ValueType,
	pf pushFunc.PushFunc,
	statefulAnas []StatefulAnalyzer,
	statelessAnas []StatelessAnalyzer,
	tr time.Duration,
) *Pusher {
	if tr < 0 {
		return nil
	}

	result := &Pusher{
		Desc:         desc,
		selfCol:      selfCol,
		valueType:    valueType,
		Data:         make([]*pushFunc.DataPair, 0),
		dataBuf:      make([]*pushFunc.DataPair, 0),
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
	p.receiver = make(chan *pushFunc.DataPair, dataReceiverLen)
	go p.pf.Push(p.receiver, ctx)

	go p.receive()

	p.closed = false
}

// Stop the Pusher, p.receiver channel must be closed by pushFunc
func (p *Pusher) Stop() {
	p.mtx.Lock()
	//fmt.Println("======= Stop the pusher ======")
	defer p.mtx.Unlock()

	if p.closed {
		return
	}

	p.cancel()

	p.closed = true
	// for range p.receiver {
	// }
}

func (p *Pusher) Describe(ch chan<- *Desc) {
	for _, a := range p.StatefulAna {
		a.Describe(ch)
	}
	for _, a := range p.StatelessAna {
		a.Describe(ch)
	}
	if p.selfCol {
		ch <- p.Desc
	}
}

// selfCollec is used to collect raw data of pusher
func (p *Pusher) selfCollect(data []*pushFunc.DataPair, ch chan<- Metric) {
	for _, dp := range data {
		cm, err := NewConstMetric(
			p.Desc,
			p.valueType,
			dp.Value,
		)
		if err != nil {
			fmt.Println(err)
			glog.Error(err)
			continue
		}
		ch <- NewTimeStampMetric(dp.Timestamp, cm)
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
	p.dataBuf = make([]*pushFunc.DataPair, 0, len(tmp))
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
			//fmt.Println("=============Analyze end========")
			wg.Done()
		}(a)
	}

	// collect self
	if p.selfCol {
		p.selfCollect(tmp, ch)
	}

	wg.Wait()
	p.mtx.Unlock()
	return
}

// /////////////////////////////////
// Pusher options, used for Pusher initialization
type PusherOpts struct {
	Opts      `yaml:"desc"`
	SelfCol   bool   `yaml:"selfcol"`
	ValueType int    `yaml:"valuetype"`
	Inv       string `yaml:"inv"`
	Pf        string `yaml:"pushFunc"`
	PfInv     string `yaml:"pfinv"`
}

type PusherInitErr struct {
	err error
}

func (pe *PusherInitErr) Error() string {
	if pe.err == nil {
		return ""
	}
	ret := fmt.Sprintf("Init Pusher error: %s", pe.err.Error())
	return ret
}

func NewPusherInitErr(err error) error {
	return &PusherInitErr{
		err: err,
	}
}

func NewPusherFromOpts(
	pid uint32,
	opt PusherOpts,
	sla []StatelessAnalyzer,
	sfa []StatefulAnalyzer,
) (*Pusher, error) {
	// add pid into desc constlabel
	opt.ConstLabels["PID"] = fmt.Sprint(pid)
	desc := NewDesc(
		opt.Name,
		opt.Help,
		opt.Level,
		nil,
		opt.ConstLabels,
	)
	if desc == nil {
		return nil, NewPusherInitErr(fmt.Errorf("Invalid desc opts."))
	}
	if opt.ValueType <= 0 || opt.ValueType >= 3 {
		return nil, NewPusherInitErr(
			fmt.Errorf(
				"Invalid Pusher ValueType %d, name: %s.",
				opt.ValueType,
				opt.Name,
			),
		)
	}
	inv, err := time.ParseDuration(opt.Inv)
	if err != nil {
		return nil, NewPusherInitErr(
			fmt.Errorf("Init Pusher error: %s when init opt.Inv.", err.Error()),
		)
	}
	pfinv, err := time.ParseDuration(opt.PfInv)
	if err != nil {
		return nil, NewPusherInitErr(
			fmt.Errorf("Init Pusher error: %s when init opt.PfInv.", err.Error()),
		)
	}

	if pfinv < time.Millisecond*100 || pfinv > time.Second*5 {
		return nil, NewPusherInitErr(
			fmt.Errorf(
				"pfInv illegal, must between 100ms and 5s, got %s.",
				pfinv.String(),
			),
		)
	}

	pf, err := pushFunc.NewPushFunc(
		pid,
		opt.Pf,
		pfinv,
		&pushFunc.PfOpts{
			TargetPath: "../bpf/test/test",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("Init Pusher error: %s when init pushFunc", err.Error())
	}

	pusher := NewPusher(
		desc,
		opt.SelfCol,
		ValueType(opt.ValueType),
		pf,
		sfa,
		sla,
		inv,
	)

	return pusher, nil
}
