// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
)

const WatchBufferSize = 600 // 10 minutes of 1-second samples

// Watch represents a single watch with its values
type Watch struct {
	WatchNum  int64
	Name      string
	Tags      []string
	WatchVals *utilds.CirBuf[ds.WatchSample]
}

// WatchesPeer manages watches for an AppRunPeer
type WatchesPeer struct {
	watches       *utilds.SyncMap[Watch]
	activeWatches map[string]bool // Tracks currently active watches
	watchNum      int64           // Counter for watch numbers
	lock          sync.RWMutex    // Lock for synchronizing watch operations
}

// MakeWatchesPeer creates a new WatchesPeer instance
func MakeWatchesPeer() *WatchesPeer {
	return &WatchesPeer{
		watches:       utilds.MakeSyncMap[Watch](),
		activeWatches: make(map[string]bool),
		watchNum:      0,
	}
}

// ProcessWatchValues processes watch values from a packet
func (wp *WatchesPeer) ProcessWatchValues(watchValues []ds.WatchSample) {
	wp.lock.Lock()
	defer wp.lock.Unlock()

	// Create a new map for active watches in this packet
	activeWatches := make(map[string]bool)

	// Process watch values
	for _, watchVal := range watchValues {
		watchName := watchVal.Name

		// Mark this watch as active
		activeWatches[watchName] = true

		// Get or create watch entry in the syncmap atomically
		watch, exists := wp.watches.GetOrCreate(watchName, func() Watch {
			// New watch - assign a new watch number
			wp.watchNum++
			watchNum := wp.watchNum

			return Watch{
				WatchNum:  watchNum,
				Name:      watchVal.Name,
				Tags:      watchVal.Tags,
				WatchVals: utilds.MakeCirBuf[ds.WatchSample](WatchBufferSize),
			}
		})

		if exists {
			// Update tags from the watch value
			watch.Tags = watchVal.Tags
		}

		// Add watch value to the circular buffer
		watch.WatchVals.Write(watchVal)

		// Update the watch in the syncmap
		wp.watches.Set(watchName, watch)
	}

	// Update the active watches map
	wp.activeWatches = activeWatches
}

// GetActiveWatchCount returns the number of active watches
func (wp *WatchesPeer) GetActiveWatchCount() int {
	wp.lock.RLock()
	defer wp.lock.RUnlock()
	return len(wp.activeWatches)
}

// GetTotalWatchCount returns the total number of watches (active and inactive)
func (wp *WatchesPeer) GetTotalWatchCount() int {
	return len(wp.watches.Keys())
}

// GetWatches returns all watches
func (wp *WatchesPeer) GetWatches() *utilds.SyncMap[Watch] {
	return wp.watches
}

// GetAllWatches returns all watches with their most recent values
func (wp *WatchesPeer) GetAllWatches() []ds.WatchSample {
	// Get all watch names
	watchNames := wp.watches.Keys()

	// Create a slice to hold all watch data
	watches := make([]ds.WatchSample, 0, len(watchNames))

	// For each watch, get the most recent value
	for _, name := range watchNames {
		watch, exists := wp.watches.GetEx(name)
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

	return watches
}
