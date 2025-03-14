// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"context"
	"fmt"

	"github.com/outrigdev/outrig/pkg/rpctypes"
)

// GetAppRunRuntimeStats retrieves runtime stats for a specific app run
func GetAppRunRuntimeStats(ctx context.Context, req rpctypes.AppRunRequest) (rpctypes.AppRunRuntimeStatsData, error) {
	// Get the app run peer
	peer := GetAppRunPeer(req.AppRunId)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunRuntimeStatsData{}, fmt.Errorf("app run not found: %s", req.AppRunId)
	}

	// Get the most recent runtime stats using GetLast
	latestStats, _, exists := peer.RuntimeStats.GetLast()
	if !exists {
		return rpctypes.AppRunRuntimeStatsData{
			AppRunId: peer.AppRunId,
			AppName:  peer.AppInfo.AppName,
			Ts:       0,
		}, nil
	}

	// Create and return AppRunRuntimeStatsData
	return rpctypes.AppRunRuntimeStatsData{
		AppRunId:       peer.AppRunId,
		AppName:        peer.AppInfo.AppName,
		Ts:             latestStats.Ts,
		CPUUsage:       latestStats.CPUUsage,
		GoRoutineCount: latestStats.GoRoutineCount,
		GoMaxProcs:     latestStats.GoMaxProcs,
		NumCPU:         latestStats.NumCPU,
		GOOS:           latestStats.GOOS,
		GOARCH:         latestStats.GOARCH,
		GoVersion:      latestStats.GoVersion,
		Pid:            latestStats.Pid,
		Cwd:            latestStats.Cwd,
		MemStats:       latestStats.MemStats,
	}, nil
}
