// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"strconv"
	"sync"

	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
)

const GoRoutineStackBufferSize = 600

// GoRoutine represents a goroutine with its stack traces
type GoRoutine struct {
	GoId        int64
	Name        string
	Tags        []string
	StackTraces *utilds.CirBuf[ds.GoRoutineStack]
}

// GoRoutinePeer manages goroutines for an AppRunPeer
type GoRoutinePeer struct {
	goRoutines       *utilds.SyncMap[GoRoutine]
	activeGoRoutines map[int64]bool // Tracks currently running goroutines
	lock             sync.RWMutex   // Lock for synchronizing goroutine operations
}

// MakeGoRoutinePeer creates a new GoRoutinePeer instance
func MakeGoRoutinePeer() *GoRoutinePeer {
	return &GoRoutinePeer{
		goRoutines:       utilds.MakeSyncMap[GoRoutine](),
		activeGoRoutines: make(map[int64]bool),
	}
}

// ProcessGoroutineStacks processes goroutine stacks from a packet
func (gp *GoRoutinePeer) ProcessGoroutineStacks(stacks []ds.GoRoutineStack) {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	activeGoroutines := make(map[int64]bool)

	// Process goroutine stacks
	for _, stack := range stacks {
		goId := stack.GoId
		goIdStr := strconv.FormatInt(goId, 10)

		activeGoroutines[goId] = true

		goroutine, _ := gp.goRoutines.GetOrCreate(goIdStr, func() GoRoutine {
			return GoRoutine{
				GoId:        goId,
				StackTraces: utilds.MakeCirBuf[ds.GoRoutineStack](GoRoutineStackBufferSize),
			}
		})

		if stack.Name != "" {
			goroutine.Name = stack.Name
		}
		if len(stack.Tags) > 0 {
			goroutine.Tags = stack.Tags
		}

		goroutine.StackTraces.Write(stack)

		gp.goRoutines.Set(goIdStr, goroutine)
	}

	gp.activeGoRoutines = activeGoroutines
}

// GetActiveGoRoutineCount returns the number of active goroutines
func (gp *GoRoutinePeer) GetActiveGoRoutineCount() int {
	gp.lock.RLock()
	defer gp.lock.RUnlock()
	return len(gp.activeGoRoutines)
}

// GetTotalGoRoutineCount returns the total number of goroutines (active and inactive)
func (gp *GoRoutinePeer) GetTotalGoRoutineCount() int {
	return gp.goRoutines.Len()
}

// GetParsedGoRoutines returns parsed goroutines for RPC
func (gp *GoRoutinePeer) GetParsedGoRoutines(moduleName string) []rpctypes.ParsedGoRoutine {
	// Get a local copy of the activeGoRoutines map under lock
	gp.lock.RLock()
	activeGoRoutinesCopy := gp.activeGoRoutines
	gp.lock.RUnlock()

	parsedGoRoutines := make([]rpctypes.ParsedGoRoutine, 0, len(activeGoRoutinesCopy))
	for goId := range activeGoRoutinesCopy {
		goIdStr := strconv.FormatInt(goId, 10)

		goroutineObj, exists := gp.goRoutines.GetEx(goIdStr)
		if !exists {
			continue
		}

		latestStack, _, exists := goroutineObj.StackTraces.GetLast()
		if !exists {
			continue
		}

		parsedGoRoutine, err := goroutine.ParseGoRoutineStackTrace(latestStack.StackTrace, moduleName)
		if err != nil {
			continue
		}
		parsedGoRoutine.Name = goroutineObj.Name
		parsedGoRoutine.Tags = goroutineObj.Tags
		parsedGoRoutines = append(parsedGoRoutines, parsedGoRoutine)
	}

	return parsedGoRoutines
}

// GetParsedGoRoutinesByIds returns parsed goroutines for specific goroutine IDs
func (gp *GoRoutinePeer) GetParsedGoRoutinesByIds(moduleName string, goIds []int64) []rpctypes.ParsedGoRoutine {
	// No lock needed as we're accessing thread-safe structures
	parsedGoRoutines := make([]rpctypes.ParsedGoRoutine, 0, len(goIds))
	for _, goId := range goIds {
		goIdStr := strconv.FormatInt(goId, 10)

		goroutineObj, exists := gp.goRoutines.GetEx(goIdStr)
		if !exists {
			continue
		}

		latestStack, _, exists := goroutineObj.StackTraces.GetLast()
		if !exists {
			continue
		}

		parsedGoRoutine, err := goroutine.ParseGoRoutineStackTrace(latestStack.StackTrace, moduleName)
		if err != nil {
			continue
		}
		parsedGoRoutine.Name = goroutineObj.Name
		parsedGoRoutine.Tags = goroutineObj.Tags
		parsedGoRoutines = append(parsedGoRoutines, parsedGoRoutine)
	}

	return parsedGoRoutines
}
