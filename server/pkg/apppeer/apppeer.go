// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/tevent"
)

const (
	MaxAppRunPeers = 8
	PruneInterval  = 15 * time.Second
)

// For tracking app run stats deltas
var (
	lastAppRunStatsMap  = make(map[string]tevent.AppRunStats)
	lastAppRunStatsLock sync.Mutex
)

const (
	AppStatusRunning      = "running"
	AppStatusDone         = "done"
	AppStatusDisconnected = "disconnected"
)

// AppRunPeer represents a peer connection to an app client
type AppRunPeer struct {
	AppRunId    string
	AppInfo     *ds.AppInfo
	Status      string     // Current status of the application
	LastModTime int64      // Last modification time in milliseconds
	refCount    int        // Reference counter
	refLock     sync.Mutex // Lock for reference counter operations
	dataLock    sync.Mutex // Lock for data fields (CollectorStatus, etc.)

	Logs            *LogLinePeer
	GoRoutines      *GoRoutinePeer
	Watches         *WatchesPeer
	RuntimeStats    *RuntimeStatsPeer
	CollectorStatus map[string]ds.CollectorStatus // Collector statuses by name

	lastSentStats *tevent.AppRunStats // Last stats sent in disconnected event
}

// Global synchronized map to hold all AppRunPeers
var appRunPeers = utilds.MakeSyncMap[string, *AppRunPeer]()

func init() {
	outrig.NewWatch("apppeer.keys").PollFunc(func() []string {
		keys := appRunPeers.Keys()
		sort.Strings(keys)
		return keys
	})

	// Start a goroutine to periodically prune app run peers
	go func() {
		outrig.SetGoRoutineName("apppeer.prune")
		for {
			time.Sleep(PruneInterval)
			numPruned := PruneAppRunPeers()
			if numPruned > 0 {
				log.Printf("Periodic pruning removed %d app run peers", numPruned)
			}
		}
	}()
}

// These functions have been moved to the logwriter package

// GetAppRunPeer gets an existing AppRunPeer by ID or creates a new one if it doesn't exist
// If incRefCount is true, increments the reference counter
func GetAppRunPeer(appRunId string, incRefCount bool) *AppRunPeer {
	peer, _ := appRunPeers.GetOrCreate(appRunId, func() *AppRunPeer {
		return &AppRunPeer{
			AppRunId:      appRunId,
			Logs:          MakeLogLinePeer(),
			GoRoutines:    MakeGoRoutinePeer(appRunId),
			Watches:       MakeWatchesPeer(appRunId),
			RuntimeStats:  MakeRuntimeStatsPeer(),
			Status:        AppStatusRunning,
			LastModTime:   time.Now().UnixMilli(),
			refCount:      0,
			lastSentStats: nil,
		}
	})

	// Increment reference counter if requested
	if incRefCount {
		peer.refLock.Lock()
		defer peer.refLock.Unlock()
		peer.refCount++
	}

	return peer
}

// Release decrements the reference counter and closes resources when it reaches zero
func (p *AppRunPeer) Release() {
	p.refLock.Lock()
	defer p.refLock.Unlock()

	p.refCount--

	if p.refCount > 0 {
		return
	}
	// Only close resources when reference count reaches zero
	if p.Status != AppStatusDone {
		p.Status = AppStatusDisconnected
		p.LastModTime = time.Now().UnixMilli()
		log.Printf("Connection closed for app run ID: %s, marked as disconnected", p.AppRunId)
	}

	// Send disconnected event
	p.sendDisconnectedEvent()
}

// GetRefCount safely returns the current reference count
func (p *AppRunPeer) GetRefCount() int {
	p.refLock.Lock()
	defer p.refLock.Unlock()
	return p.refCount
}

// GetAllAppRunPeers returns all AppRunPeers
func GetAllAppRunPeers() []*AppRunPeer {
	keys := appRunPeers.Keys()
	peers := make([]*AppRunPeer, 0, len(keys))
	for _, key := range keys {
		if peer, exists := appRunPeers.GetEx(key); exists {
			peers = append(peers, peer)
		}
	}

	return peers
}

// GetAllAppRunPeerInfos returns AppRunInfo for all valid app run peers
// If since > 0, only returns peers that have been modified since the given timestamp
func GetAllAppRunPeerInfos(since int64) []rpctypes.AppRunInfo {
	appRunPeers := GetAllAppRunPeers()
	appRuns := make([]rpctypes.AppRunInfo, 0, len(appRunPeers))
	for _, peer := range appRunPeers {
		// Skip peers with no AppInfo
		if peer.AppInfo == nil {
			continue
		}

		// Skip peers that haven't been modified since the given timestamp
		if peer.LastModTime <= since {
			continue
		}

		// Get AppRunInfo from the peer
		appRun := peer.GetAppRunInfo()
		appRuns = append(appRuns, appRun)
	}

	return appRuns
}

