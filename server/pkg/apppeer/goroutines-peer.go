// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"slices"
	"sort"
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
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
	goRoutines       *utilds.SyncMap[int64, GoRoutine]
	activeGoRoutines map[int64]bool // Tracks currently running goroutines
	lock             sync.RWMutex   // Lock for synchronizing goroutine operations
	currentIteration int64          // Current iteration counter
	maxGoId          int64          // Maximum goroutine ID seen
}

// MakeGoRoutinePeer creates a new GoRoutinePeer instance
func MakeGoRoutinePeer() *GoRoutinePeer {
	return &GoRoutinePeer{
		goRoutines:       utilds.MakeSyncMap[int64, GoRoutine](),
		activeGoRoutines: make(map[int64]bool),
		currentIteration: 0,
		maxGoId:          0,
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

	// Process goroutine stacks
	for _, stack := range info.Stacks {
		goId := stack.GoId

		activeGoroutines[goId] = true

		// Update maxGoId if we see a larger goroutine ID
		if goId > gp.maxGoId {
			gp.maxGoId = goId
		}

		goroutine, _ := gp.goRoutines.GetOrCreate(goId, func() GoRoutine {
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

		if stack.Name != "" {
			goroutine.Name = stack.Name
		}
		if len(stack.Tags) > 0 {
			goroutine.Tags = stack.Tags
		}

		goroutine.StackTraces.Write(stack)

		gp.goRoutines.Set(goId, goroutine)
	}

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

// getMaxGoId returns the maximum goroutine ID seen with proper locking
func (gp *GoRoutinePeer) getMaxGoId() int64 {
	gp.lock.RLock()
	defer gp.lock.RUnlock()
	return gp.maxGoId
}

// GetGoRoutineCounts returns (total, active, activeOutrig) goroutine counts
func (gp *GoRoutinePeer) GetGoRoutineCounts() (int, int, int) {
	activeGoRoutinesCopy := gp.getActiveGoRoutinesCopy()

	total := int(gp.getMaxGoId()) // Goroutine IDs start at 1
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
	activeGoRoutinesCopy := gp.getActiveGoRoutinesCopy()

	parsedGoRoutines := make([]rpctypes.ParsedGoRoutine, 0, len(activeGoRoutinesCopy))
	for goId := range activeGoRoutinesCopy {
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
		parsedGoRoutine.Active = true // All goroutines returned by this method are active
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
