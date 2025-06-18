// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runtimestats

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilds"
)

// RuntimeStatsCollector implements the collector.Collector interface for runtime stats collection
type RuntimeStatsCollector struct {
	config   *utilds.SetOnceConfig[config.RuntimeStatsConfig]
	executor *collector.PeriodicExecutor
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
		instance = &RuntimeStatsCollector{
			config: utilds.NewSetOnceConfig(config.DefaultConfig().RuntimeStatsConfig),
		}
		instance.executor = collector.MakePeriodicExecutor("RuntimeStatsCollector", 1*time.Second, instance.CollectRuntimeStats)
	})
	return instance
}

func Init(cfg *config.RuntimeStatsConfig) error {
	rc := GetInstance()
	if rc.executor.IsEnabled() {
		return fmt.Errorf("runtime stats collector is already initialized")
	}
	ok := rc.config.SetOnce(cfg)
	if !ok {
		return fmt.Errorf("runtime stats collector configuration already set")
	}
	collector.RegisterCollector(rc)
	return nil
}

// Enable is called when the collector should start collecting data
func (rc *RuntimeStatsCollector) Enable() {
	cfg := rc.config.Get()
	if !cfg.Enabled {
		return
	}
	rc.executor.Enable()
}

// Disable stops the collector
func (rc *RuntimeStatsCollector) Disable() {
	rc.executor.Disable()
}

// OnNewConnection is called when a new connection is established
func (rc *RuntimeStatsCollector) OnNewConnection() {
	// No action needed for runtime stats collector
}

// CollectRuntimeStats collects runtime statistics and sends them to the controller
func (rc *RuntimeStatsCollector) CollectRuntimeStats() {
	if !global.OutrigEnabled.Load() {
		return
	}
	ctl := global.GetController()
	if ctl == nil {
		return
	}

	// Collect memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Create memory stats info
	memStatsInfo := ds.MemoryStatsInfo{
		Alloc:            memStats.Alloc,
		TotalAlloc:       memStats.TotalAlloc,
		Sys:              memStats.Sys,
		HeapAlloc:        memStats.HeapAlloc,
		HeapSys:          memStats.HeapSys,
		HeapIdle:         memStats.HeapIdle,
		HeapInuse:        memStats.HeapInuse,
		StackInuse:       memStats.StackInuse,
		StackSys:         memStats.StackSys,
		MSpanInuse:       memStats.MSpanInuse,
		MSpanSys:         memStats.MSpanSys,
		MCacheInuse:      memStats.MCacheInuse,
		MCacheSys:        memStats.MCacheSys,
		GCSys:            memStats.GCSys,
		OtherSys:         memStats.OtherSys,
		NextGC:           memStats.NextGC,
		LastGC:           memStats.LastGC,
		PauseTotalNs:     memStats.PauseTotalNs,
		NumGC:            memStats.NumGC,
		TotalHeapObj:     memStats.Mallocs,
		TotalHeapObjFree: memStats.Frees,
	}

	// Get current process information
	pid := os.Getpid()

	// Default values
	cwd, _ := os.Getwd() // Get current working directory from os package

	// Create runtime stats info
	runtimeStats := &ds.RuntimeStatsInfo{
		Ts:             time.Now().UnixMilli(),
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

	ctl.SendPacket(pk)
}

// GetStatus returns the current status of the runtime stats collector
func (rc *RuntimeStatsCollector) GetStatus() ds.CollectorStatus {
	cfg := rc.config.Get()
	status := ds.CollectorStatus{
		Running: cfg.Enabled,
	}

	if !cfg.Enabled {
		status.Info = "Disabled in configuration"
	} else {
		status.Info = "Runtime statistics collection active"
		status.CollectDuration = rc.executor.GetLastExecDuration()

		if lastErr := rc.executor.GetLastErr(); lastErr != nil {
			status.Errors = append(status.Errors, lastErr.Error())
		}
	}

	return status
}
