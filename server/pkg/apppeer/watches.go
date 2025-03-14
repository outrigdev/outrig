// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"context"
	"fmt"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
)

// GetAppRunWatches retrieves all watches for a specific app run
func GetAppRunWatches(ctx context.Context, req rpctypes.AppRunRequest) (rpctypes.AppRunWatchesData, error) {
	// Get the app run peer
	peer := GetAppRunPeer(req.AppRunId)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunWatchesData{}, fmt.Errorf("app run not found: %s", req.AppRunId)
	}

	// Get all watch names
	watchNames := peer.Watches.Keys()

	// Create a slice to hold all watch data
	watches := make([]ds.Watch, 0, len(watchNames))

	// For each watch, get the most recent value
	for _, name := range watchNames {
		watch, exists := peer.Watches.GetEx(name)
		if !exists {
			continue
		}

		// Get the most recent watch value using GetLast
		latestWatch, _, exists := watch.WatchVals.GetLast()
		if !exists {
			continue
		}

		watches = append(watches, latestWatch)
	}

	// Create and return AppRunWatchesData
	return rpctypes.AppRunWatchesData{
		AppRunId: peer.AppRunId,
		AppName:  peer.AppInfo.AppName,
		Watches:  watches,
	}, nil
}
