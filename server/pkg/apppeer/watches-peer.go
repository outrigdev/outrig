// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"sort"
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
	watches        *utilds.SyncMap[int64, Watch]
	nameToWatchNum map[string]int64 // Maps watch names to their watch numbers
	activeWatches  map[int64]bool   // Tracks currently active watches by watchnum
	watchNum       int64            // Counter for watch numbers
	lock           sync.RWMutex     // Lock for synchronizing watch operations
}

// MakeWatchesPeer creates a new WatchesPeer instance
func MakeWatchesPeer() *WatchesPeer {
	return &WatchesPeer{
		watches:        utilds.MakeSyncMap[int64, Watch](),
		nameToWatchNum: make(map[string]int64),
		activeWatches:  make(map[int64]bool),
		watchNum:       0,
	}
}

// ProcessWatchValues processes watch values from a packet
func (wp *WatchesPeer) ProcessWatchValues(watchValues []ds.WatchSample) {
	wp.lock.Lock()
	defer wp.lock.Unlock()

	activeWatches := make(map[int64]bool)

	// Process watch values
	for _, watchVal := range watchValues {
		watchName := watchVal.Name

		// Look up the watch number for this name
		watchNum, exists := wp.nameToWatchNum[watchName]
		if !exists {
			// Create a new watch with a new number
			wp.watchNum++
			watchNum = wp.watchNum
			wp.nameToWatchNum[watchName] = watchNum

			watch := Watch{
				WatchNum:  watchNum,
				Name:      watchName,
				Tags:      watchVal.Tags,
				WatchVals: utilds.MakeCirBuf[ds.WatchSample](WatchBufferSize),
			}

			wp.watches.Set(watchNum, watch)
		}

		// Set the WatchNum field in the WatchSample
		watchVal.WatchNum = watchNum

		// Mark this watch as active using its watchnum
		activeWatches[watchNum] = true

		// Get the watch and update it
		watch, exists := wp.watches.GetEx(watchNum)
		if exists {
			watch.Tags = watchVal.Tags
			watch.WatchVals.Write(watchVal)
			wp.watches.Set(watchNum, watch)
		}
	}

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

// GetAllWatches returns all watches with their most recent values
func (wp *WatchesPeer) GetAllWatches() []ds.WatchSample {
	watchNums := wp.watches.Keys()
	watches := make([]ds.WatchSample, 0, len(watchNums))
	for _, num := range watchNums {
		watch, exists := wp.watches.GetEx(num)
		if !exists {
			continue
		}

		latestWatch, _, exists := watch.WatchVals.GetLast()
		if !exists {
			continue
		}

		watches = append(watches, latestWatch)
	}

	return watches
}

// GetWatchesByIds returns watches for specific watch IDs
func (wp *WatchesPeer) GetWatchesByIds(watchIds []int64) []ds.WatchSample {
	// Get watches by their IDs directly
	watches := make([]ds.WatchSample, 0, len(watchIds))
	for _, watchId := range watchIds {
		watch, exists := wp.watches.GetEx(watchId)
		if !exists {
			continue
		}

		latestWatch, _, exists := watch.WatchVals.GetLast()
		if !exists {
			continue
		}

		watches = append(watches, latestWatch)
	}

	// Sort watches by name for consistent ordering
	sort.Slice(watches, func(i, j int) bool {
		return watches[i].Name < watches[j].Name
	})

	return watches
}
