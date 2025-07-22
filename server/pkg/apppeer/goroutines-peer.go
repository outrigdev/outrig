// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/logutil"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/stacktrace"
)

const GoRoutineStackBufferSize = 600
const GoRoutinePruneThreshold = 600 // Number of iterations after which inactive goroutines are pruned

// GoRoutine represents a goroutine with its stack traces
type GoRoutine struct {
	GoId                int64
	Name                string
	Tags                []string
	CreatedByGoId       int64                // ID of the goroutine that created this one
	CreatedByFrame      *rpctypes.StackFrame // Frame information for the creation point
	StackTraces         *utilds.CirBuf[ds.GoRoutineStack]
	TimeSpan            rpctypes.TimeSpan // Time span when the goroutine was active
	LastActiveIteration int64             // Iteration when the goroutine was last active
	Decl                *ds.GoDecl        // Declaration information for this goroutine
}

// GoRoutinePeer manages goroutines for an AppRunPeer
type GoRoutinePeer struct {
	goRoutines        *utilds.SyncMap[int64, GoRoutine]
	activeGoRoutines  map[int64]bool                                  // Tracks currently running goroutines
	timeSpanMap       *utilds.VersionedMap[uint64, rpctypes.TimeSpan] // Tracks TimeSpan changes for goroutines
	lock              sync.RWMutex                                    // Lock for synchronizing goroutine operations
	currentIteration  int64                                           // Current iteration counter
	maxGoId           int64                                           // Maximum goroutine ID seen
	hasSeenFullUpdate bool                                            // Flag to track if we've seen a full update
	appRunId          string                                          // ID of the app run this peer belongs to
	timeSpan          rpctypes.TimeSpan                               // Time range for goroutine collections
	timeAligner       *utilds.TimeSampleAligner                       // Aligns goroutine stack timestamps to logical indices
	droppedCount      atomic.Int64                                    // Count of goroutines dropped during pruning (synchronized with atomic operations)
}

// GoRoutinesAtTimestampResult contains the result of GetParsedGoRoutinesAtTimestamp
type GoRoutinesAtTimestampResult struct {
	GoRoutines         []rpctypes.ParsedGoRoutine
	TotalCount         int   // Total count of all goroutines (not filtered by activeOnly)
	TotalNonOutrig     int   // Total count of non-outrig goroutines (not filtered by activeOnly)
	EffectiveTimestamp int64 // The timestamp that was actually used for the query
}

// mergeGoRoutineStacks combines a base stack with a delta stack to create a complete stack
// If deltaStack.Same is true, returns the base stack unchanged
// Otherwise, returns the delta stack (which contains all current field values)
func mergeGoRoutineStacks(baseStack, deltaStack ds.GoRoutineStack) ds.GoRoutineStack {
	if deltaStack.Same {
		// All fields are the same, but update the timestamp
		// Safe to modify baseStack since it's passed by value
		baseStack.Ts = deltaStack.Ts
		return baseStack
	}

	// Fields have changed, return the delta stack which contains all current values
	return deltaStack
}

// MakeGoRoutinePeer creates a new GoRoutinePeer instance
func MakeGoRoutinePeer(appRunId string) *GoRoutinePeer {
	return &GoRoutinePeer{
		goRoutines:       utilds.MakeSyncMap[int64, GoRoutine](),
		activeGoRoutines: make(map[int64]bool),
		timeSpanMap:      utilds.MakeVersionedMap[uint64, rpctypes.TimeSpan](),
		currentIteration: 0,
		maxGoId:          0,
		appRunId:         appRunId,
		timeAligner:      utilds.MakeTimeSampleAligner(GoRoutineStackBufferSize),
	}
}

