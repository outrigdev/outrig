// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package goroutine

import (
	"bytes"
	"log"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
)

// MinStackBufferSize is the minimum buffer size for goroutine stack dumps (1MB)
const MinStackBufferSize = 1 << 20
const SingleStackBufferSize = 8 * 1024

const (
	GoState_Init    = 0
	GoState_Running = 1
	GoState_Done    = 2
)

// GoroutineCollector implements the collector.Collector interface for goroutine collection
type GoroutineCollector struct {
	lock                sync.Mutex
	executor            *collector.PeriodicExecutor
	controller          ds.Controller
	config              config.GoRoutineConfig
	goroutineDecls      map[int64]*ds.GoDecl        // map from goroutine ID to GoDecl
	lastGoroutineStacks map[int64]ds.GoRoutineStack // last set of goroutine stacks for delta calculation
	nextSendFull        bool                        // true for full update, false for delta update
	lastStackSize       int                         // last actual stack size (not buffer size)
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
			goroutineDecls:      make(map[int64]*ds.GoDecl),
			lastGoroutineStacks: make(map[int64]ds.GoRoutineStack),
			nextSendFull:        true,               // First send is always a full update
			lastStackSize:       MinStackBufferSize, // Start with minimum stack size estimate
		}
		instance.executor = collector.MakePeriodicExecutor("GoroutineCollector", 1*time.Second, instance.DumpGoroutines)
	})
	return instance
}

// InitCollector initializes the goroutine collector with a controller and configuration
func (gc *GoroutineCollector) InitCollector(controller ds.Controller, cfg any, appRunContext ds.AppRunContext) error {
	gc.controller = controller
	if goConfig, ok := cfg.(config.GoRoutineConfig); ok {
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

func (gc *GoroutineCollector) setGoRoutineDecl(decl *ds.GoDecl) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	if decl.GoId == 0 {
		return
	}
	if gc.goroutineDecls[decl.GoId] != nil {
		// this is weird and should never happen
		return
	}
	gc.goroutineDecls[decl.GoId] = decl
}

// incrementParentSpawnCount increments the NumSpawned counter for a parent goroutine
func (gc *GoroutineCollector) incrementParentSpawnCount(parentGoId int64) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	if parentDecl, ok := gc.goroutineDecls[parentGoId]; ok {
		atomic.AddInt64(&parentDecl.NumSpawned, 1)
	}
}

func (gc *GoroutineCollector) UpdateGoRoutineName(decl *ds.GoDecl, newName string) {
	// we use the gc.Lock to synchronize access to existing decls
	gc.lock.Lock()
	defer gc.lock.Unlock()
	decl.Name = newName
}

func (gc *GoroutineCollector) UpdateGoRoutineTags(decl *ds.GoDecl, newTags []string) {
	// we use the gc.Lock to synchronize access to existing decls
	gc.lock.Lock()
	defer gc.lock.Unlock()
	decl.Tags = newTags
}

func (gc *GoroutineCollector) RecordGoRoutineStart(decl *ds.GoDecl, stack []byte) {
	if len(stack) == 0 {
		stack = gc.dumpSingleStack()
		if len(stack) == 0 {
			return
		}
	}
	// Extract the goroutine ID from the stack trace
	goMatches := goCreationRe.FindSubmatch(stack)
	if len(goMatches) >= 2 {
		goId, err := strconv.ParseInt(string(goMatches[1]), 10, 64)
		if err == nil {
			decl.GoId = goId
		}
	}

	// Extract the parent goroutine ID from the stack trace
	parentMatches := parentGoRe.FindSubmatch(stack)
	if len(parentMatches) >= 2 {
		parentGoId, err := strconv.ParseInt(string(parentMatches[1]), 10, 64)
		if err == nil {
			decl.ParentGoId = parentGoId
			gc.incrementParentSpawnCount(parentGoId)
		}
	}

	gc.setGoRoutineDecl(decl)
}

