// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"context"
	"fmt"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
)

// GetAppRunRuntimeStats retrieves runtime stats for a specific app run
// If sinceTs is provided, it returns all stats with timestamps greater than sinceTs
func GetAppRunRuntimeStats(ctx context.Context, req rpctypes.AppRunRequest) (rpctypes.AppRunRuntimeStatsData, error) {
	// Get the app run peer
	peer := GetAppRunPeer(req.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunRuntimeStatsData{}, fmt.Errorf("app run not found: %s", req.AppRunId)
	}

	// Initialize empty result
	result := rpctypes.AppRunRuntimeStatsData{
		AppRunId: peer.AppRunId,
		AppName:  peer.AppInfo.AppName,
		Stats:    []rpctypes.RuntimeStatData{},
	}

	// If the buffer is empty, return empty stats
	if peer.RuntimeStats.IsEmpty() {
		return result, nil
	}

	// Filter stats based on sinceTs using the new FilterItems method
	filteredStats := peer.RuntimeStats.FilterItems(func(stat ds.RuntimeStatsInfo, _ int) bool {
		return stat.Ts > req.Since
	})

	// Convert each ds.RuntimeStatsInfo to rpctypes.RuntimeStatData
	for _, stat := range filteredStats {
		result.Stats = append(result.Stats, rpctypes.RuntimeStatData{
			Ts:             stat.Ts,
			CPUUsage:       stat.CPUUsage,
			GoRoutineCount: stat.GoRoutineCount,
			GoMaxProcs:     stat.GoMaxProcs,
			NumCPU:         stat.NumCPU,
			GOOS:           stat.GOOS,
			GOARCH:         stat.GOARCH,
			GoVersion:      stat.GoVersion,
			Pid:            stat.Pid,
			Cwd:            stat.Cwd,
			MemStats:       stat.MemStats,
		})
	}

	return result, nil
}
