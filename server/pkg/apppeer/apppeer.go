// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
)

const LogLineBufferSize = 10000
const GoRoutineStackBufferSize = 600 // 10 minutes of 1-second samples
const WatchBufferSize = 600          // 10 minutes of 1-second samples
const RuntimeStatsBufferSize = 600   // 10 minutes of 1-second samples

// Application status constants
const (
	AppStatusRunning      = "running"
	AppStatusDone         = "done"
	AppStatusDisconnected = "disconnected"
)

// AppRunPeer represents a peer connection to an app client
type AppRunPeer struct {
	AppRunId         string
	AppInfo          *ds.AppInfo
	Logs             *utilds.CirBuf[ds.LogLine]
	GoRoutines       *utilds.SyncMap[GoRoutine]
	ActiveGoRoutines map[int]bool // Tracks currently running goroutines
	Watches          *utilds.SyncMap[Watch]
	ActiveWatches    map[string]bool                     // Tracks currently active watches
	RuntimeStats     *utilds.CirBuf[ds.RuntimeStatsInfo] // History of runtime stats
	Status           string                              // Current status of the application
	LastModTime      int64                               // Last modification time in milliseconds
}

type GoRoutine struct {
	GoId        int
	StackTraces *utilds.CirBuf[ds.GoRoutineStack]
}

type Watch struct {
	Name      string
	WatchVals *utilds.CirBuf[ds.Watch]
}

// Global synchronized map to hold all AppRunPeers
var appRunPeers = utilds.MakeSyncMap[*AppRunPeer]()

// GetAppRunPeer gets an existing AppRunPeer by ID or creates a new one if it doesn't exist
func GetAppRunPeer(appRunId string) *AppRunPeer {
	peer, _ := appRunPeers.GetOrCreate(appRunId, func() *AppRunPeer {
		return &AppRunPeer{
			AppRunId:      appRunId,
			Logs:          utilds.MakeCirBuf[ds.LogLine](LogLineBufferSize),
			GoRoutines:    utilds.MakeSyncMap[GoRoutine](),
			Watches:       utilds.MakeSyncMap[Watch](),
			ActiveWatches: make(map[string]bool),
			RuntimeStats:  utilds.MakeCirBuf[ds.RuntimeStatsInfo](RuntimeStatsBufferSize),
			Status:        AppStatusRunning,
			LastModTime:   time.Now().UnixMilli(),
		}
	})

	return peer
}

