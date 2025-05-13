// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package goroutine

import (
	"bytes"
	"regexp"
	"runtime"
	"slices"
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
	lock                sync.Mutex
	executor            *collector.PeriodicExecutor
	controller          ds.Controller
	config              ds.GoRoutineConfig
	goroutineNames      map[int64]string            // map from goroutine ID to name
	lastGoroutineStacks map[int64]ds.GoRoutineStack // last set of goroutine stacks for delta calculation
	nextSendFull        bool                        // true for full update, false for delta update
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
			goroutineNames:      make(map[int64]string),
			lastGoroutineStacks: make(map[int64]ds.GoRoutineStack),
			nextSendFull:        true, // First send is always a full update
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

// getSendFullAndReset returns the current sendFull value and always sets it to false
func (gc *GoroutineCollector) getSendFullAndReset() bool {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	sendFull := gc.nextSendFull
	gc.nextSendFull = false // Always set to false after getting the value
	return sendFull
}

// SetNextSendFull sets the nextSendFull flag to force a full update on the next dump
func (gc *GoroutineCollector) SetNextSendFull(full bool) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	gc.nextSendFull = full
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

	// Determine if this should be a full or delta update
	sendFull := gc.getSendFullAndReset()

	// Parse the stack data
	goroutineInfo := gc.parseGoroutineStacks(stackData, !sendFull)

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

// computeDeltaStack compares current and last goroutine stack and returns a delta stack
// For delta updates, we always include the goroutine ID, but only include other fields if they've changed
func (gc *GoroutineCollector) computeDeltaStack(id int64, current ds.GoRoutineStack) ds.GoRoutineStack {
	lastStack, exists := gc.lastGoroutineStacks[id]
	if !exists {
		// New goroutine, include all fields
		return current
	}

	// For delta updates, we always include the goroutine ID
	// but only include other fields if they've changed
	deltaStack := ds.GoRoutineStack{
		GoId: id,
	}

	// Only include fields that have changed
	if lastStack.State != current.State {
		deltaStack.State = current.State
	}

	if lastStack.StackTrace != current.StackTrace {
		deltaStack.StackTrace = current.StackTrace
	}

	if lastStack.Name != current.Name {
		deltaStack.Name = current.Name
	}

	// Compare tags using slices.Equal
	tagsChanged := !slices.Equal(lastStack.Tags, current.Tags)

	if tagsChanged {
		deltaStack.Tags = current.Tags
	}

	return deltaStack
}

func (gc *GoroutineCollector) parseGoroutineStacks(stackData []byte, delta bool) *ds.GoroutineInfo {
	goroutineStacks := make([]ds.GoRoutineStack, 0)
	activeGoroutines := make(map[int64]bool)
	currentStacks := make(map[int64]ds.GoRoutineStack)

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

		state := string(matches[2])
		stackTrace := string(bytes.TrimSpace(matches[3]))

		grStack := ds.GoRoutineStack{
			GoId:       id,
			State:      state,
			StackTrace: stackTrace,
		}

		if name, ok := gc.GetGoRoutineName(id); ok {
			grStack.Name, grStack.Tags = utilfn.ParseNameAndTags(name)
		}

		currentStacks[id] = grStack

		// For delta updates, only include changed fields
		if delta {
			deltaStack := gc.computeDeltaStack(id, grStack)
			goroutineStacks = append(goroutineStacks, deltaStack)
		} else {
			// Full update, include all fields
			goroutineStacks = append(goroutineStacks, grStack)
		}
	}

	// Store current stacks for next delta calculation
	gc.lock.Lock()
	gc.lastGoroutineStacks = currentStacks
	gc.lock.Unlock()

	gc.cleanupGoroutineNames(activeGoroutines)
	return &ds.GoroutineInfo{
		Ts:     time.Now().UnixMilli(),
		Count:  len(currentStacks), // Always report the total count
		Stacks: goroutineStacks,
		Delta:  delta,
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
