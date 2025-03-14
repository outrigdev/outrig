package runtimestats

import (
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
)

// RuntimeStatsCollector implements the collector.Collector interface for runtime stats collection
type RuntimeStatsCollector struct {
	lock       sync.Mutex
	controller ds.Controller
	ticker     *time.Ticker
}

// CollectorName returns the unique name of the collector
func (rc *RuntimeStatsCollector) CollectorName() string {
	return "runtimestats"
}

// singleton instance
var instance *RuntimeStatsCollector
var instanceOnce sync.Once

// GetInstance returns the singleton instance of RuntimeStatsCollector
func GetInstance() *RuntimeStatsCollector {
	instanceOnce.Do(func() {
		instance = &RuntimeStatsCollector{}
	})
	return instance
}

// InitCollector initializes the runtime stats collector with a controller
func (rc *RuntimeStatsCollector) InitCollector(controller ds.Controller) error {
	rc.controller = controller
	return nil
}

// Enable is called when the collector should start collecting data
func (rc *RuntimeStatsCollector) Enable() {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	if rc.ticker != nil {
		return
	}
	go rc.CollectRuntimeStats()
	rc.ticker = time.NewTicker(1 * time.Second)
	go func() {
		for range rc.ticker.C {
			rc.CollectRuntimeStats()
		}
	}()
}

// Disable stops the collector
func (rc *RuntimeStatsCollector) Disable() {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	if rc.ticker == nil {
		return
	}
	rc.ticker.Stop()
	rc.ticker = nil
}

// CollectRuntimeStats collects runtime statistics and sends them to the controller
func (rc *RuntimeStatsCollector) CollectRuntimeStats() {
	if !global.OutrigEnabled.Load() || rc.controller == nil {
		return
	}

	// For now, we're just creating a placeholder structure
	// Actual implementation will be added later
	runtimeStats := &ds.RuntimeStatsInfo{
		Ts: time.Now().UnixMilli(),
		// Other fields will be populated later
	}

	// Send the runtime stats packet
	pk := &ds.PacketType{
		Type: ds.PacketTypeRuntimeStats,
		Data: runtimeStats,
	}

	rc.controller.SendPacket(pk)
}
