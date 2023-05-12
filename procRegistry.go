package main

import (
	"bytes"
	"fmt"
	"strings"

	analyzer "wanggj.com/abyss/analyzers"
	"wanggj.com/abyss/collector"
	"wanggj.com/abyss/module"
)

// ProcRegistry is used to register collectors and pushers.
// The registered pusher need to call Start func to start collect, and
// syscall monitor func need initialization.
//
// ProcRegistry provide Start and Close func to initialize and release
// system resource.
type ProcRegistry struct {
	registry *collector.Registry

	// pusher store all pushers, which use push module and
	// need Start and Close
	pusherByName map[string]*collector.Pusher

	// otherCollectors is all collectors can use pull module
	pullerByName map[string]collector.Collector
}

// func PullerReg is used to registry a collector, which dose not
// need other actions.
func (p *ProcRegistry) PullerReg(name string, collector collector.Collector) error {
	if _, ok := p.pullerByName[name]; ok {
		return fmt.Errorf("Puller named %s has been existed.", name)
	}

	if err := p.registry.Register(collector); err != nil {
		return err
	}
	p.pullerByName[name] = collector
	return nil
}

// func PullerUnreg unregistry a puller
func (p *ProcRegistry) PullerUnreg(name string) {
	c, ok := p.pullerByName[name]
	if !ok {
		return
	}
	p.registry.Unregister(c)
	delete(p.pullerByName, name)
	return
}

// func PusherReg is used to registry a pusher
func (p *ProcRegistry) PusherReg(name string, pusher *collector.Pusher) error {
	if _, ok := p.pusherByName[name]; ok {
		return fmt.Errorf("Puller named %s has been existed.", name)
	}

	if err := p.registry.Register(pusher); err != nil {
		return err
	}
	p.pusherByName[name] = pusher
	return nil
}

// func PusherUnreg unregistry a pusher
func (p *ProcRegistry) PusherUnreg(name string) {
	c, ok := p.pusherByName[name]
	if !ok {
		return
	}
	c.Stop()
	p.registry.Unregister(c)
	delete(p.pusherByName, name)
	return
}

// func Gather is used to Gather module.MetricFamily from registry
func (p *ProcRegistry) Gather() (map[int][]*module.MetricFamily, error) {
	return p.registry.Gather()
}

// func Start initialize the pushers, which need to start the pushFunc to
// collector data
func (p *ProcRegistry) Start() {
	for _, pu := range p.pusherByName {
		pu.Start()
	}
}

// func Stop stop all pushers
func (p *ProcRegistry) Stop() {
	for _, pu := range p.pusherByName {
		pu.Stop()
	}
}

// PusherConfig is used to generate a pusher with analyzers
type PusherConfig struct {
	collector.PusherOpts `yaml:"pusher"`
	SlAna                []analyzer.AnaConfig `yaml:"slana,omitempty"`
	SfAna                []analyzer.AnaConfig `yaml:"sfana,omitempty"`
}

func NewPusherFromConfig(pid uint32, pc *PusherConfig) (*collector.Pusher, error) {
	sla, sfa := []collector.StatelessAnalyzer{}, []collector.StatefulAnalyzer{}
	for _, cfg := range pc.SlAna {
		alz, err := analyzer.GetSlaFromConfig(pid, cfg)
		if err != nil {
			return nil, err
		}
		sla = append(sla, alz)
	}
	for _, cfg := range pc.SfAna {
		alz, err := analyzer.GetSfaFromConfig(pid, cfg)
		if err != nil {
			return nil, err
		}
		sfa = append(sfa, alz)
	}
	return collector.NewPusherFromOpts(pid, pc.PusherOpts, sla, sfa)
}

// ProcRegOpts is the config struct used to create a ProcRegistry, generate collectors
// and register then into ProcRegistry.
type ProcConfig struct {
	Pusher []PusherConfig `yaml:"pushercfg,omitempty"`
}

func NewProcRegFromConfig(pid uint32, cfg *ProcConfig) (*ProcRegistry, error) {
	errs := collector.MultiError{}
	procReg := &ProcRegistry{
		registry:     collector.NewRegistry(),
		pusherByName: map[string]*collector.Pusher{},
		pullerByName: map[string]collector.Collector{},
	}
	for idx := range cfg.Pusher {
		pu, err := NewPusherFromConfig(pid, &(cfg.Pusher[idx]))
		if err != nil {
			fmt.Printf("New Puhser error, %s.", err.Error())
			errs.Append(err)
			continue
		}

		// register the pusher into procRegistry
		err = procReg.PusherReg(
			PusherName(pid, &(cfg.Pusher[idx])),
			pu,
		)
		if err != nil {
			fmt.Printf("Pusher Register error: %s.\n", err.Error())
			errs.Append(err)
			continue
		}
	}
	fmt.Println(errs.Error(), len(errs))
	return procReg, errs
}

func PusherName(pid uint32, cfg *PusherConfig) string {
	buf := bytes.NewBufferString("Pid_")
	fmt.Fprint(buf, pid)
	fmt.Fprint(buf, '_')
	fields := strings.Split(cfg.PusherOpts.Pf, ":")
	fmt.Fprint(buf, fields[1])
	return buf.String()
}