// getOrCreateGoRoutine gets or creates a goroutine with the given ID and timestamp
// Returns the goroutine and a boolean indicating if it was found (true) or created (false)
func (gp *GoRoutinePeer) getOrCreateGoRoutine(goId int64, timestamp int64, logicalTime int) (GoRoutine, bool) {
	goroutine, wasFound := gp.goRoutines.GetOrCreate(goId, func() GoRoutine {
		return GoRoutine{
			GoId:        goId,
			StackTraces: utilds.MakeCirBuf[ds.GoRoutineStack](GoRoutineStackBufferSize),
			TimeSpan: rpctypes.TimeSpan{
				Start:    timestamp,
				StartIdx: logicalTime,
				End:      -1, // Keep End at -1 for ongoing goroutines per TimeSpan spec
				EndIdx:   -1, // Keep EndIdx at -1 for ongoing goroutines
			},
			LastActiveIteration: gp.currentIteration,
		}
	})

	// If this was a new goroutine, update the timespan map
	if !wasFound {
		gp.updateTimeSpanMap(goId, goroutine)
	}

	return goroutine, wasFound
}

// updateTimeSpanMap updates the versioned map with the current timespan for a goroutine
// only if the timespan has actually changed
func (gp *GoRoutinePeer) updateTimeSpanMap(goId int64, goroutine GoRoutine) {
	newTimeSpan := goroutine.TimeSpan
	shouldUpdate := true

	// Check if we already have a timespan for this goroutine
	if existingTimeSpan, _, exists := gp.timeSpanMap.Get(uint64(goId)); exists {
		shouldUpdate = existingTimeSpan != newTimeSpan
	}

	if shouldUpdate {
		gp.timeSpanMap.Set(uint64(goId), newTimeSpan)
	}
}

