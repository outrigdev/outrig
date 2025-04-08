// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import "github.com/outrigdev/outrig/pkg/ds"

// Collector defines the interface for collection functionality
type Collector interface {
	// CollectorName returns the unique name of the collector
	CollectorName() string

	// InitCollector initializes the collector with a controller
	// The controller can be nil during early initialization
	InitCollector(controller ds.Controller) error

	Enable()
	Disable()
}
