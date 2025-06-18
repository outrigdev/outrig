// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logprocess

import (
	"fmt"
	"sync"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilds"
)

// LogCollector implements the collector.Collector interface for log collection
type LogCollector struct {
	config               *utilds.SetOnceConfig[config.LogProcessorConfig]
	dataLock             sync.RWMutex // protects externalLogWrapError
	externalLogWrapError error        // Store any error from external log wrapping
}

// CollectorName returns the unique name of the collector
func (lc *LogCollector) CollectorName() string {
	return "logprocess"
}

// singleton instance
var instance *LogCollector
var instanceOnce sync.Once

// GetInstance returns the singleton instance of LogCollector
func GetInstance() *LogCollector {
	instanceOnce.Do(func() {
		instance = &LogCollector{
			config: utilds.NewSetOnceConfig(config.DefaultConfig().LogProcessorConfig),
		}
	})
	return instance
}

func Init(cfg *config.LogProcessorConfig) error {
	lc := GetInstance()
	if loginitex.IsExternalLogWrapActive() {
		return fmt.Errorf("log process collector is already initialized")
	}
	ok := lc.config.SetOnce(cfg)
	if !ok {
		return fmt.Errorf("log process collector configuration already set")
	}
	collector.RegisterCollector(lc)
	return nil
}

func (lc *LogCollector) Enable() {
	cfg := lc.config.Get()
	if !cfg.Enabled {
		return
	}
	
	// Check if already enabled
	if loginitex.IsExternalLogWrapActive() {
		return
	}
	
	// Enable external log wrapping if controller is available
	// Get the appRunId from the config
	appRunId := config.GetAppRunId()
	isDev := config.UseDevConfig()

	// Use the new external log capture mechanism
	err := loginitex.EnableExternalLogWrap(appRunId, cfg, isDev)
	lc.setExternalLogWrapError(err)

	ctl := global.GetController()
	if ctl != nil {
		if err != nil {
			ctl.ILog("Failed to enable external log wrapping: %v", err)
		} else {
			ctl.ILog("External log wrapping enabled")
		}
	}
}

func (lc *LogCollector) Disable() {
	// TODO - don't disable log wrapping once enabled
	// It is risky to disable because there can be a race condition which causes SIGPIPE errors
	//   as we try to swap the file descriptors back and coordinate killing the external process

	// Disable external log wrapping
	// loginitex.DisableExternalLogWrap()
}

// OnNewConnection is called when a new connection is established
func (lc *LogCollector) OnNewConnection() {
	// No action needed for log collector
}

// GetStatus returns the current status of the log collector
func (lc *LogCollector) GetStatus() ds.CollectorStatus {
	cfg := lc.config.Get()
	status := ds.CollectorStatus{
		Running: cfg.Enabled,
	}

	if !cfg.Enabled {
		status.Info = "Disabled in configuration"
	} else {
		// Check if external log wrapping is active
		isExternalActive := loginitex.IsExternalLogWrapActive()
		if isExternalActive {
			status.Info = "Log processing active (external log wrapping enabled)"
		} else {
			status.Info = "Log processing active (external log wrapping disabled)"
		}

		// Check for external log wrap error
		if err := lc.getExternalLogWrapError(); err != nil {
			status.Errors = append(status.Errors, "External log wrapping failed: "+err.Error())
		}
	}

	return status
}

// setExternalLogWrapError sets the external log wrap error with proper locking
func (lc *LogCollector) setExternalLogWrapError(err error) {
	lc.dataLock.Lock()
	defer lc.dataLock.Unlock()
	lc.externalLogWrapError = err
}

// getExternalLogWrapError gets the external log wrap error with proper locking
func (lc *LogCollector) getExternalLogWrapError() error {
	lc.dataLock.Lock()
	defer lc.dataLock.Unlock()
	return lc.externalLogWrapError
}
