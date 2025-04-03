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

	activeWatches := make(map[string]bool)

	// Process watch values
	for _, watchVal := range watchValues {
		watchName := watchVal.Name

		activeWatches[watchName] = true

		watch, exists := wp.watches.GetOrCreate(watchName, func() Watch {
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
			watch.Tags = watchVal.Tags
		}

		watch.WatchVals.Write(watchVal)

		wp.watches.Set(watchName, watch)
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

// GetWatches returns all watches
func (wp *WatchesPeer) GetWatches() *utilds.SyncMap[Watch] {
	return wp.watches
}

// GetAllWatches returns all watches with their most recent values
func (wp *WatchesPeer) GetAllWatches() []ds.WatchSample {
	watchNames := wp.watches.Keys()
	watches := make([]ds.WatchSample, 0, len(watchNames))
	for _, name := range watchNames {
		watch, exists := wp.watches.GetEx(name)
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

// GetWatchNameToNumMap returns a map of watch names to their corresponding watch numbers
func (wp *WatchesPeer) GetWatchNameToNumMap() map[string]int64 {
	watchNameToNum := make(map[string]int64)
	watchNames := wp.watches.Keys()
	
	for _, name := range watchNames {
		watch, exists := wp.watches.GetEx(name)
		if !exists {
			continue
		}
		watchNameToNum[name] = watch.WatchNum
	}
	
	return watchNameToNum
}

// GetWatchesByIds returns watches for specific watch IDs
func (wp *WatchesPeer) GetWatchesByIds(watchIds []int64) []ds.WatchSample {
	// Build a map of WatchNum to watch name for efficient lookups
	watchNumToName := make(map[int64]string)
	watchNames := wp.watches.Keys()
	
	for _, name := range watchNames {
		watch, exists := wp.watches.GetEx(name)
		if !exists {
			continue
		}
		watchNumToName[watch.WatchNum] = name
	}
	
	// Get watches by their IDs using the map
	watches := make([]ds.WatchSample, 0, len(watchIds))
	for _, watchId := range watchIds {
		name, exists := watchNumToName[watchId]
		if !exists {
			continue
		}
		
		watch, exists := wp.watches.GetEx(name)
		if !exists {
			continue
		}
		
		latestWatch, _, exists := watch.WatchVals.GetLast()
		if !exists {
			continue
		}
		
		watches = append(watches, latestWatch)
	}
	
	// Sort watches by ID for consistent ordering
	sort.Slice(watches, func(i, j int) bool {
		return watches[i].Name < watches[j].Name
	})
	
	return watches
}
