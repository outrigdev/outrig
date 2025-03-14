// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"context"
	"fmt"

	"github.com/outrigdev/outrig/pkg/rpctypes"
)

// GetAppRunGoRoutines retrieves goroutines for a specific app run
func GetAppRunGoRoutines(ctx context.Context, req rpctypes.AppRunRequest) (rpctypes.AppRunGoroutinesData, error) {
	// Get the app run peer
	peer := GetAppRunPeer(req.AppRunId)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunGoroutinesData{}, fmt.Errorf("app run not found: %s", req.AppRunId)
	}

	// Get all goroutine keys
	goroutineKeys := peer.GoRoutines.Keys()
	goroutines := make([]rpctypes.GoroutineData, 0, len(goroutineKeys))

	// For each goroutine, get the most recent stack trace
	for _, key := range goroutineKeys {
		goroutine, exists := peer.GoRoutines.GetEx(key)
		if !exists {
			continue
		}

		// Get the most recent stack trace using GetLast
		latestStack, _, exists := goroutine.StackTraces.GetLast()
		if !exists {
			continue
		}

		goroutines = append(goroutines, rpctypes.GoroutineData{
			GoId:       latestStack.GoId,
			State:      latestStack.State,
			StackTrace: latestStack.StackTrace,
		})
	}

	return rpctypes.AppRunGoroutinesData{
		AppRunId:   peer.AppRunId,
		AppName:    peer.AppInfo.AppName,
		GoRoutines: goroutines,
	}, nil
}
