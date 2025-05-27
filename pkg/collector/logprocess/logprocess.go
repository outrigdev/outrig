// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logprocess

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
)

// LogCollector implements the collector.Collector interface for log collection
type LogCollector struct {
	controller           ds.Controller
	config               config.LogProcessorConfig
	appRunContext        ds.AppRunContext
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
		instance = &LogCollector{}
	})
	return instance
}

// InitCollector initializes the log collector with a controller and configuration
func (lc *LogCollector) InitCollector(controller ds.Controller, cfg any, appRunContext ds.AppRunContext) error {
	lc.controller = controller
	if logConfig, ok := cfg.(config.LogProcessorConfig); ok {
		lc.config = logConfig
	}
	lc.appRunContext = appRunContext
	return nil
}

func (lc *LogCollector) Enable() {
	// Enable external log wrapping if controller is available
	// Get the appRunId from the controller
	appRunId := lc.appRunContext.AppRunId
	isDev := lc.appRunContext.IsDev

	// Use the new external log capture mechanism
	err := loginitex.EnableExternalLogWrap(appRunId, lc.config, isDev)
	lc.setExternalLogWrapError(err)
	
	if err != nil {
		lc.controller.ILog("Failed to enable external log wrapping: %v", err)
	} else {
		lc.controller.ILog("External log wrapping enabled")
	}
}

func (lc *LogCollector) Disable() {
	// TODO - don't disable log wrapping once enabled
	// It is risky to disable because there can be a race condition which causes SIGPIPE errors
	//   as we try to swap the file descriptors back and coordinate killing the external process

	// Disable external log wrapping
	// loginitex.DisableExternalLogWrap()
}

// GetStatus returns the current status of the log collector
func (lc *LogCollector) GetStatus() ds.CollectorStatus {
	status := ds.CollectorStatus{
		Running: lc.config.Enabled,
	}
	
	if !lc.config.Enabled {
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
