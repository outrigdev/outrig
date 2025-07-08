// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
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
	StackTraces         *utilds.CirBuf[ds.GoRoutineStack]
	FirstSeen           int64 // Timestamp when the goroutine was first seen
	LastSeen            int64 // Timestamp when the goroutine was last seen
	LastActiveIteration int64 // Iteration when the goroutine was last active
}

// GoRoutinePeer manages goroutines for an AppRunPeer
type GoRoutinePeer struct {
	goRoutines        *utilds.SyncMap[int64, GoRoutine]
	activeGoRoutines  map[int64]bool // Tracks currently running goroutines
	lock              sync.RWMutex   // Lock for synchronizing goroutine operations
	currentIteration  int64          // Current iteration counter
	maxGoId           int64          // Maximum goroutine ID seen
	hasSeenFullUpdate bool           // Flag to track if we've seen a full update
	appRunId          string         // ID of the app run this peer belongs to
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
		currentIteration: 0,
		maxGoId:          0,
		appRunId:         appRunId,
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

	// If this is a delta update but we haven't seen a full update yet, ignore it
	if isDelta && !gp.hasSeenFullUpdate {
		fmt.Printf("WARNING: [AppRun: %s] Ignoring delta update because no full update has been seen yet\n", gp.appRunId)
		return
	}

	// If this is a full update, mark that we've seen one
	if !isDelta {
		gp.hasSeenFullUpdate = true
	}

	// Process goroutine stacks
	for _, stack := range info.Stacks {
		goId := stack.GoId

		activeGoroutines[goId] = true

		// Update maxGoId if we see a larger goroutine ID
		if goId > gp.maxGoId {
			gp.maxGoId = goId
		}

		goroutine, wasFound := gp.goRoutines.GetOrCreate(goId, func() GoRoutine {
			return GoRoutine{
				GoId:                goId,
				StackTraces:         utilds.MakeCirBuf[ds.GoRoutineStack](GoRoutineStackBufferSize),
				FirstSeen:           timestamp, // Set FirstSeen to the timestamp from GoroutineInfo
				LastSeen:            timestamp, // Set LastSeen to the timestamp from GoroutineInfo
				LastActiveIteration: gp.currentIteration,
			}
		})

		// Update the last active iteration and last seen timestamp
		goroutine.LastActiveIteration = gp.currentIteration
		goroutine.LastSeen = timestamp

		// Update fields based on the stack data
		if stack.Name != "" {
			goroutine.Name = stack.Name
		}
		if len(stack.Tags) > 0 {
			goroutine.Tags = stack.Tags
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
				goroutine.StackTraces.Write(completeStack)
			} else {
				logKey := fmt.Sprintf("goroutine-nodeltaupdate-%s", gp.appRunId)
				logutil.LogfOnce(logKey, "WARNING: [AppRun: %s] Delta update received for goroutine %d with no last stack\n", gp.appRunId, goId)
			}
		} else {
			// full updates write the stack directly
			goroutine.StackTraces.Write(stack)
		}

		gp.goRoutines.Set(goId, goroutine)
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

// GetParsedGoRoutines returns parsed goroutines for RPC
func (gp *GoRoutinePeer) GetParsedGoRoutines(moduleName string) []rpctypes.ParsedGoRoutine {
	return gp.GetParsedGoRoutinesAtTimestamp(moduleName, 0)
}

// GetParsedGoRoutinesAtTimestamp returns parsed goroutines for RPC at a specific timestamp
// If timestamp is 0, returns the latest goroutines (same as GetParsedGoRoutines)
// If timestamp is provided, returns all goroutines that were active at that timestamp by finding
// the stack trace with the largest timestamp <= the provided timestamp
func (gp *GoRoutinePeer) GetParsedGoRoutinesAtTimestamp(moduleName string, timestamp int64) []rpctypes.ParsedGoRoutine {
	var goroutineIds []int64
	var parsedGoRoutines []rpctypes.ParsedGoRoutine

	if timestamp == 0 {
		// If timestamp is 0, use currently active goroutines
		activeGoRoutinesCopy := gp.getActiveGoRoutinesCopy()
		goroutineIds = make([]int64, 0, len(activeGoRoutinesCopy))
		for goId := range activeGoRoutinesCopy {
			goroutineIds = append(goroutineIds, goId)
		}
		parsedGoRoutines = make([]rpctypes.ParsedGoRoutine, 0, len(activeGoRoutinesCopy))
	} else {
		// If timestamp is provided, check all goroutines to see which were active at that time
		allKeys := gp.goRoutines.Keys()
		goroutineIds = make([]int64, 0, len(allKeys))
		for _, goId := range allKeys {
			goroutineIds = append(goroutineIds, goId)
		}
		parsedGoRoutines = make([]rpctypes.ParsedGoRoutine, 0, len(allKeys))
	}

	activeGoRoutinesCopy := gp.getActiveGoRoutinesCopy()

	for _, goId := range goroutineIds {
		goroutineObj, exists := gp.goRoutines.GetEx(goId)
		if !exists {
			continue
		}

		var bestStack ds.GoRoutineStack
		var found bool

		if timestamp == 0 {
			// Use latest stack if timestamp is 0
			bestStack, _, found = goroutineObj.StackTraces.GetLast()
		} else {
			// Find the stack trace with the largest timestamp <= the provided timestamp
			// Since ForEach iterates from oldest to newest, we can break early once we find a timestamp > target
			goroutineObj.StackTraces.ForEach(func(stack ds.GoRoutineStack) bool {
				if stack.Ts > timestamp {
					// Since timestamps are in order, we can stop here
					return false
				}
				// Keep updating bestStack since they're in chronological order
				bestStack = stack
				found = true
				return true // Continue iteration
			})
		}

		if !found {
			continue
		}

		parsedGoRoutine, err := stacktrace.ParseGoRoutineStackTrace(bestStack.StackTrace, moduleName, bestStack.GoId, bestStack.State)
		if err != nil {
			continue
		}
		parsedGoRoutine.Name = goroutineObj.Name
		parsedGoRoutine.Tags = goroutineObj.Tags
		parsedGoRoutine.FirstSeen = goroutineObj.FirstSeen
		parsedGoRoutine.LastSeen = goroutineObj.LastSeen
		// Set Active flag based on whether it's currently active (timestamp=0) or was active at the given timestamp
		if timestamp == 0 {
			parsedGoRoutine.Active = activeGoRoutinesCopy[goId]
		} else {
			// For historical timestamps, consider it active if we found a stack trace at that time
			parsedGoRoutine.Active = found
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
func (gp *GoRoutinePeer) GetParsedGoRoutinesByIds(moduleName string, goIds []int64) []rpctypes.ParsedGoRoutine {
	activeGoRoutinesCopy := gp.getActiveGoRoutinesCopy()

	parsedGoRoutines := make([]rpctypes.ParsedGoRoutine, 0, len(goIds))
	for _, goId := range goIds {
		goroutineObj, exists := gp.goRoutines.GetEx(goId)
		if !exists {
			continue
		}

		latestStack, _, exists := goroutineObj.StackTraces.GetLast()
		if !exists {
			continue
		}

		parsedGoRoutine, err := stacktrace.ParseGoRoutineStackTrace(latestStack.StackTrace, moduleName, latestStack.GoId, latestStack.State)
		if err != nil {
			continue
		}
		parsedGoRoutine.Name = goroutineObj.Name
		parsedGoRoutine.Tags = goroutineObj.Tags
		parsedGoRoutine.FirstSeen = goroutineObj.FirstSeen
		parsedGoRoutine.LastSeen = goroutineObj.LastSeen
		parsedGoRoutine.Active = activeGoRoutinesCopy[goId] // Set active flag based on whether it's in the activeGoRoutines map
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
