// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"github.com/outrigdev/outrig/pkg/utilds"
)

// AppRunPeer represents a peer connection to an app client
type AppRunPeer struct {
	AppRunId string
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
