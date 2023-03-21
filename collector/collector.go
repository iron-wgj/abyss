package collector

// Collector is the interface implemented by anything that can be used by
// Abyss to collect metrics. A Collector has to be registered for collection.
//
// An implemention of Collector may collect multiple metrics onec time.
type Collector interface {
	// Describe sends the duper-set of all possible descriptors of metrics
	// collected by this collector to the provided channel and returns
	// once the last Desc has been sent.
	//
	// It is valid if one and the same Collector sends duplicate
	// Desc. Those Duplicateds are simply ignored. However, two
	// different Collectors must not send duplicate Descs.
	//
	// This method sends the same Descs throughout the lifetime
	// of the Collector. It may be called concurrently and
	// therefore must be implemented in a concurrency safe way.
	//
	// If a Collector encounters an error while executing this method, it
	// must send a invalid Desc (created with NewInvalidDesc) to signal
	// the error to the registery.
	Describe(chan<- *Desc)

	// Collect is called by the Prometheus registry when collecting
	// metrics. The implementation sends each collected metric via the
	// provided channel and returns once the last metric has been sent.
	// The Desc of each sent metric is one tof those returned by Describe
	// method.
	//
	// This method may be called concurrently and must therefore be
	// implemented in a concurrency safe way.
	Collect(chan<- Metric)
}
