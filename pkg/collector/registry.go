// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
)

var (
	collectors        map[string]Collector
	collectorsEnabled bool
	collectorsLock    sync.Mutex
)

func init() {
	collectors = make(map[string]Collector)
}

func RegisterCollector(c Collector) {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()
	collectors[c.CollectorName()] = c
	if collectorsEnabled {
		c.Enable()
	}
}

func GetCollectorByName(name string) Collector {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()
	return collectors[name]
}

func GetCollectorStatuses() map[string]ds.CollectorStatus {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()

	statuses := make(map[string]ds.CollectorStatus)
	for name, collector := range collectors {
		statuses[name] = collector.GetStatus()
	}
	return statuses
}

func SetCollectorsEnabled(enabled bool, cfg *config.Config) {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()

	if collectorsEnabled == enabled {
		return
	}
	collectorsEnabled = enabled
	if enabled {
		for _, collector := range collectors {
			collector.Enable()
		}
	} else {
		for _, collector := range collectors {
			collector.Disable()
		}
	}
}