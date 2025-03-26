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

// InitCollector initializes the log collector with a controller
func (lc *LogCollector) InitCollector(controller ds.Controller) error {
	lc.controller = controller
	return nil
}

func (lc *LogCollector) Enable() {
	// Enable external log wrapping if controller is available
	config := lc.controller.GetConfig()
	if config.LogProcessorConfig != nil {
		// Get the appRunId from the controller
		appRunId := lc.controller.GetAppRunId()
		
		// Use the new external log capture mechanism
		err := loginitex.EnableExternalLogWrap(appRunId, *config.LogProcessorConfig, config.Dev)
		if err != nil {
			fmt.Printf("Failed to enable external log wrapping: %v\n", err)
		} else {
			fmt.Printf("External log wrapping enabled\n")
		}
	}
}

func (lc *LogCollector) Disable() {
	// Disable external log wrapping
	loginitex.DisableExternalLogWrap()
}
