// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package goroutine

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilds"
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
	config              *utilds.SetOnceConfig[config.GoRoutineConfig]
	executor            *collector.PeriodicExecutor
	goroutineDecls      map[int64]*ds.GoDecl        // map from goroutine ID to GoDecl
	lastGoroutineStacks map[int64]ds.GoRoutineStack // last set of goroutine stacks for delta calculation
	nextSendFull        bool                        // true for full update, false for delta update
	lastStackSize       int                         // last actual stack size (not buffer size)
	updatedDecls        []ds.GoDecl                 // declarations updated since last send
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
			config:              utilds.NewSetOnceConfig(config.DefaultConfig().GoRoutineConfig),
			goroutineDecls:      make(map[int64]*ds.GoDecl),
			lastGoroutineStacks: make(map[int64]ds.GoRoutineStack),
			nextSendFull:        true,               // First send is always a full update
			lastStackSize:       MinStackBufferSize, // Start with minimum stack size estimate
		}
		instance.executor = collector.MakePeriodicExecutor("GoroutineCollector", 1*time.Second, instance.DumpGoroutines)
	})
	return instance
}

func Init(cfg *config.GoRoutineConfig) error {
	gc := GetInstance()
	if gc.executor.IsEnabled() {
		return fmt.Errorf("goroutine collector is already initialized")
	}
	ok := gc.config.SetOnce(cfg)
	if !ok {
		return fmt.Errorf("goroutine collector configuration already set")
	}
	return nil
}


// InitCollector initializes the goroutine collector with a controller and configuration
func (gc *GoroutineCollector) InitCollector(controller ds.Controller, cfg any) error {
	if goConfig, ok := cfg.(config.GoRoutineConfig); ok {
		gc.config.SetOnce(&goConfig)
	}
	return nil
}

// Enable is called when the collector should start collecting data
func (gc *GoroutineCollector) Enable() {
	cfg := gc.config.Get()
	if !cfg.Enabled {
		return
	}
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

	// Add to updated declarations (make a copy to avoid reference issues)
	declCopy := *decl
	gc.updatedDecls = append(gc.updatedDecls, declCopy)
}

// incrementParentSpawnCount increments the NumSpawned counter for a parent goroutine
func (gc *GoroutineCollector) incrementParentSpawnCount(parentGoId int64) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	if parentDecl, ok := gc.goroutineDecls[parentGoId]; ok {
		atomic.AddInt64(&parentDecl.NumSpawned, 1)

		// Add to updated declarations (make a copy to avoid reference issues)
		declCopy := *parentDecl
		gc.updatedDecls = append(gc.updatedDecls, declCopy)
	}
}

func (gc *GoroutineCollector) UpdateGoRoutineName(decl *ds.GoDecl, newName string) {
	// we use the gc.Lock to synchronize access to existing decls
	gc.lock.Lock()
	defer gc.lock.Unlock()
	decl.Name = newName

	// Add to updated declarations (make a copy to avoid reference issues)
	declCopy := *decl
	gc.updatedDecls = append(gc.updatedDecls, declCopy)
}

func (gc *GoroutineCollector) UpdateGoRoutineTags(decl *ds.GoDecl, newTags []string) {
	// we use the gc.Lock to synchronize access to existing decls
	gc.lock.Lock()
	defer gc.lock.Unlock()
	decl.Tags = newTags

	// Add to updated declarations (make a copy to avoid reference issues)
	declCopy := *decl
	gc.updatedDecls = append(gc.updatedDecls, declCopy)
}

func (gc *GoroutineCollector) UpdateGoRoutinePkg(decl *ds.GoDecl, newPkg string) {
	// we use the gc.Lock to synchronize access to existing decls
	gc.lock.Lock()
	defer gc.lock.Unlock()
	decl.Pkg = newPkg

	// Add to updated declarations (make a copy to avoid reference issues)
	declCopy := *decl
	gc.updatedDecls = append(gc.updatedDecls, declCopy)
}

func (gc *GoroutineCollector) setInitialGoDeclInfo(decl *ds.GoDecl, stack []byte) {
	if decl.GoId != 0 && decl.ParentGoId != 0 && decl.Pkg != "" {
		return // all fields are already set
	}
	if len(stack) == 0 {
		stack = gc.dumpSingleStack()
		if len(stack) == 0 {
			return
		}
	}

	// Extract the goroutine ID from the stack trace
	if decl.GoId == 0 {
		goMatches := goCreationRe.FindSubmatch(stack)
		if len(goMatches) >= 2 {
			goId, err := strconv.ParseInt(string(goMatches[1]), 10, 64)
			if err == nil {
				decl.GoId = goId
			}
		}
	}

	// Extract the parent goroutine ID from the stack trace
	if decl.ParentGoId == 0 {
		parentMatches := parentGoRe.FindSubmatch(stack)
		if len(parentMatches) >= 2 {
			parentGoId, err := strconv.ParseInt(string(parentMatches[1]), 10, 64)
			if err == nil {
				decl.ParentGoId = parentGoId
			}
		}
	}

	// Extract the package name from the stack trace
	if decl.Pkg == "" {
		createdByMatches := createdByRe.FindSubmatch(stack)
		if len(createdByMatches) >= 2 {
			funcName := string(createdByMatches[1])
			decl.Pkg = extractPackage(funcName)
		}
	}
}

