package collector

import "github.com/outrigdev/outrig/pkg/ds"

// Collector defines the interface for collection functionality
type Collector interface {
	// InitCollector initializes the collector with a controller
	// The controller can be nil during early initialization
	InitCollector(controller ds.Controller) error

	// OnFirstConnect is called when the first connection is established
	OnFirstConnect()
}