func (gc *GoroutineCollector) RecordGoRoutineEnd(decl *ds.GoDecl, panicVal any, flush bool) {
	atomic.StoreInt32(&decl.State, GoState_Done)
	endTs := time.Now().UnixMilli()
	atomic.StoreInt64(&decl.EndTs, endTs)
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

func (gc *GoroutineCollector) getLastStackSize() int {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	return gc.lastStackSize
}

func (gc *GoroutineCollector) setLastStackSize(size int) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	gc.lastStackSize = size
}

// dumpAllStacks gets all goroutine stacks, automatically increasing buffer size if needed
// and storing the last successful buffer size for future calls
func (gc *GoroutineCollector) dumpAllStacks() []byte {
	// Get the last stack size and increase by 30% to provide headroom
	bufSize := int(float64(gc.getLastStackSize()) * 1.3)
	if bufSize < MinStackBufferSize {
		bufSize = MinStackBufferSize
	}
	for {
		buf := make([]byte, bufSize)
		stackLen := runtime.Stack(buf, true)
		// If we filled the buffer completely, it's likely truncated, so try again with a larger buffer
		if stackLen == bufSize {
			bufSize *= 2
			continue
		}
		gc.setLastStackSize(stackLen)
		return buf[:stackLen]
	}
}

func (gc *GoroutineCollector) dumpSingleStack() []byte {
	buf := make([]byte, SingleStackBufferSize)
	stackLen := runtime.Stack(buf, false)
	if stackLen == SingleStackBufferSize {
		// truncated, return nil
		return nil
	}
	return buf[:stackLen]
}

// DumpGoroutines dumps all goroutines and sends the information
func (gc *GoroutineCollector) DumpGoroutines() {
	if !global.OutrigEnabled.Load() || gc.controller == nil {
		return
	}
	stackData := gc.dumpAllStacks()
	sendFull := gc.getSendFullAndReset()
	goroutineInfo := gc.parseGoroutineStacks(stackData, !sendFull)
	pk := &ds.PacketType{
		Type: ds.PacketTypeGoroutine,
		Data: goroutineInfo,
	}
	gc.controller.SendPacket(pk)
}

// GetGoRoutineName gets the name for a goroutine
func (gc *GoroutineCollector) GetGoRoutineName(goId int64) (string, bool) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	decl, ok := gc.goroutineDecls[goId]
	if !ok || decl.Name == "" {
		return "", false
	}
	return decl.Name, true
}

// GetGoRoutineDecl gets the declaration for a goroutine by ID
// Returns nil if no declaration exists for the given ID
func (gc *GoroutineCollector) GetGoRoutineDecl(goId int64) *ds.GoDecl {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	decl, ok := gc.goroutineDecls[goId]
	if !ok {
		return nil
	}
	return decl
}

var startRe = regexp.MustCompile(`(?m)^goroutine\s+\d+`)
var stackRe = regexp.MustCompile(`goroutine (\d+) \[([^\]]+)\].*\n((?s).*)`)
var goCreationRe = regexp.MustCompile(`goroutine (\d+) \[([^\]]+)\]`)
var parentGoRe = regexp.MustCompile(`created by .* in goroutine (\d+)`)

// computeDeltaStack compares current and last goroutine stack and returns a delta stack
// For delta updates, we start with a full copy of current and clear fields that haven't changed
func (gc *GoroutineCollector) computeDeltaStack(id int64, current ds.GoRoutineStack) (ds.GoRoutineStack, bool) {
	lastStack, exists := gc.lastGoroutineStacks[id]
	if !exists {
		// New goroutine, include all fields
		return current, false
	}
	// For delta updates, start with a full copy of current and clear fields that haven't changed
	deltaStack := current
	sameStack := true
	if lastStack.State == current.State {
		deltaStack.State = ""
	}
	if lastStack.StackTrace == current.StackTrace {
		deltaStack.StackTrace = ""
	} else {
		sameStack = false
	}
	if lastStack.Name == current.Name {
		deltaStack.Name = ""
	}
	if slices.Equal(lastStack.Tags, current.Tags) {
		deltaStack.Tags = nil
	}
	return deltaStack, sameStack
}

