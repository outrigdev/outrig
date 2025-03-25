package runtimestats

import (
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/shirou/gopsutil/v4/process"
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
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig RuntimeStatsCollector")
		rc.CollectRuntimeStats()
	}()
	rc.ticker = time.NewTicker(1 * time.Second)
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig RuntimeStatsCollector")
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

	// Collect memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Create memory stats info
	memStatsInfo := ds.MemoryStatsInfo{
		Alloc:        memStats.Alloc,
		TotalAlloc:   memStats.TotalAlloc,
		Sys:          memStats.Sys,
		HeapAlloc:    memStats.HeapAlloc,
		HeapSys:      memStats.HeapSys,
		HeapIdle:     memStats.HeapIdle,
		HeapInuse:    memStats.HeapInuse,
		StackInuse:   memStats.StackInuse,
		StackSys:     memStats.StackSys,
		MSpanInuse:   memStats.MSpanInuse,
		MSpanSys:     memStats.MSpanSys,
		MCacheInuse:  memStats.MCacheInuse,
		MCacheSys:    memStats.MCacheSys,
		GCSys:        memStats.GCSys,
		OtherSys:     memStats.OtherSys,
		NextGC:       memStats.NextGC,
		LastGC:       memStats.LastGC,
		PauseTotalNs: memStats.PauseTotalNs,
		NumGC:        memStats.NumGC,
	}

	// Get current process information
	pid := os.Getpid()

	// Default values
	cpuPercent := 0.0
	cwd, _ := os.Getwd() // Get current working directory from os package

	// Get CPU percent using gopsutil
	proc, err := process.NewProcess(int32(pid))
	if err == nil {
		// Get CPU percent (might return 0 on first call)
		cpuPercent, _ = proc.CPUPercent()
	}

	// Create runtime stats info
	runtimeStats := &ds.RuntimeStatsInfo{
		Ts:             time.Now().UnixMilli(),
		CPUUsage:       cpuPercent,
		GoRoutineCount: runtime.NumGoroutine(),
		GoMaxProcs:     runtime.GOMAXPROCS(0), // 0 means get current value without changing it
		NumCPU:         runtime.NumCPU(),
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		GoVersion:      runtime.Version(),
		Pid:            pid,
		Cwd:            cwd,
		MemStats:       memStatsInfo,
	}

	// Send the runtime stats packet
	pk := &ds.PacketType{
		Type: ds.PacketTypeRuntimeStats,
		Data: runtimeStats,
	}

	rc.controller.SendPacket(pk)
}
