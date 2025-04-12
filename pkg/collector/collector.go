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
	// The config parameter is the collector-specific configuration, which can be cast to the appropriate type
	InitCollector(controller ds.Controller, config any, appRunContext ds.AppRunContext) error

	Enable()
	Disable()
}