func (gc *GoroutineCollector) RecordGoRoutineStart(decl *ds.GoDecl, stack []byte) {
	gc.setInitialGoDeclInfo(decl, stack)
	if decl.ParentGoId != 0 {
		gc.incrementParentSpawnCount(decl.ParentGoId)
	}
	gc.setGoRoutineDecl(decl)
}

func (gc *GoroutineCollector) RecordGoRoutineEnd(decl *ds.GoDecl, panicVal any, flush bool) {
	atomic.StoreInt32(&decl.State, GoState_Done)
	endTs := time.Now().UnixMilli()
	atomic.StoreInt64(&decl.EndTs, endTs)

	// Add to updated declarations (make a copy to avoid reference issues)
	gc.lock.Lock()
	defer gc.lock.Unlock()
	declCopy := *decl
	gc.updatedDecls = append(gc.updatedDecls, declCopy)
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

// OnNewConnection is called when a new connection is established
func (gc *GoroutineCollector) OnNewConnection() {
	gc.SetNextSendFull(true)
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

// getDeclList returns the list of declarations to send
// For full updates, it returns all declarations
// For delta updates, it returns only the updated declarations
func (gc *GoroutineCollector) getDeclList(delta bool) []ds.GoDecl {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	if !delta {
		// For full updates, return all declarations
		declList := make([]ds.GoDecl, 0, len(gc.goroutineDecls))
		for _, decl := range gc.goroutineDecls {
			declList = append(declList, *decl)
		}
		// Clear updated declarations after a full update
		gc.updatedDecls = nil
		return declList
	}

	// For delta updates, return only the updated declarations
	declList := gc.updatedDecls
	gc.updatedDecls = nil
	return declList
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
	if !global.OutrigEnabled.Load() {
		return
	}
	ctl := global.GetController()
	if ctl == nil {
		return
	}
	stackData := gc.dumpAllStacks()
	sendFull := gc.getSendFullAndReset()
	goroutineInfo := gc.parseGoroutineStacks(stackData, !sendFull)
	pk := &ds.PacketType{
		Type: ds.PacketTypeGoroutine,
		Data: goroutineInfo,
	}
	ctl.SendPacket(pk)
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
var createdByRe = regexp.MustCompile(`created by\s+(\S+)`)

// extractPackage extracts the package name from a stack trace function name
func extractPackage(funcName string) string {
	lastSlash := strings.LastIndex(funcName, "/")
	if lastSlash == -1 {
		// No slash, look for first dot
		if dot := strings.Index(funcName, "."); dot != -1 {
			return funcName[:dot]
		}
		return funcName
	}

	// Find first dot after last slash
	remaining := funcName[lastSlash:]
	if dot := strings.Index(remaining, "."); dot != -1 {
		return funcName[:lastSlash+dot]
	}
	return funcName
}

// computeDeltaStack compares current and last goroutine stack and returns a delta stack
// If all fields are the same, returns a stack with only GoId and Same=true
// Otherwise returns the full current stack with Same=false
func (gc *GoroutineCollector) computeDeltaStack(id int64, current ds.GoRoutineStack) ds.GoRoutineStack {
	lastStack, exists := gc.lastGoroutineStacks[id]
	if !exists {
		// New goroutine, include all fields
		return current
	}

	// Check if all fields are the same
	allSame := lastStack.State == current.State &&
		lastStack.StackTrace == current.StackTrace &&
		lastStack.Name == current.Name &&
		slices.Equal(lastStack.Tags, current.Tags)

	if allSame {
		// All fields are the same, clear all fields and set Same
		return ds.GoRoutineStack{
			GoId: current.GoId,
			Same: true,
		}
	}

	// Fields differ, send all fields and don't set Same
	return current
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

		if decl, ok := gc.goroutineDecls[id]; ok {
			if decl.Name != "" {
				grStack.Name = decl.Name
				grStack.Tags = decl.Tags
			}
			// Patch the stack trace to replace Outrig SDK frames with real creator
			grStack.StackTrace = patchCreatedByStack(decl, grStack.StackTrace)
		}

		currentStacks[id] = grStack

		// For delta updates, only include changed fields
		if delta {
			deltaStack := gc.computeDeltaStack(id, grStack)
			if deltaStack.Same {
				numSameStack++
			}
			goroutineStacks = append(goroutineStacks, deltaStack)
		} else {
			// Full update, include all fields
			goroutineStacks = append(goroutineStacks, grStack)
		}
	}

	// Store current stacks for next delta calculation
	gc.setLastGoroutineStacksAndCleanupNames(currentStacks)
	return &ds.GoroutineInfo{
		Ts:     time.Now().UnixMilli(),
		Count:  len(currentStacks), // Always report the total count
		Stacks: goroutineStacks,
		Delta:  delta,
		Decls:  gc.getDeclList(delta),
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

func NewRunningGoDecl(goId int64) *ds.GoDecl {
	// Create a new GoDecl with the given ID and default values
	decl := &ds.GoDecl{
		GoId:  goId,
		State: GoState_Running,
	}
	if goId == 1 {
		decl.Name = "main" // Special case for goroutine 1
	}
	return decl
}

// recordPolledGoroutine records information about a goroutine discovered during polling
// It sets the parent goroutine ID if it's the first time we see the goroutine
// and updates FirstPollTs and LastPollTs appropriately
func (gc *GoroutineCollector) recordPolledGoroutine(goId int64, goroutineData []byte) {
	now := time.Now().UnixMilli()
	decl := gc.GetGoRoutineDecl(goId)
	if decl != nil {
		// Check if FirstPollTs was updated (was 0 before)
		wasFirstPollUpdated := atomic.CompareAndSwapInt64(&decl.FirstPollTs, 0, now)

		// Always update LastPollTs
		atomic.StoreInt64(&decl.LastPollTs, now)

		// Only add to updated declarations if something other than LastPollTs changed
		if wasFirstPollUpdated {
			gc.lock.Lock()
			declCopy := *decl
			gc.updatedDecls = append(gc.updatedDecls, declCopy)
			gc.lock.Unlock()
		}
		return
	}

	// First time we've seen this goroutine
	decl = NewRunningGoDecl(goId)
	decl.FirstPollTs = now
	decl.LastPollTs = now
	gc.RecordGoRoutineStart(decl, goroutineData)
}

// getMonitoringCounts returns the current monitoring counts with proper locking
func (gc *GoroutineCollector) getMonitoringCounts() (int, int) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	return len(gc.lastGoroutineStacks), len(gc.goroutineDecls)
}

// GetStatus returns the current status of the goroutine collector
func (gc *GoroutineCollector) GetStatus() ds.CollectorStatus {
	cfg := gc.config.Get()
	status := ds.CollectorStatus{
		Running: cfg.Enabled,
	}
	if !cfg.Enabled {
		status.Info = "Disabled in configuration"
	} else {
		activeGoroutines, totalDecls := gc.getMonitoringCounts()
		status.Info = fmt.Sprintf("Monitoring %d active goroutines, %d total declarations", activeGoroutines, totalDecls)
		status.CollectDuration = gc.executor.GetLastExecDuration()

		if lastErr := gc.executor.GetLastErr(); lastErr != nil {
			status.Errors = append(status.Errors, lastErr.Error())
		}
	}

	return status
}

// patchCreatedByStack patches the stack trace to replace Outrig SDK frames with the real creator
func patchCreatedByStack(decl *ds.GoDecl, stack string) string {
	if decl.RealCreatedBy == "" {
		return stack
	}

	lines := strings.Split(stack, "\n")

	// Check if we have at least 4 lines and the last 4 lines match the expected pattern:
	// Line N-3: github.com/outrigdev/outrig.(*GoRoutine).Run.func1()
	// Line N-2: 	/path/to/outrig.go:537 +0xc8
	// Line N-1: created by github.com/outrigdev/outrig.(*GoRoutine).Run in goroutine X
	// Line N:   	/path/to/outrig.go:519 +0x110

	if len(lines) < 4 {
		return stack
	}

	lastIdx := len(lines) - 1

	// Check the pattern from the end
	if strings.Contains(lines[lastIdx-3], "github.com/outrigdev/outrig.(*GoRoutine).Run") &&
		strings.Contains(lines[lastIdx-1], "created by github.com/outrigdev/outrig.(*GoRoutine).Run") {

		// Pattern matches, remove the last 4 lines and replace with RealCreatedBy
		lines = lines[:lastIdx-3]
		lines = append(lines, decl.RealCreatedBy)

		return strings.Join(lines, "\n")
	}

	// Pattern doesn't match, return original stack
	return stack
}
