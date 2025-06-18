// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import "github.com/outrigdev/outrig/pkg/ds"

// Collector defines the interface for collection functionality
// Implementations: goroutine/goroutine.go, logprocess/logprocess.go, runtimestats/runtimestats.go, watch/watch.go
type Collector interface {
	// CollectorName returns the unique name of the collector
	CollectorName() string

	Enable()
	Disable()

	// GetStatus returns the current status of the collector
	GetStatus() ds.CollectorStatus

	// OnNewConnection is called when a new connection is established
	// Collectors that need to send full updates on new connections should implement this
	OnNewConnection()
}
