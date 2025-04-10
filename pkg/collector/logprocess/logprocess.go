// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logprocess

import (
	"fmt"
	"sync"

	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/ds"
)

// LogCollector implements the collector.Collector interface for log collection
type LogCollector struct {
	controller ds.Controller
	config     ds.LogProcessorConfig
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
func (lc *LogCollector) InitCollector(controller ds.Controller, config any) error {
	lc.controller = controller
	if logConfig, ok := config.(ds.LogProcessorConfig); ok {
		lc.config = logConfig
	}
	return nil
}

func (lc *LogCollector) Enable() {
	// Enable external log wrapping if controller is available
	// Get the appRunId from the controller
	appRunId := lc.controller.GetAppRunId()
	config := lc.controller.GetConfig()

	// Use the new external log capture mechanism
	err := loginitex.EnableExternalLogWrap(appRunId, lc.config, config.Dev)
	if err != nil {
		fmt.Printf("Failed to enable external log wrapping: %v\n", err)
	} else {
		fmt.Printf("External log wrapping enabled\n")
	}
}

func (lc *LogCollector) Disable() {
	// Disable external log wrapping
	loginitex.DisableExternalLogWrap()
}
