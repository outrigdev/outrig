// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package goroutine

import (
	"bytes"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// GoroutineCollector implements the collector.Collector interface for goroutine collection
type GoroutineCollector struct {
	lock           sync.Mutex
	executor       *collector.PeriodicExecutor
	controller     ds.Controller
	config         ds.GoRoutineConfig
	goroutineNames map[int64]string // map from goroutine ID to name
}

// CollectorName returns the unique name of the collector
func (gc *GoroutineCollector) CollectorName() string {
	return "goroutine"
}

// singleton instance
var instance *GoroutineCollector
var instanceOnce sync.Once

// GetInstance returns the singleton instance of GoroutineCollector
func GetInstance() *GoroutineCollector {
	instanceOnce.Do(func() {
		instance = &GoroutineCollector{
			goroutineNames: make(map[int64]string),
		}
		instance.executor = collector.MakePeriodicExecutor("GoroutineCollector", 1*time.Second, instance.DumpGoroutines)
	})
	return instance
}

// InitCollector initializes the goroutine collector with a controller and configuration
func (gc *GoroutineCollector) InitCollector(controller ds.Controller, config any, appRunContext ds.AppRunContext) error {
	gc.controller = controller
	if goConfig, ok := config.(ds.GoRoutineConfig); ok {
		gc.config = goConfig
	}
	return nil
}

// Enable is called when the collector should start collecting data
func (gc *GoroutineCollector) Enable() {
	gc.executor.Enable()
}

func (gc *GoroutineCollector) Disable() {
	gc.executor.Disable()
}

// DumpGoroutines dumps all goroutines and sends the information
func (gc *GoroutineCollector) DumpGoroutines() {
	if !global.OutrigEnabled.Load() || gc.controller == nil {
		return
	}

	// Get all goroutine stacks
	buf := make([]byte, 1<<20)
	stackLen := runtime.Stack(buf, true)
	stackData := buf[:stackLen]

	// Parse the stack data
	goroutineInfo := gc.parseGoroutineStacks(stackData)

	// Send the goroutine packet
	pk := &ds.PacketType{
		Type: ds.PacketTypeGoroutine,
		Data: goroutineInfo,
	}

	gc.controller.SendPacket(pk)
}

// SetGoRoutineName sets a name for a goroutine
func (gc *GoroutineCollector) SetGoRoutineName(goId int64, name string) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	gc.goroutineNames[goId] = name
}

// GetGoRoutineName gets the name for a goroutine
func (gc *GoroutineCollector) GetGoRoutineName(goId int64) (string, bool) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	name, ok := gc.goroutineNames[goId]
	return name, ok
}

var startRe = regexp.MustCompile(`(?m)^goroutine\s+\d+`)
var stackRe = regexp.MustCompile(`goroutine (\d+) \[([^\]]+)\].*\n((?s).*)`)

func (gc *GoroutineCollector) parseGoroutineStacks(stackData []byte) *ds.GoroutineInfo {
	goroutineStacks := make([]ds.GoRoutineStack, 0)
	activeGoroutines := make(map[int64]bool)

	startIndices := startRe.FindAllIndex(stackData, -1)
	for i, startIdx := range startIndices {
		start := startIdx[0]
		end := len(stackData)
		if i+1 < len(startIndices) {
			end = startIndices[i+1][0]
		}
		goroutineData := stackData[start:end]
		matches := stackRe.FindSubmatch(goroutineData)
		if len(matches) < 4 {
			continue
		}
		id, _ := strconv.ParseInt(string(matches[1]), 10, 64) // this is safe because the regex guarantees a number
		activeGoroutines[id] = true
		grStack := ds.GoRoutineStack{
			GoId:       id,
			State:      string(matches[2]),
			StackTrace: string(bytes.TrimSpace(matches[3])),
		}
		if name, ok := gc.GetGoRoutineName(id); ok {
			grStack.Name, grStack.Tags = utilfn.ParseNameAndTags(name)
		}
		goroutineStacks = append(goroutineStacks, grStack)
	}

	gc.cleanupGoroutineNames(activeGoroutines)
	return &ds.GoroutineInfo{
		Ts:     time.Now().UnixMilli(),
		Count:  len(goroutineStacks),
		Stacks: goroutineStacks,
	}
}

// cleanupGoroutineNames removes names for goroutines that are no longer active
func (gc *GoroutineCollector) cleanupGoroutineNames(activeGoroutines map[int64]bool) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	// Remove names for goroutines that no longer exist
	for id := range gc.goroutineNames {
		if !activeGoroutines[id] {
			delete(gc.goroutineNames, id)
		}
	}
}
