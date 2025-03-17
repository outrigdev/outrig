// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"context"
	"fmt"

	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/rpctypes"
)

// GetAppRunGoRoutinesCommand retrieves goroutines for a specific app run
func GetAppRunGoRoutinesCommand(ctx context.Context, req rpctypes.AppRunRequest) (rpctypes.AppRunGoRoutinesData, error) {
	// Get the app run peer
	peer := GetAppRunPeer(req.AppRunId)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunGoRoutinesData{}, fmt.Errorf("app run not found: %s", req.AppRunId)
	}

	// Get all goroutine keys
	goroutineKeys := peer.GoRoutines.Keys()
	parsedGoRoutines := make([]rpctypes.ParsedGoRoutine, 0, len(goroutineKeys))

	// For each goroutine, get the most recent stack trace
	for _, key := range goroutineKeys {
		goroutineObj, exists := peer.GoRoutines.GetEx(key)
		if !exists {
			continue
		}

		// Get the most recent stack trace using GetLast
		latestStack, _, exists := goroutineObj.StackTraces.GetLast()
		if !exists {
			continue
		}

		// Parse the stack trace, passing the module name from AppInfo
		moduleName := ""
		if peer.AppInfo != nil {
			moduleName = peer.AppInfo.ModuleName
		}
		parsedGoRoutine, err := goroutine.ParseGoRoutineStackTrace(latestStack.StackTrace, moduleName)
		if err != nil {
			// If parsing fails, skip this goroutine
			continue
		}

		parsedGoRoutines = append(parsedGoRoutines, parsedGoRoutine)
	}

	return rpctypes.AppRunGoRoutinesData{
		AppRunId:   peer.AppRunId,
		AppName:    peer.AppInfo.AppName,
		GoRoutines: parsedGoRoutines,
	}, nil
}
