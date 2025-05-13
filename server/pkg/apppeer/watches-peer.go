// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"fmt"
	"sort"
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/logutil"
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
	watches           *utilds.SyncMap[int64, Watch]
	nameToWatchNum    map[string]int64 // Maps watch names to their watch numbers
	activeWatches     map[int64]bool   // Tracks currently active watches by watchnum
	watchNum          int64            // Counter for watch numbers
	lock              sync.RWMutex     // Lock for synchronizing watch operations
	hasSeenFullUpdate bool             // Flag to track if we've seen a full update
	appRunId          string           // ID of the app run this peer belongs to
}

// MakeWatchesPeer creates a new WatchesPeer instance
func MakeWatchesPeer(appRunId string) *WatchesPeer {
	return &WatchesPeer{
		watches:           utilds.MakeSyncMap[int64, Watch](),
		nameToWatchNum:    make(map[string]int64),
		activeWatches:     make(map[int64]bool),
		watchNum:          0,
		hasSeenFullUpdate: false,
		appRunId:          appRunId,
	}
}

// mergeWatchSamples combines a base watch sample with a delta sample to create a complete sample
// It starts with the delta sample and fills in missing fields from the base sample
func mergeWatchSamples(baseSample, deltaSample ds.WatchSample) ds.WatchSample {
	// Start with the delta sample
	completeSample := deltaSample

	// Fill in fields from base sample if they're empty in the delta
	if completeSample.Flags == 0 {
		completeSample.Flags = baseSample.Flags
	}
	if completeSample.Type == "" {
		completeSample.Type = baseSample.Type
	}
	if completeSample.StrVal == "" {
		completeSample.StrVal = baseSample.StrVal
	}
	if completeSample.GoFmtVal == "" {
		completeSample.GoFmtVal = baseSample.GoFmtVal
	}
	if completeSample.JsonVal == "" {
		completeSample.JsonVal = baseSample.JsonVal
	}
	if len(completeSample.Addr) == 0 {
		completeSample.Addr = baseSample.Addr
	}
	if len(completeSample.Tags) == 0 {
		completeSample.Tags = baseSample.Tags
	}

	return completeSample
}

// getOrCreateWatch_nolock gets or creates a watch by name
// Assumes the lock is already held
func (wp *WatchesPeer) getOrCreateWatch_nolock(watchName string, tags []string) (Watch, int64) {
	watchNum, exists := wp.nameToWatchNum[watchName]
	if !exists {
		// Create a new watch with a new number
		wp.watchNum++
		watchNum = wp.watchNum
		wp.nameToWatchNum[watchName] = watchNum

		wp.watches.Set(watchNum, Watch{
			WatchNum:  watchNum,
			Name:      watchName,
			Tags:      tags,
			WatchVals: utilds.MakeCirBuf[ds.WatchSample](WatchBufferSize),
		})
	}

	// Get the watch (it must exist now)
	watch, _ := wp.watches.GetEx(watchNum)
	return watch, watchNum
}

// ProcessWatchValues processes watch values from a packet
func (wp *WatchesPeer) ProcessWatchValues(watchValues []ds.WatchSample, isDelta bool) {
	wp.lock.Lock()
	defer wp.lock.Unlock()

	activeWatches := make(map[int64]bool)

	// If this is a delta update but we haven't seen a full update yet, ignore it
	if isDelta && !wp.hasSeenFullUpdate {
		fmt.Printf("WARNING: [AppRun: %s] Ignoring delta update because no full update has been seen yet\n", wp.appRunId)
		return
	}

	// If this is a full update, mark that we've seen one
	if !isDelta {
		wp.hasSeenFullUpdate = true
	}

	// Process watch values
	for _, watchVal := range watchValues {
		// Get or create the watch
		watch, watchNum := wp.getOrCreateWatch_nolock(watchVal.Name, watchVal.Tags)
		watchVal.WatchNum = watchNum
		if len(watchVal.Tags) > 0 {
			watch.Tags = watchVal.Tags
		}
		activeWatches[watchNum] = true

		// Handle watch value updates based on whether it's a delta update
		if isDelta && !watchVal.IsPush() { // Push watches are always full updates
			// Delta updates need a base sample to merge with
			lastSample, _, lastExists := watch.WatchVals.GetLast()
			if lastExists {
				completeSample := mergeWatchSamples(lastSample, watchVal)
				watch.WatchVals.Write(completeSample)
			} else {
				logKey := fmt.Sprintf("watches-nodeltaupdate-%s", wp.appRunId)
				logutil.LogfOnce(logKey, "WARNING: [AppRun: %s] Delta update received for watch %s with no last sample\n", wp.appRunId, watchVal.Name)
			}
		} else {
			// Full update, write the sample directly
			watch.WatchVals.Write(watchVal)
		}

		wp.watches.Set(watchNum, watch)
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

// GetWatchNumeric returns an array of numeric values for a specific watch
// If the watch is a counter, it returns deltas between consecutive values
func (wp *WatchesPeer) GetWatchNumeric(watchNum int64) []float64 {
	// Get the watch directly by its number
	watch, exists := wp.watches.GetEx(watchNum)
	if !exists {
		return nil
	}

	// Get all samples from the circular buffer
	samples, _ := watch.WatchVals.GetAll()
	if len(samples) == 0 {
		return nil
	}

	// Convert each sample to a numeric value
	numericValues := make([]float64, 0, len(samples))
	for _, sample := range samples {
		numericValues = append(numericValues, sample.GetNumericVal())
	}

	// Check if this is a counter by examining the flags of the first sample
	// (assuming all samples have the same flags)
	if len(samples) > 0 && (samples[0].Flags&ds.WatchFlag_Counter) != 0 {
		// For counters, convert to deltas
		return utilfn.CalculateDeltas(numericValues)
	}

	return numericValues
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

// GetWatchHistory returns the full history of samples for a specific watch
func (wp *WatchesPeer) GetWatchHistory(watchNum int64) []ds.WatchSample {
	// Get the watch directly by its number
	watch, exists := wp.watches.GetEx(watchNum)
	if !exists {
		return nil
	}

	// Get all samples from the circular buffer
	samples, _ := watch.WatchVals.GetAll()
	if len(samples) == 0 {
		return nil
	}

	return samples
}
