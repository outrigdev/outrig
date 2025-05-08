// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logprocess

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/ds"
)

// LogCollector implements the collector.Collector interface for log collection
type LogCollector struct {
	controller    ds.Controller
	config        ds.LogProcessorConfig
	appRunContext ds.AppRunContext
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
func (lc *LogCollector) InitCollector(controller ds.Controller, config any, appRunContext ds.AppRunContext) error {
	lc.controller = controller
	if logConfig, ok := config.(ds.LogProcessorConfig); ok {
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
