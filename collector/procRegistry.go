package collector

import (
	"fmt"

	"wanggj.com/abyss/module"
)

// ProcRegistry is used to register collectors and pushers.
// The registered pusher need to call Start func to start collect, and
// syscall monitor func need initialization.
//
// ProcRegistry provide Start and Close func to initialize and release
// system resource.
type ProcRegistry struct {
	registry Registry

	// pusher store all pushers, which use push module and
	// need Start and Close
	pusherByName map[string]*Pusher

	// otherCollectors is all collectors can use pull module
	pullerByName map[string]Collector
}

// func PullerReg is used to registry a collector, which dose not
// need other actions.
func (p *ProcRegistry) PullerReg(name string, collector Collector) error {
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
func (p *ProcRegistry) PusherReg(name string, pusher *Pusher) error {
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

// ProcRegOpts is the config struct used to create a ProcRegistry, generate collectors
// and register then into ProcRegistry.
type ProcRegOpts struct {
}
