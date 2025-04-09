// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig/server/pkg/tevent"
)

const (
	MaxAppRunPeers = 8
	PruneInterval  = 15 * time.Second
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

	Logs         *LogLinePeer
	GoRoutines   *GoRoutinePeer
	Watches      *WatchesPeer
	RuntimeStats *RuntimeStatsPeer
}

// Global synchronized map to hold all AppRunPeers
var appRunPeers = utilds.MakeSyncMap[string, *AppRunPeer]()

func init() {
	outrig.WatchFunc("apppeer.keys", func() []string {
		keys := appRunPeers.Keys()
		sort.Strings(keys)
		return keys
	}, nil)

	// Start a goroutine to periodically prune app run peers
	go func() {
		for {
			time.Sleep(PruneInterval)
			numPruned := PruneAppRunPeers()
			if numPruned > 0 {
				log.Printf("Periodic pruning removed %d app run peers", numPruned)
			}
		}
	}()
}

// getAppRunDir returns the directory for storing app run data
func getAppRunDir(appRunId string) string {
	dataDir := utilfn.ExpandHomeDir(serverbase.GetOutrigDataDir())
	return filepath.Join(dataDir, appRunId)
}

// These functions have been moved to the logwriter package

// GetAppRunPeer gets an existing AppRunPeer by ID or creates a new one if it doesn't exist
// If incRefCount is true, increments the reference counter
func GetAppRunPeer(appRunId string, incRefCount bool) *AppRunPeer {
	peer, _ := appRunPeers.GetOrCreate(appRunId, func() *AppRunPeer {
		// Create a directory for this app run
		appRunDir := getAppRunDir(appRunId)
		err := os.MkdirAll(appRunDir, 0755)
		if err != nil {
			log.Printf("Failed to create directory for app run %s: %v", appRunId, err)
		}

		return &AppRunPeer{
			AppRunId:     appRunId,
			Logs:         MakeLogLinePeer(),
			GoRoutines:   MakeGoRoutinePeer(),
			Watches:      MakeWatchesPeer(),
			RuntimeStats: MakeRuntimeStatsPeer(),
			Status:       AppStatusRunning,
			LastModTime:  time.Now().UnixMilli(),
			refCount:     0,
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
		tevent.SendAppRunConnectedEvent()

	case ds.PacketTypeLog:
		var logLine ds.LogLine
		if err := json.Unmarshal(packetData, &logLine); err != nil {
			return fmt.Errorf("failed to unmarshal LogLine: %w", err)
		}
		p.Logs.ProcessLogLine(logLine)

	case ds.PacketTypeGoroutine:
		var goroutineInfo ds.GoroutineInfo
		if err := json.Unmarshal(packetData, &goroutineInfo); err != nil {
			return fmt.Errorf("failed to unmarshal GoroutineInfo: %w", err)
		}

		p.GoRoutines.ProcessGoroutineStacks(goroutineInfo)

		log.Printf("Processed %d goroutines for app run ID: %s", len(goroutineInfo.Stacks), p.AppRunId)

	case ds.PacketTypeWatch:
		var watchInfo ds.WatchInfo
		if err := json.Unmarshal(packetData, &watchInfo); err != nil {
			return fmt.Errorf("failed to unmarshal WatchInfo: %w", err)
		}

		p.Watches.ProcessWatchValues(watchInfo.Watches)

		log.Printf("Processed %d watches for app run ID: %s", len(watchInfo.Watches), p.AppRunId)

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
	numActiveGoRoutines := p.GoRoutines.GetActiveGoRoutineCount()
	numTotalGoRoutines := p.GoRoutines.GetTotalGoRoutineCount()
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
		NumActiveGoRoutines: numActiveGoRoutines,
		NumTotalGoRoutines:  numTotalGoRoutines,
		NumActiveWatches:    numActiveWatches,
		NumTotalWatches:     numTotalWatches,
		LastModTime:         p.LastModTime,
		ModuleName:          p.AppInfo.ModuleName,
		Executable:          p.AppInfo.Executable,
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
