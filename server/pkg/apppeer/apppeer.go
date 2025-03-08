// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
)

// AppRunPeer represents a peer connection to an app client
type AppRunPeer struct {
	AppRunId string
	AppInfo  *ds.AppInfo
}

// Global synchronized map to hold all AppRunPeers
var appRunPeers = utilds.MakeSyncMap[*AppRunPeer]()

// GetAppRunPeer gets an existing AppRunPeer by ID or creates a new one if it doesn't exist
func GetAppRunPeer(appRunId string) *AppRunPeer {
	peer, _ := appRunPeers.GetOrCreate(appRunId, func() *AppRunPeer {
		return &AppRunPeer{
			AppRunId: appRunId,
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
		
		// Process log line
		optNewLine := ""
		if !strings.HasSuffix(logLine.Msg, "\n") {
			optNewLine = "\n"
		}
		fmt.Printf("logline: %s %d %s%s", logLine.Source, logLine.LineNum, logLine.Msg, optNewLine)
		
	default:
		log.Printf("Unknown packet type: %s", packetType)
	}
	
	return nil
}
