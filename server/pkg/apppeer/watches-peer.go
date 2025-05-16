// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"

	watchcollector "github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/logutil"
)

const WatchBufferSize = 600 // 10 minutes of 1-second samples

// Watch represents a single watch with its values
type Watch struct {
	WatchNum  int64
	Decl      ds.WatchDecl
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
// When a sample is marked as "same", it means all fields are the same as the previous sample
// and were cleared in the delta to save bandwidth
func mergeWatchSamples(baseSample, deltaSample ds.WatchSample) ds.WatchSample {
	// If the sample is marked as "same", copy all fields from the base sample
	if deltaSample.Same {
		// Create a new sample with the name and timestamp from the delta
		// but all other fields from the base sample
		return ds.WatchSample{
			Name:    deltaSample.Name,
			Ts:      deltaSample.Ts,
			PollDur: deltaSample.PollDur,
			Same:    false, // Reset Same flag as this is now a complete sample
			Kind:    baseSample.Kind,
			Type:    baseSample.Type,
			Val:     baseSample.Val,
			Error:   baseSample.Error,
			Addr:    baseSample.Addr,
			Cap:     baseSample.Cap,
			Len:     baseSample.Len,
		}
	}

	// Not marked as same, so the delta sample already contains all necessary fields
	return deltaSample
}

// getOrCreateWatch_nolock gets or creates a watch by name
// Assumes the lock is already held
func (wp *WatchesPeer) getOrCreateWatch_nolock(watchDecl ds.WatchDecl) (Watch, int64) {
	watchName := watchDecl.Name
	watchNum, exists := wp.nameToWatchNum[watchName]
	if !exists {
		// Create a new watch with a new number
		wp.watchNum++
		watchNum = wp.watchNum
		wp.nameToWatchNum[watchName] = watchNum

		wp.watches.Set(watchNum, Watch{
			WatchNum:  watchNum,
			Decl:      watchDecl,
			WatchVals: utilds.MakeCirBuf[ds.WatchSample](WatchBufferSize),
		})
	} else {
		// Update the declaration for existing watch
		watch, _ := wp.watches.GetEx(watchNum)
		watch.Decl = watchDecl
		wp.watches.Set(watchNum, watch)
	}

	// Get the watch (it must exist now)
	watch, _ := wp.watches.GetEx(watchNum)
	return watch, watchNum
}

// ProcessWatchInfo processes watch information from a packet
func (wp *WatchesPeer) ProcessWatchInfo(watchInfo ds.WatchInfo) {
	wp.lock.Lock()
	defer wp.lock.Unlock()

	activeWatches := make(map[int64]bool)

	// If this is a delta update but we haven't seen a full update yet, ignore it
	if watchInfo.Delta && !wp.hasSeenFullUpdate {
		fmt.Printf("WARNING: [AppRun: %s] Ignoring delta update because no full update has been seen yet\n", wp.appRunId)
		return
	}

	// If this is a full update, mark that we've seen one
	if !watchInfo.Delta {
		wp.hasSeenFullUpdate = true
	}

	// Process watch declarations first
	declMap := make(map[string]ds.WatchDecl)
	for _, decl := range watchInfo.Decls {
		declMap[decl.Name] = decl
	}

	// Process watch samples
	for _, sample := range watchInfo.Watches {
		// Get the declaration for this watch
		decl, declExists := declMap[sample.Name]
		if !declExists {
			// If we don't have a declaration in this update, try to get it from existing watches
			if watchNum, exists := wp.nameToWatchNum[sample.Name]; exists {
				watch, watchExists := wp.watches.GetEx(watchNum)
				if watchExists {
					decl = watch.Decl
					declExists = true
				}
			}

			// If we still don't have a declaration, create a minimal one
			if !declExists {
				decl = ds.WatchDecl{
					Name:      sample.Name,
					WatchType: sample.Type,
				}
			}
		}

		// Get or create the watch
		watch, watchNum := wp.getOrCreateWatch_nolock(decl)
		activeWatches[watchNum] = true

		// Handle watch value updates based on whether it's a delta update
		isPush := decl.Format == watchcollector.WatchType_Push // Equivalent to the old IsPush() check

		if watchInfo.Delta && !isPush { // Push watches are always full updates
			// Delta updates need a base sample to merge with
			lastSample, _, lastExists := watch.WatchVals.GetLast()
			if lastExists {
				completeSample := mergeWatchSamples(lastSample, sample)
				watch.WatchVals.Write(completeSample)
			} else {
				logKey := fmt.Sprintf("watches-nodeltaupdate-%s", wp.appRunId)
				logutil.LogfOnce(logKey, "WARNING: [AppRun: %s] Delta update received for watch %s with no last sample\n", wp.appRunId, sample.Name)
			}
		} else {
			// Full update, write the sample directly
			watch.WatchVals.Write(sample)
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
func (wp *WatchesPeer) GetAllWatches() []ds.WatchInfo {
	watchNums := wp.watches.Keys()

	// Group watches by active status
	activeWatches := make([]ds.WatchSample, 0, len(wp.activeWatches))
	activeDecls := make([]ds.WatchDecl, 0, len(wp.activeWatches))

	for _, num := range watchNums {
		watch, exists := wp.watches.GetEx(num)
		if !exists {
			continue
		}

		latestWatch, _, exists := watch.WatchVals.GetLast()
		if !exists {
			continue
		}

		// Only include active watches
		if _, isActive := wp.activeWatches[num]; isActive {
			activeWatches = append(activeWatches, latestWatch)
			activeDecls = append(activeDecls, watch.Decl)
		}
	}

	// Create a single WatchInfo with all watches
	return []ds.WatchInfo{
		{
			Ts:      activeWatches[0].Ts, // Use timestamp from first watch
			Delta:   false,
			Decls:   activeDecls,
			Watches: activeWatches,
		},
	}
}

// getNumericVal returns a float64 representation of a WatchSample value
func getNumericVal(sample ds.WatchSample) float64 {
	if sample.Error != "" {
		return 0
	}

	kind := reflect.Kind(sample.Kind)
	switch kind {
	case reflect.Bool:
		if sample.Val == "true" {
			return 1
		}
		return 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(sample.Val, 64)
		if err != nil {
			return 0
		}
		return val
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return float64(sample.Len)
	default:
		return 0
	}
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
		numericValues = append(numericValues, getNumericVal(sample))
	}

	// Check if this is a counter
	if watch.Decl.Counter {
		// For counters, convert to deltas
		return utilfn.CalculateDeltas(numericValues)
	}

	return numericValues
}

// GetWatchesByIds returns watches for specific watch IDs
func (wp *WatchesPeer) GetWatchesByIds(watchIds []int64) ds.WatchInfo {
	// Get watches by their IDs directly
	samples := make([]ds.WatchSample, 0, len(watchIds))
	decls := make([]ds.WatchDecl, 0, len(watchIds))

	for _, watchId := range watchIds {
		watch, exists := wp.watches.GetEx(watchId)
		if !exists {
			continue
		}

		latestWatch, _, exists := watch.WatchVals.GetLast()
		if !exists {
			continue
		}

		samples = append(samples, latestWatch)
		decls = append(decls, watch.Decl)
	}

	// Sort watches by name for consistent ordering
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].Name < samples[j].Name
	})

	// Sort declarations to match
	sort.Slice(decls, func(i, j int) bool {
		return decls[i].Name < decls[j].Name
	})

	// If no watches found, return empty WatchInfo
	if len(samples) == 0 {
		return ds.WatchInfo{}
	}

	// Create a WatchInfo with the requested watches
	return ds.WatchInfo{
		Ts:      samples[0].Ts, // Use timestamp from first watch
		Delta:   false,
		Decls:   decls,
		Watches: samples,
	}
}

// GetWatchHistory returns the full history of samples for a specific watch
func (wp *WatchesPeer) GetWatchHistory(watchNum int64) ds.WatchInfo {
	// Get the watch directly by its number
	watch, exists := wp.watches.GetEx(watchNum)
	if !exists {
		return ds.WatchInfo{}
	}

	// Get all samples from the circular buffer
	samples, _ := watch.WatchVals.GetAll()
	if len(samples) == 0 {
		return ds.WatchInfo{}
	}

	// Create a WatchInfo with the watch history
	return ds.WatchInfo{
		Ts:      samples[0].Ts, // Use timestamp from first sample
		Delta:   false,
		Decls:   []ds.WatchDecl{watch.Decl},
		Watches: samples,
	}
}