// HandlePacket processes a packet received from the domain socket connection
func (p *AppRunPeer) HandlePacket(packetType string, packetData json.RawMessage) error {
	p.LastModTime = time.Now().UnixMilli()

	switch packetType {
	case ds.PacketTypeAppInfo:
		var appInfo ds.AppInfo
		if err := json.Unmarshal(packetData, &appInfo); err != nil {
			return fmt.Errorf("failed to unmarshal AppInfo: %w", err)
		}
		p.AppInfo = &appInfo
		p.Status = AppStatusRunning
		log.Printf("Received AppInfo for app run ID: %s, app: %s", p.AppRunId, appInfo.AppName)

		// Extract Go version if available
		goVersion := ""
		if appInfo.BuildInfo != nil {
			goVersion = appInfo.BuildInfo.GoVersion
		}
		tevent.SendAppRunConnectedEvent(appInfo.OutrigSDKVersion, goVersion)

	case ds.PacketTypeLog:
		var logLine ds.LogLine
		if err := json.Unmarshal(packetData, &logLine); err != nil {
			return fmt.Errorf("failed to unmarshal LogLine: %w", err)
		}
		p.Logs.ProcessLogLine(logLine)

	case ds.PacketTypeMultiLog:
		var multiLogLines ds.MultiLogLines
		if err := json.Unmarshal(packetData, &multiLogLines); err != nil {
			return fmt.Errorf("failed to unmarshal MultiLogLines: %w", err)
		}
		p.Logs.ProcessMultiLogLines(multiLogLines.LogLines)

	case ds.PacketTypeGoroutine:
		var goroutineInfo ds.GoroutineInfo
		if err := json.Unmarshal(packetData, &goroutineInfo); err != nil {
			return fmt.Errorf("failed to unmarshal GoroutineInfo: %w", err)
		}
		p.GoRoutines.ProcessGoroutineStacks(goroutineInfo)
		log.Printf("Processed %d goroutines for app run ID: %s (delta: %v)", len(goroutineInfo.Stacks), p.AppRunId, goroutineInfo.Delta)

	case ds.PacketTypeWatch:
		var watchInfo ds.WatchInfo
		if err := json.Unmarshal(packetData, &watchInfo); err != nil {
			return fmt.Errorf("failed to unmarshal WatchInfo: %w", err)
		}
		p.Watches.ProcessWatchInfo(watchInfo)
		log.Printf("Processed %d watches for app run ID: %s (delta: %v)", len(watchInfo.Watches), p.AppRunId, watchInfo.Delta)

	case ds.PacketTypeAppDone:
		p.Status = AppStatusDone
		log.Printf("Received AppDone for app run ID: %s", p.AppRunId)

	case ds.PacketTypeRuntimeStats:
		var runtimeStats ds.RuntimeStatsInfo
		if err := json.Unmarshal(packetData, &runtimeStats); err != nil {
			return fmt.Errorf("failed to unmarshal RuntimeStatsInfo: %w", err)
		}
		p.RuntimeStats.ProcessRuntimeStats(runtimeStats)
		log.Printf("Received runtime stats for app run ID: %s", p.AppRunId)

	case ds.PacketTypeCollectorStatus:
		var collectorStatuses map[string]ds.CollectorStatus
		if err := json.Unmarshal(packetData, &collectorStatuses); err != nil {
			return fmt.Errorf("failed to unmarshal CollectorStatus: %w", err)
		}
		p.dataLock.Lock()
		p.CollectorStatus = collectorStatuses
		p.dataLock.Unlock()
		log.Printf("Received collector statuses for app run ID: %s (%d collectors)", p.AppRunId, len(collectorStatuses))

	default:
		log.Printf("Unknown packet type: %s", packetType)
	}

	return nil
}

// PruneAppRunPeers removes old app run peers to keep the total count under MaxAppRunPeers
// It will not prune peers that are running or have a non-zero reference count
func PruneAppRunPeers() int {
	allPeers := GetAllAppRunPeers()
	if len(allPeers) <= MaxAppRunPeers {
		return 0
	}

	sort.Slice(allPeers, func(i, j int) bool {
		return allPeers[i].LastModTime < allPeers[j].LastModTime
	})

	numPruned := 0
	for _, peer := range allPeers {
		if len(allPeers)-numPruned <= MaxAppRunPeers {
			break
		}
		if peer.Status == AppStatusRunning {
			continue
		}
		if peer.GetRefCount() > 0 {
			continue
		}
		appRunPeers.Delete(peer.AppRunId)
		log.Printf("Pruned app run peer: %s (last modified: %s)",
			peer.AppRunId, time.UnixMilli(peer.LastModTime).Format(time.RFC3339))
		numPruned++
	}

	return numPruned
}