// ProcessGoroutineStacks processes goroutine stacks from a packet
func (gp *GoRoutinePeer) ProcessGoroutineStacks(info ds.GoroutineInfo) {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	// Increment the iteration counter
	gp.currentIteration++

	activeGoroutines := make(map[int64]bool)
	timestamp := info.Ts
	isDelta := info.Delta

	// Add sample to time aligner
	logicalTime, err := gp.timeAligner.AddSample(timestamp)
	if err != nil {
		fmt.Printf("WARNING: [AppRun: %s] TimeSampleAligner error: %v\n", gp.appRunId, err)
		return // Drop this sample
	}
	firstSampleTs := gp.timeAligner.GetFirstTimestamp()

	// Set the version to match the logical time
	gp.timeSpanMap.SetVersion(int64(logicalTime))

	// Update the overall TimeSpan for goroutine collections
	if gp.timeSpan.Start == 0 || timestamp < gp.timeSpan.Start {
		gp.timeSpan.Start = timestamp
		gp.timeSpan.StartIdx = logicalTime
	}
	if timestamp > gp.timeSpan.End {
		gp.timeSpan.End = timestamp
		gp.timeSpan.EndIdx = logicalTime
	}

	// If this is a delta update but we haven't seen a full update yet, ignore it
	if isDelta && !gp.hasSeenFullUpdate {
		fmt.Printf("WARNING: [AppRun: %s] Ignoring delta update because no full update has been seen yet\n", gp.appRunId)
		return
	}

	// If this is a full update, mark that we've seen one
	if !isDelta {
		gp.hasSeenFullUpdate = true
	}

	// Process goroutine declarations first
	for _, decl := range info.Decls {
		goId := decl.GoId
		if goId == 0 {
			continue // Skip declarations without a valid GoId
		}

		// Update maxGoId if we see a larger goroutine ID
		if goId > gp.maxGoId {
			gp.maxGoId = goId
		}

		goroutine, _ := gp.getOrCreateGoRoutine(goId, timestamp, logicalTime)

		// Store the declaration information
		goroutine.Decl = &decl

		// Update fields from declaration if available
		if decl.Name != "" {
			goroutine.Name = decl.Name
		}
		if len(decl.Tags) > 0 {
			goroutine.Tags = decl.Tags
		}

		// If GoDecl has StartTs set, this is the exact start time for the goroutine
		if decl.StartTs != 0 {
			if decl.StartTs < firstSampleTs {
				decl.StartTs = firstSampleTs
			}
			goroutine.TimeSpan.Start = decl.StartTs
			goroutine.TimeSpan.StartIdx = gp.timeAligner.GetLogicalTimeFromRealTimestamp(decl.StartTs)
			goroutine.TimeSpan.Exact = true
		}

		// If GoDecl has EndTs set, this is the exact end time for the goroutine
		if decl.EndTs != 0 {
			if decl.EndTs < firstSampleTs {
				decl.EndTs = firstSampleTs
			}
			goroutine.TimeSpan.End = decl.EndTs
			goroutine.TimeSpan.EndIdx = gp.timeAligner.GetLogicalTimeFromRealTimestamp(decl.EndTs)
		}

		// If GoDecl has RealCreatedBy set, extract created by information
		if decl.RealCreatedBy != "" && goroutine.CreatedByFrame == nil {
			gp.parseRealCreatedBy(&decl, &goroutine)
		}

		gp.goRoutines.Set(goId, goroutine)
		// Update timespan map since declaration info affects timespan calculation
		gp.updateTimeSpanMap(goId, goroutine)
	}

	// Process goroutine stacks
	for _, stack := range info.Stacks {
		goId := stack.GoId

		activeGoroutines[goId] = true

		// Update maxGoId if we see a larger goroutine ID
		if goId > gp.maxGoId {
			gp.maxGoId = goId
		}

		goroutine, wasFound := gp.getOrCreateGoRoutine(goId, timestamp, logicalTime)

		// Update the last active iteration
		goroutine.LastActiveIteration = gp.currentIteration

		// Update fields based on the stack data
		if stack.Name != "" {
			goroutine.Name = stack.Name
		}
		if len(stack.Tags) > 0 {
			goroutine.Tags = stack.Tags
		}

		// Set CreatedByGoId and CreatedByFrame from the first stack trace we see for this goroutine
		if goroutine.CreatedByGoId == 0 && goroutine.CreatedByFrame == nil && stack.StackTrace != "" {
			// Parse the stack trace to extract creation information
			if parsedGoRoutine, err := stacktrace.ParseGoRoutineStackTrace(stack.StackTrace, "", stack.GoId, stack.State); err == nil {
				if parsedGoRoutine.CreatedByGoId != 0 {
					goroutine.CreatedByGoId = parsedGoRoutine.CreatedByGoId
				}
				if parsedGoRoutine.CreatedByFrame != nil {
					goroutine.CreatedByFrame = parsedGoRoutine.CreatedByFrame
				}
			}
		}

		// Handle stack trace updates based on whether it's a delta update
		if isDelta && stack.Same {
			// Delta updates need a base stack to merge with
			var exists bool
			var lastStack ds.GoRoutineStack
			if wasFound {
				lastStack, _, exists = goroutine.StackTraces.GetLast()
			}
			if exists {
				completeStack := mergeGoRoutineStacks(lastStack, stack)
				goroutine.StackTraces.WriteAt(completeStack, logicalTime)
			} else {
				logKey := fmt.Sprintf("goroutine-nodeltaupdate-%s", gp.appRunId)
				logutil.LogfOnce(logKey, "WARNING: [AppRun: %s] Delta update received for goroutine %d with no last stack\n", gp.appRunId, goId)
			}
		} else {
			// full updates write the stack directly
			goroutine.StackTraces.WriteAt(stack, logicalTime)
		}

		gp.goRoutines.Set(goId, goroutine)
		// Update timespan map since LastSeen was updated
		gp.updateTimeSpanMap(goId, goroutine)
	}

	// Check for goroutines that should be marked as ended:
	// 1. Previously active goroutines no longer in current active set
	// 2. Goroutines with StartTs before current timestamp but not in current active set
	allGoRoutineIds := gp.goRoutines.Keys()
	for _, goId := range allGoRoutineIds {
		if activeGoroutines[goId] {
			continue
		}
		goroutine, exists := gp.goRoutines.GetEx(goId)
		if !exists {
			continue
		}
		if goroutine.TimeSpan.End != -1 {
			continue
		}
		// Mark as ended if StartTs is before current timestamp (not in future due to clock skew)
		if goroutine.TimeSpan.Start > 0 && goroutine.TimeSpan.Start < timestamp {
			goroutine.TimeSpan.End = timestamp
			goroutine.TimeSpan.EndIdx = logicalTime
			gp.goRoutines.Set(goId, goroutine)
			gp.updateTimeSpanMap(goId, goroutine)
		}
	}

	// Always update the active goroutines map
	// Delta updates include all active goroutines, not just the ones that have changed
	gp.activeGoRoutines = activeGoroutines

	gp.pruneOldGoroutines()
}