func (gc *GoroutineCollector) parseGoroutineStacks(stackData []byte, delta bool) *ds.GoroutineInfo {
	goroutineStacks := make([]ds.GoRoutineStack, 0)
	activeGoroutines := make(map[int64]bool)
	currentStacks := make(map[int64]ds.GoRoutineStack)

	startIndices := startRe.FindAllIndex(stackData, -1)
	numSameStack := 0
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

		// Record this goroutine if we haven't seen it before or update its poll timestamps
		gc.recordPolledGoroutine(id, goroutineData)

		grStack := ds.GoRoutineStack{
			GoId:       id,
			State:      state,
			StackTrace: stackTrace,
		}

		if decl, ok := gc.goroutineDecls[id]; ok && decl.Name != "" {
			grStack.Name = decl.Name
			grStack.Tags = decl.Tags
		}

		currentStacks[id] = grStack

		// For delta updates, only include changed fields
		if delta {
			deltaStack, sameStack := gc.computeDeltaStack(id, grStack)
			if sameStack {
				numSameStack++
			}
			goroutineStacks = append(goroutineStacks, deltaStack)
		} else {
			// Full update, include all fields
			goroutineStacks = append(goroutineStacks, grStack)
		}
	}

	log.Printf("GoroutineCollector: %d goroutines, %d same stacks", len(activeGoroutines), numSameStack)

	// Store current stacks for next delta calculation
	gc.setLastGoroutineStacksAndCleanupNames(currentStacks)
	return &ds.GoroutineInfo{
		Ts:     time.Now().UnixMilli(),
		Count:  len(currentStacks), // Always report the total count
		Stacks: goroutineStacks,
		Delta:  delta,
	}
}

func (gc *GoroutineCollector) setLastGoroutineStacksAndCleanupNames(stacks map[int64]ds.GoRoutineStack) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	gc.lastGoroutineStacks = stacks

	// Map to track goroutines we want to keep (active ones and their ancestors)
	keepMap := make(map[int64]bool)
	var keepStack []int64 // for DFS processing	of ancestors
	// seed the stack with all active goroutines
	for id := range stacks {
		keepMap[id] = true
		keepStack = append(keepStack, id)
	}

	// Process the stack to find all ancestors
	for len(keepStack) > 0 {
		// Pop from stack
		n := len(keepStack) - 1
		currentID := keepStack[n]
		keepStack = keepStack[:n]

		// Get the parent ID if available
		decl, ok := gc.goroutineDecls[currentID]
		if ok && decl.ParentGoId != 0 {
			parentID := decl.ParentGoId
			// If we haven't processed this parent yet, add it to keep map and stack
			if !keepMap[parentID] {
				keepMap[parentID] = true
				keepStack = append(keepStack, parentID)
			}
		}
	}

	// Remove declarations for goroutines that are not in the keep map
	for id := range gc.goroutineDecls {
		if !keepMap[id] {
			delete(gc.goroutineDecls, id)
		}
	}
}

// recordPolledGoroutine records information about a goroutine discovered during polling
// It sets the parent goroutine ID if it's the first time we see the goroutine
// and updates FirstPollTs and LastPollTs appropriately
func (gc *GoroutineCollector) recordPolledGoroutine(goId int64, goroutineData []byte) {
	now := time.Now().UnixMilli()
	decl := gc.GetGoRoutineDecl(goId)
	if decl != nil {
		atomic.CompareAndSwapInt64(&decl.FirstPollTs, 0, now)
		atomic.StoreInt64(&decl.LastPollTs, now)
		return
	}
	// First time we've seen this goroutine
	decl = &ds.GoDecl{
		GoId:        goId,
		State:       GoState_Running,
		FirstPollTs: now,
		LastPollTs:  now,
	}
	gc.RecordGoRoutineStart(decl, goroutineData)
}
