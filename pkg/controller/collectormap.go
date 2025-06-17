// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
)

var (
	collectors     map[string]collector.Collector
	collectorsLock sync.Mutex
)

func init() {
	collectors = make(map[string]collector.Collector)
}

func RegisterCollector(c collector.Collector) {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()
	collectors[c.CollectorName()] = c
}

func getCollectorByName(name string) collector.Collector {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()
	return collectors[name]
}

func getCollectorStatuses() map[string]ds.CollectorStatus {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()

	statuses := make(map[string]ds.CollectorStatus)
	for name, collector := range collectors {
		statuses[name] = collector.GetStatus()
	}
	return statuses
}

func setCollectorsEnabled(enabled bool, cfg *config.Config) {
	collectorsLock.Lock()
	defer collectorsLock.Unlock()

	if enabled {
		// Only enable collectors that have Enabled set to true in their config
		for name, collector := range collectors {
			switch name {
			case "logprocess":
				if cfg.LogProcessorConfig.Enabled {
					collector.Enable()
				}
			case "goroutine":
				if cfg.GoRoutineConfig.Enabled {
					collector.Enable()
				}
			case "watch":
				if cfg.WatchConfig.Enabled {
					collector.Enable()
				}
			case "runtimestats":
				if cfg.RuntimeStatsConfig.Enabled {
					collector.Enable()
				}
			}
		}
	} else {
		for _, collector := range collectors {
			collector.Disable()
		}
	}
}