// pruneOldGoroutines removes goroutines that haven't been active for more than GoRoutinePruneThreshold iterations
func (gp *GoRoutinePeer) pruneOldGoroutines() {
	// Calculate the cutoff iteration
	cutoffIteration := gp.currentIteration - GoRoutinePruneThreshold

	// Only prune if we have enough iterations
	if cutoffIteration <= 0 {
		return
	}

	// Get all goroutine IDs
	keys := gp.goRoutines.Keys()

	// Check each goroutine
	for _, key := range keys {
		goroutine, exists := gp.goRoutines.GetEx(key)
		if !exists {
			continue
		}

		// If the goroutine hasn't been active for more than GoRoutinePruneThreshold iterations, remove it
		if goroutine.LastActiveIteration < cutoffIteration {
			gp.goRoutines.Delete(key)
			gp.droppedCount.Add(1)
		}
	}
}

func (gp *GoRoutinePeer) getActiveGoRoutinesCopy() map[int64]bool {
	gp.lock.RLock()
	defer gp.lock.RUnlock()
	return gp.activeGoRoutines
}

// GetMaxGoId returns the maximum goroutine ID seen with proper locking
func (gp *GoRoutinePeer) GetMaxGoId() int64 {
	gp.lock.RLock()
	defer gp.lock.RUnlock()
	return gp.maxGoId
}

// getTimeSpan returns the overall time range for goroutine collections
func (gp *GoRoutinePeer) getTimeSpan() rpctypes.TimeSpan {
	gp.lock.RLock()
	defer gp.lock.RUnlock()
	return gp.timeSpan
}

// GetGoRoutineCounts returns (total, active, activeOutrig) goroutine counts
func (gp *GoRoutinePeer) GetGoRoutineCounts() (int, int, int) {
	activeGoRoutinesCopy := gp.getActiveGoRoutinesCopy()

	total := int(gp.GetMaxGoId()) // Goroutine IDs start at 1
	active := len(activeGoRoutinesCopy)

	// Count active Outrig goroutines by checking for the "outrig" tag
	activeOutrigCount := 0

	for goId := range activeGoRoutinesCopy {
		goroutine, exists := gp.goRoutines.GetEx(goId)
		if !exists {
			continue
		}

		if slices.Contains(goroutine.Tags, "outrig") {
			activeOutrigCount++
		}
	}

	return total, active, activeOutrigCount
}

// GetParsedGoRoutinesAtTimestamp returns parsed goroutines for RPC at a specific timestamp
// If timestamp is 0, returns the latest goroutines (same as GetParsedGoRoutines)
// If timestamp is provided, returns all goroutines that were active at that timestamp by finding
// the stack trace with the largest timestamp <= the provided timestamp
// If activeOnly is false, returns all goroutines regardless of active status
func (gp *GoRoutinePeer) GetParsedGoRoutinesAtTimestamp(moduleName string, timestamp int64, activeOnly bool) GoRoutinesAtTimestampResult {
	gp.lock.RLock()
	defer gp.lock.RUnlock()

	effectiveTimestamp := timestamp
	if effectiveTimestamp == 0 {
		effectiveTimestamp = gp.timeSpan.End
	}

	// Always get all goroutine IDs for total count
	allGoroutineIds := gp.goRoutines.Keys()
	totalCount := len(allGoroutineIds)

	// Calculate total non-outrig goroutines count
	totalNonOutrig := 0
	for _, goId := range allGoroutineIds {
		goroutineObj, exists := gp.goRoutines.GetEx(goId)
		if !exists {
			continue
		}
		isOutrig := slices.Contains(goroutineObj.Tags, "outrig")
		if !isOutrig {
			totalNonOutrig++
		}
	}

	var goroutineIds []int64
	if activeOnly && timestamp == 0 {
		// If activeOnly and timestamp is 0, use currently active goroutines
		activeGoRoutinesCopy := gp.getActiveGoRoutinesCopy()
		goroutineIds = utilfn.GetKeys(activeGoRoutinesCopy)
	} else {
		// For all other cases: use all goroutines (either activeOnly with timestamp, or not activeOnly)
		goroutineIds = allGoroutineIds
	}
	parsedGoRoutines := gp.getParsedGoRoutinesAtTimestamp_nolock(moduleName, goroutineIds, timestamp, activeOnly)

	return GoRoutinesAtTimestampResult{
		GoRoutines:         parsedGoRoutines,
		TotalCount:         totalCount,
		TotalNonOutrig:     totalNonOutrig,
		EffectiveTimestamp: effectiveTimestamp,
	}
}

