// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
)

const GoRoutineStackBufferSize = 600 // 10 minutes of 1-second samples

// GoRoutine represents a single goroutine with its stack traces
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

	// Create a new map for active goroutines in this packet
	activeGoroutines := make(map[int64]bool)

	// Process goroutine stacks
	for _, stack := range stacks {
		goId := stack.GoId
		goIdStr := strconv.FormatInt(goId, 10)

		// Mark this goroutine as active
		activeGoroutines[goId] = true

		// Get or create goroutine entry in the syncmap atomically
		goroutine, _ := gp.goRoutines.GetOrCreate(goIdStr, func() GoRoutine {
			// New goroutine
			return GoRoutine{
				GoId:        goId,
				StackTraces: utilds.MakeCirBuf[ds.GoRoutineStack](GoRoutineStackBufferSize),
			}
		})

		// Update name if provided
		if stack.Name != "" {
			goroutine.Name = stack.Name
		}
		if len(stack.Tags) > 0 {
			goroutine.Tags = stack.Tags
		}

		// Add stack trace to the circular buffer
		goroutine.StackTraces.Write(stack)

		// Update the goroutine in the syncmap
		gp.goRoutines.Set(goIdStr, goroutine)
	}

	// Update the active goroutines map
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
	return len(gp.goRoutines.Keys())
}

// GetParsedGoRoutines returns parsed goroutines for RPC
func (gp *GoRoutinePeer) GetParsedGoRoutines(moduleName string) []rpctypes.ParsedGoRoutine {
	gp.lock.RLock()
	defer gp.lock.RUnlock()

	// Prepare a slice to hold the parsed goroutines
	parsedGoRoutines := make([]rpctypes.ParsedGoRoutine, 0, len(gp.activeGoRoutines))

	// Iterate through active goroutines only
	for goId := range gp.activeGoRoutines {
		// Convert goroutine ID to string key
		goIdStr := strconv.FormatInt(goId, 10)

		// Get the goroutine object
		goroutineObj, exists := gp.goRoutines.GetEx(goIdStr)
		if !exists {
			continue
		}

		// Get the most recent stack trace using GetLast
		latestStack, _, exists := goroutineObj.StackTraces.GetLast()
		if !exists {
			continue
		}

		// Parse the stack trace
		parsedGoRoutine, err := goroutine.ParseGoRoutineStackTrace(latestStack.StackTrace, moduleName)
		if err != nil {
			// If parsing fails, skip this goroutine
			continue
		}
		parsedGoRoutine.Name = goroutineObj.Name
		parsedGoRoutine.Tags = goroutineObj.Tags
		parsedGoRoutines = append(parsedGoRoutines, parsedGoRoutine)
	}

	return parsedGoRoutines
}

// GetAppRunGoRoutinesCommand retrieves active goroutines for a specific app run
func GetAppRunGoRoutinesCommand(ctx context.Context, req rpctypes.AppRunRequest) (rpctypes.AppRunGoRoutinesData, error) {
	// Get the app run peer
	peer := GetAppRunPeer(req.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunGoRoutinesData{}, fmt.Errorf("app run not found: %s", req.AppRunId)
	}

	// Get module name from AppInfo
	moduleName := ""
	if peer.AppInfo != nil {
		moduleName = peer.AppInfo.ModuleName
	}

	// Get parsed goroutines from the GoRoutinePeer
	parsedGoRoutines := peer.GoRoutines.GetParsedGoRoutines(moduleName)

	return rpctypes.AppRunGoRoutinesData{
		AppRunId:   peer.AppRunId,
		AppName:    peer.AppInfo.AppName,
		GoRoutines: parsedGoRoutines,
	}, nil
}
