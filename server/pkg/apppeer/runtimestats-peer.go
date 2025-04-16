// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

const RuntimeStatsBufferSize = 600 // 10 minutes of 1-second samples

// RuntimeStatsPeer manages runtime stats for an AppRunPeer
type RuntimeStatsPeer struct {
	runtimeStats *utilds.CirBuf[ds.RuntimeStatsInfo]
	lock         sync.RWMutex
}

// MakeRuntimeStatsPeer creates a new RuntimeStatsPeer instance
func MakeRuntimeStatsPeer() *RuntimeStatsPeer {
	return &RuntimeStatsPeer{
		runtimeStats: utilds.MakeCirBuf[ds.RuntimeStatsInfo](RuntimeStatsBufferSize),
	}
}

// ProcessRuntimeStats processes runtime stats from a packet
func (rsp *RuntimeStatsPeer) ProcessRuntimeStats(stats ds.RuntimeStatsInfo) {
	rsp.lock.Lock()
	defer rsp.lock.Unlock()

	rsp.runtimeStats.Write(stats)
}

// IsEmpty returns true if the runtime stats buffer is empty
func (rsp *RuntimeStatsPeer) IsEmpty() bool {
	rsp.lock.RLock()
	defer rsp.lock.RUnlock()
	return rsp.runtimeStats.IsEmpty()
}

// GetFilteredStats returns runtime stats filtered by a timestamp
func (rsp *RuntimeStatsPeer) GetFilteredStats(sinceTs int64) []ds.RuntimeStatsInfo {
	rsp.lock.RLock()
	defer rsp.lock.RUnlock()

	return rsp.runtimeStats.FilterItems(func(stat ds.RuntimeStatsInfo, _ int) bool {
		return stat.Ts > sinceTs
	})
}

// ConvertToRuntimeStatData converts a ds.RuntimeStatsInfo to rpctypes.RuntimeStatData
func ConvertToRuntimeStatData(stat ds.RuntimeStatsInfo) rpctypes.RuntimeStatData {
	return rpctypes.RuntimeStatData{
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
	}
}

// GetRuntimeStats retrieves runtime stats for RPC
func (rsp *RuntimeStatsPeer) GetRuntimeStats(sinceTs int64) []rpctypes.RuntimeStatData {
	filteredStats := rsp.GetFilteredStats(sinceTs)
	result := make([]rpctypes.RuntimeStatData, 0, len(filteredStats))
	for _, stat := range filteredStats {
		result = append(result, ConvertToRuntimeStatData(stat))
	}

	return result
}

// GetTotalCollectionCount returns the total number of runtime stats collected
func (rsp *RuntimeStatsPeer) GetTotalCollectionCount() int {
	rsp.lock.RLock()
	defer rsp.lock.RUnlock()
	totalCount, _ := rsp.runtimeStats.GetTotalCountAndHeadOffset()
	return totalCount
}