// createParsedGoRoutine creates a ParsedGoRoutine from a GoRoutine and stack trace
func (gp *GoRoutinePeer) createParsedGoRoutine(goroutineObj GoRoutine, stack *ds.GoRoutineStack, moduleName string, isActive bool) (rpctypes.ParsedGoRoutine, error) {
	var parsedGoRoutine rpctypes.ParsedGoRoutine
	var err error

	if stack == nil {
		// Create a basic ParsedGoRoutine with just the GoId set
		parsedGoRoutine = rpctypes.ParsedGoRoutine{
			GoId:         goroutineObj.GoId,
			PrimaryState: "inactive",
		}
	} else {
		parsedGoRoutine, err = stacktrace.ParseGoRoutineStackTrace(stack.StackTrace, moduleName, stack.GoId, stack.State)
		if err != nil {
			return rpctypes.ParsedGoRoutine{}, err
		}
	}

	parsedGoRoutine.Name = goroutineObj.Name
	parsedGoRoutine.Tags = goroutineObj.Tags
	parsedGoRoutine.Active = isActive

	// Set CSNum from declaration if available
	if goroutineObj.Decl != nil {
		parsedGoRoutine.CSNum = goroutineObj.Decl.CSNum
	}

	// Set CreatedBy information from stored values
	parsedGoRoutine.CreatedByGoId = goroutineObj.CreatedByGoId
	parsedGoRoutine.CreatedByFrame = goroutineObj.CreatedByFrame

	// Set the active time span
	parsedGoRoutine.ActiveTimeSpan = goroutineObj.TimeSpan

	return parsedGoRoutine, nil
}

// getParsedGoRoutinesAtTimestamp_nolock is the internal implementation that assumes the lock is already held
func (gp *GoRoutinePeer) getParsedGoRoutinesAtTimestamp_nolock(moduleName string, goroutineIds []int64, timestamp int64, activeOnly bool) []rpctypes.ParsedGoRoutine {
	effectiveTimestamp := timestamp
	if effectiveTimestamp == 0 {
		effectiveTimestamp = gp.timeSpan.End
	}

	parsedGoRoutines := make([]rpctypes.ParsedGoRoutine, 0, len(goroutineIds))
	for _, goId := range goroutineIds {
		goroutineObj, exists := gp.goRoutines.GetEx(goId)
		if !exists {
			continue
		}

		// Convert effective timestamp to logical index and get stack directly from CirBuf
		logicalIndex := gp.timeAligner.GetLogicalTimeFromRealTimestamp(effectiveTimestamp)
		bestStack, found := goroutineObj.StackTraces.GetAt(logicalIndex)

		// Determine if goroutine is active at this timestamp
		isActive := goroutineObj.TimeSpan.IsWithinSpanTs(effectiveTimestamp)
		if activeOnly && !isActive {
			continue
		}

		// Check if stack is valid (not a zero value) and pass pointer or nil
		var stackPtr *ds.GoRoutineStack
		if found && bestStack.GoId != 0 {
			stackPtr = &bestStack
		}

		parsedGoRoutine, err := gp.createParsedGoRoutine(goroutineObj, stackPtr, moduleName, isActive)
		if err != nil {
			continue
		}
		parsedGoRoutines = append(parsedGoRoutines, parsedGoRoutine)
	}

	// Sort goroutines by ID to ensure consistent ordering
	if len(parsedGoRoutines) > 1 {
		sort.Slice(parsedGoRoutines, func(i, j int) bool {
			return parsedGoRoutines[i].GoId < parsedGoRoutines[j].GoId
		})
	}

	return parsedGoRoutines
}