// GetAllAppRunPeers returns all AppRunPeers
func GetAllAppRunPeers() []*AppRunPeer {
	// Get all keys from the sync map
	keys := appRunPeers.Keys()

	// Create a slice to hold all peers
	peers := make([]*AppRunPeer, 0, len(keys))

	// Get each peer and add it to the slice
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
	// Get all app run peers
	appRunPeers := GetAllAppRunPeers()

	// Convert to AppRunInfo slice
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
	// Update the last modification time to the current time in milliseconds
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

	case ds.PacketTypeLog:
		var logLine ds.LogLine
		if err := json.Unmarshal(packetData, &logLine); err != nil {
			return fmt.Errorf("failed to unmarshal LogLine: %w", err)
		}

		// Add log line to circular buffer
		p.Logs.Write(logLine)

	case ds.PacketTypeGoroutine:
		var goroutineInfo ds.GoroutineInfo
		if err := json.Unmarshal(packetData, &goroutineInfo); err != nil {
			return fmt.Errorf("failed to unmarshal GoroutineInfo: %w", err)
		}

		// Create a new map for active goroutines in this packet
		activeGoroutines := make(map[int]bool)

		// Process goroutine stacks
		for _, stack := range goroutineInfo.Stacks {
			goId := int(stack.GoId)
			goIdStr := strconv.Itoa(goId)

			// Mark this goroutine as active
			activeGoroutines[goId] = true

			// Get or create goroutine entry in the syncmap
			goroutine, exists := p.GoRoutines.GetEx(goIdStr)
			if !exists {
				// New goroutine
				goroutine = GoRoutine{
					GoId:        goId,
					StackTraces: utilds.MakeCirBuf[ds.GoRoutineStack](GoRoutineStackBufferSize),
				}
			}
			// Add stack trace to the circular buffer
			goroutine.StackTraces.Write(stack)

			// Update the goroutine in the syncmap
			p.GoRoutines.Set(goIdStr, goroutine)
		}

		// Update the active goroutines map
		p.ActiveGoRoutines = activeGoroutines

		log.Printf("Processed %d goroutines for app run ID: %s", len(goroutineInfo.Stacks), p.AppRunId)

	case ds.PacketTypeWatch:
		var watchInfo ds.WatchInfo
		if err := json.Unmarshal(packetData, &watchInfo); err != nil {
			return fmt.Errorf("failed to unmarshal WatchInfo: %w", err)
		}

		// Create a new map for active watches in this packet
		activeWatches := make(map[string]bool)

		// Process watch values
		for _, watchVal := range watchInfo.Watches {
			watchName := watchVal.Name

			// Mark this watch as active
			activeWatches[watchName] = true

			// Get or create watch entry in the syncmap
			watch, exists := p.Watches.GetEx(watchName)
			if !exists {
				// New watch
				watch = Watch{
					Name:      watchName,
					WatchVals: utilds.MakeCirBuf[ds.Watch](WatchBufferSize),
				}
			}
			// Add watch value to the circular buffer
			watch.WatchVals.Write(watchVal)

			// Update the watch in the syncmap
			p.Watches.Set(watchName, watch)
		}

		// Update the active watches map
		p.ActiveWatches = activeWatches

		log.Printf("Processed %d watches for app run ID: %s", len(watchInfo.Watches), p.AppRunId)

	case ds.PacketTypeAppDone:
		p.Status = AppStatusDone
		log.Printf("Received AppDone for app run ID: %s", p.AppRunId)

	case ds.PacketTypeRuntimeStats:
		var runtimeStats ds.RuntimeStatsInfo
		if err := json.Unmarshal(packetData, &runtimeStats); err != nil {
			return fmt.Errorf("failed to unmarshal RuntimeStatsInfo: %w", err)
		}

		// Add runtime stats to circular buffer
		p.RuntimeStats.Write(runtimeStats)

		log.Printf("Received runtime stats for app run ID: %s", p.AppRunId)

	default:
		log.Printf("Unknown packet type: %s", packetType)
	}

	return nil
}

// SetConnectionClosed marks the peer as disconnected
func (p *AppRunPeer) SetConnectionClosed() {
	if p.Status != AppStatusDone {
		p.Status = AppStatusDisconnected
		p.LastModTime = time.Now().UnixMilli()
		log.Printf("Connection closed for app run ID: %s, marked as disconnected", p.AppRunId)
	}
}

// GetAppRunInfo constructs and returns an AppRunInfo struct for this peer
func (p *AppRunPeer) GetAppRunInfo() rpctypes.AppRunInfo {
	// Skip peers with no AppInfo
	if p.AppInfo == nil {
		return rpctypes.AppRunInfo{}
	}

	// Determine if the app is still running based on its status
	isRunning := p.Status == AppStatusRunning

	// Get the number of active and total goroutines
	numActiveGoRoutines := len(p.ActiveGoRoutines)
	numTotalGoRoutines := len(p.GoRoutines.Keys())

	// Get the number of active and total watches
	numActiveWatches := len(p.ActiveWatches)
	numTotalWatches := len(p.Watches.Keys())

	// Create AppRunInfo
	appRunInfo := rpctypes.AppRunInfo{
		AppRunId:            p.AppRunId,
		AppName:             p.AppInfo.AppName,
		StartTime:           p.AppInfo.StartTime,
		IsRunning:           isRunning,
		Status:              p.Status,
		NumLogs:             p.Logs.Size(),
		NumActiveGoRoutines: numActiveGoRoutines,
		NumTotalGoRoutines:  numTotalGoRoutines,
		NumActiveWatches:    numActiveWatches,
		NumTotalWatches:     numTotalWatches,
		LastModTime:         p.LastModTime,
	}

	// Add build info if available
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
