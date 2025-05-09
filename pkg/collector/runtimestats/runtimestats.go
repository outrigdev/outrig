// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runtimestats

import (
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
)

// RuntimeStatsCollector implements the collector.Collector interface for runtime stats collection
type RuntimeStatsCollector struct {
	lock       sync.Mutex
	executor   *collector.PeriodicExecutor
	controller ds.Controller
	config     ds.RuntimeStatsConfig
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
		instance.executor = collector.MakePeriodicExecutor("RuntimeStatsCollector", 1*time.Second, instance.CollectRuntimeStats)
	})
	return instance
}

// InitCollector initializes the runtime stats collector with a controller and configuration
func (rc *RuntimeStatsCollector) InitCollector(controller ds.Controller, config any, arCtx ds.AppRunContext) error {
	rc.controller = controller
	if statsConfig, ok := config.(ds.RuntimeStatsConfig); ok {
		rc.config = statsConfig
	}
	return nil
}

// Enable is called when the collector should start collecting data
func (rc *RuntimeStatsCollector) Enable() {
	rc.executor.Enable()
}

// Disable stops the collector
func (rc *RuntimeStatsCollector) Disable() {
	rc.executor.Disable()
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

	rc.controller.SendPacket(pk)
}