// GetParsedGoRoutinesByIds returns parsed goroutines for specific goroutine IDs
func (gp *GoRoutinePeer) GetParsedGoRoutinesByIds(moduleName string, goIds []int64, timestamp int64) []rpctypes.ParsedGoRoutine {
	gp.lock.RLock()
	defer gp.lock.RUnlock()

	return gp.getParsedGoRoutinesAtTimestamp_nolock(moduleName, goIds, timestamp, false)
}

// getActiveCountAtTimeIdx returns the number of goroutines active at the given logical time index
func (gp *GoRoutinePeer) getActiveCountAtTimeIdx(timeIdx int) int {
	activeCount := 0
	gp.timeSpanMap.ForEach(func(goId uint64, timeSpan rpctypes.TimeSpan, version int64) {
		if timeSpan.IsWithinSpanIdx(timeIdx) {
			activeCount++
		}
	})
	return activeCount
}

// GetTimeSpansSinceTickIdx returns the complete GoRoutineTimeSpansResponse for the given tick index
func (gp *GoRoutinePeer) GetTimeSpansSinceTickIdx(sinceTickIdx int64) rpctypes.GoRoutineTimeSpansResponse {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	updatedTimeSpans, _ := gp.timeSpanMap.GetSinceVersion(sinceTickIdx)
	fullTimeSpan := gp.timeSpan
	result := make([]rpctypes.GoTimeSpan, 0, len(updatedTimeSpans))
	for goId, timeSpan := range updatedTimeSpans {
		result = append(result, rpctypes.GoTimeSpan{
			GoId: goId,
			Span: timeSpan,
		})
	}

	// Get all timestamps from the time aligner
	baseLogical, timestamps := gp.timeAligner.GetTimestamps()

	// Filter timestamps to only include those after sinceTickIdx
	startIdx := utilfn.BoundValue(int(sinceTickIdx)-baseLogical+1, 0, len(timestamps))
	filteredTimestamps := timestamps[startIdx:]
	baseLogical += startIdx

	// Create ActiveCounts for each filtered timestamp
	activeCounts := make([]rpctypes.GoRoutineActiveCount, 0, len(filteredTimestamps))
	for i, ts := range filteredTimestamps {
		logicalIdx := baseLogical + i
		activeCount := gp.getActiveCountAtTimeIdx(logicalIdx)
		activeCounts = append(activeCounts, rpctypes.GoRoutineActiveCount{
			TimeIdx: logicalIdx,
			Ts:      ts,
			Count:   activeCount,
		})
	}

	// Get the last tick from the time aligner
	maxLogicalTime := gp.timeAligner.GetMaxLogicalTime()
	lastTickTs := gp.timeAligner.GetRealTimestampFromLogical(maxLogicalTime)
	lastTick := rpctypes.Tick{
		Idx: maxLogicalTime,
		Ts:  lastTickTs,
	}

	return rpctypes.GoRoutineTimeSpansResponse{
		Data:         result,
		FullTimeSpan: fullTimeSpan,
		LastTick:     lastTick,
		ActiveCounts: activeCounts,
		DroppedCount: gp.droppedCount.Load(),
	}
}

// parseRealCreatedBy extracts created by information from decl.RealCreatedBy and sets it in the goroutine
func (gp *GoRoutinePeer) parseRealCreatedBy(decl *ds.GoDecl, goroutine *GoRoutine) {
	// Split RealCreatedBy into function line and file line
	lines := strings.Split(decl.RealCreatedBy, "\n")
	if len(lines) >= 2 {
		funcLine := strings.TrimSpace(lines[0])
		fileLine := strings.TrimSpace(lines[1])
		frame, createdByGoId, ok := stacktrace.ParseCreatedByFrame(funcLine, fileLine)
		if ok {
			stacktrace.AnnotateFrame(frame, "")
			goroutine.CreatedByGoId = int64(createdByGoId)
			goroutine.CreatedByFrame = frame
		}
	}
}