// GetAppRunInfo constructs and returns an AppRunInfo struct for this peer
func (p *AppRunPeer) GetAppRunInfo() rpctypes.AppRunInfo {
	if p.AppInfo == nil {
		return rpctypes.AppRunInfo{}
	}

	isRunning := p.Status == AppStatusRunning
	numTotalGoRoutines, numActiveGoRoutines, numOutrigGoRoutines := p.GoRoutines.GetGoRoutineCounts()
	numActiveWatches := p.Watches.GetActiveWatchCount()
	numTotalWatches := p.Watches.GetTotalWatchCount()
	numLogs := p.Logs.GetTotalCount()
	appRunInfo := rpctypes.AppRunInfo{
		AppRunId:            p.AppRunId,
		AppName:             p.AppInfo.AppName,
		StartTime:           p.AppInfo.StartTime,
		IsRunning:           isRunning,
		Status:              p.Status,
		NumLogs:             numLogs,
		NumTotalGoRoutines:  numTotalGoRoutines,
		NumActiveGoRoutines: numActiveGoRoutines,
		NumOutrigGoRoutines: numOutrigGoRoutines,
		NumActiveWatches:    numActiveWatches,
		NumTotalWatches:     numTotalWatches,
		LastModTime:         p.LastModTime,
		ModuleName:          p.AppInfo.ModuleName,
		Executable:          p.AppInfo.Executable,
		OutrigSDKVersion:    p.AppInfo.OutrigSDKVersion,
	}

	if p.AppInfo.BuildInfo != nil {
		appRunInfo.BuildInfo = &rpctypes.BuildInfoData{
			GoVersion: p.AppInfo.BuildInfo.GoVersion,
			Path:      p.AppInfo.BuildInfo.Path,
			Version:   p.AppInfo.BuildInfo.Version,
			Settings:  p.AppInfo.BuildInfo.Settings,
		}
	}

	return appRunInfo
}

// GetPeerStats returns the stats and status for this peer
func (p *AppRunPeer) GetPeerStats() (tevent.AppRunStats, string) {
	if p.AppInfo == nil {
		return tevent.AppRunStats{}, ""
	}

	// Calculate connection time in milliseconds
	var connTime int64
	if p.AppInfo.StartTime > 0 {
		connTime = time.Now().UnixMilli() - p.AppInfo.StartTime
	}

	// Collect stats
	numTotalGoRoutines := int(p.GoRoutines.GetMaxGoId())
	numTotalWatches := p.Watches.GetTotalWatchCount()
	numLogs := p.Logs.GetTotalCount()
	numCollections := p.RuntimeStats.GetTotalCollectionCount()

	stats := tevent.AppRunStats{
		LogLines:    numLogs,
		GoRoutines:  numTotalGoRoutines,
		Watches:     numTotalWatches,
		Collections: numCollections,
		SDKVersion:  p.AppInfo.OutrigSDKVersion,
		ConnTimeMs:  connTime,
		AppRunCount: 1,
	}

	return stats, p.Status
}

// sendDisconnectedEvent sends an apprun:disconnected telemetry event with stats
// should be holding the lock
func (p *AppRunPeer) sendDisconnectedEvent() {
	if p.AppInfo == nil {
		return
	}

	currentStats, _ := p.GetPeerStats()

	// If we have previous stats, send only the delta
	if p.lastSentStats != nil {
		// Calculate the difference between current and last stats
		deltaStats := currentStats.Sub(*p.lastSentStats)
		tevent.SendAppRunDisconnectedEvent(deltaStats)
	} else {
		// First time sending stats, send the full stats
		tevent.SendAppRunDisconnectedEvent(currentStats)
	}

	// Store the current stats for next time
	statsCopy := currentStats
	p.lastSentStats = &statsCopy
}

// GetAppRunStatsDelta returns the delta of cumulative stats from all app runs
// since the last time this function was called, and the count of active app runs.
func GetAppRunStatsDelta() (tevent.AppRunStats, int) {
	allPeers := GetAllAppRunPeers()
	activeAppRuns := 0

	lastAppRunStatsLock.Lock()
	defer lastAppRunStatsLock.Unlock()

	var deltaStats tevent.AppRunStats
	newStatsMap := make(map[string]tevent.AppRunStats)

	for _, peer := range allPeers {
		if peer.AppInfo == nil {
			continue
		}

		peerStats, peerStatus := peer.GetPeerStats()
		if peerStatus == AppStatusRunning {
			activeAppRuns++
		}
		newStatsMap[peer.AppRunId] = peerStats

		if lastPeerStats, exists := lastAppRunStatsMap[peer.AppRunId]; exists {
			peerDelta := peerStats.Sub(lastPeerStats)
			deltaStats = deltaStats.Add(peerDelta)
		} else {
			deltaStats = deltaStats.Add(peerStats)
		}
	}

	lastAppRunStatsMap = newStatsMap

	// Clear SDK version as it's not relevant in aggregated form
	deltaStats.SDKVersion = ""

	return deltaStats, activeAppRuns
}
