// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
)

const LogLineBufferSize = 10000
const GoRoutineStackBufferSize = 600 // 10 minutes of 1-second samples

// AppRunPeer represents a peer connection to an app client
type AppRunPeer struct {
	AppRunId         string
	AppInfo          *ds.AppInfo
	Logs             *utilds.CirBuf[ds.LogLine]
	GoRoutines       *utilds.SyncMap[GoRoutine]
	ActiveGoRoutines map[int]bool // Tracks currently running goroutines
}

type GoRoutine struct {
	GoId        int
	StackTraces *utilds.CirBuf[ds.GoRoutineStack]
}

// Global synchronized map to hold all AppRunPeers
var appRunPeers = utilds.MakeSyncMap[*AppRunPeer]()

// GetAppRunPeer gets an existing AppRunPeer by ID or creates a new one if it doesn't exist
func GetAppRunPeer(appRunId string) *AppRunPeer {
	peer, _ := appRunPeers.GetOrCreate(appRunId, func() *AppRunPeer {
		return &AppRunPeer{
			AppRunId:   appRunId,
			Logs:       utilds.MakeCirBuf[ds.LogLine](LogLineBufferSize),
			GoRoutines: utilds.MakeSyncMap[GoRoutine](),
		}
	})

	return peer
}

// HandlePacket processes a packet received from the domain socket connection
func (p *AppRunPeer) HandlePacket(packetType string, packetData json.RawMessage) error {
	switch packetType {
	case ds.PacketTypeAppInfo:
		var appInfo ds.AppInfo
		if err := json.Unmarshal(packetData, &appInfo); err != nil {
			return fmt.Errorf("failed to unmarshal AppInfo: %w", err)
		}
		p.AppInfo = &appInfo
		log.Printf("Received AppInfo for app run ID: %s, app: %s", p.AppRunId, appInfo.AppName)

	case ds.PacketTypeLog:
		var logLine ds.LogLine
		if err := json.Unmarshal(packetData, &logLine); err != nil {
			return fmt.Errorf("failed to unmarshal LogLine: %w", err)
		}

		// Add log line to circular buffer
		p.Logs.Write(logLine)

		fmt.Printf("got logline: %s %d %q\n", logLine.Source, logLine.LineNum, logLine.Msg)

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

	default:
		log.Printf("Unknown packet type: %s", packetType)
	}

	return nil
}
