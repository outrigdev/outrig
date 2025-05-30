// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/server/pkg/logutil"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
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
			Fmt:     baseSample.Fmt,
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

func (wp *WatchesPeer) getWatchByName_nolock(name string) *Watch {
	watchNum, exists := wp.nameToWatchNum[name]
	if !exists {
		return nil // Watch not found
	}
	watch, exists := wp.watches.GetEx(watchNum)
	if !exists {
		return nil // Watch not found
	}
	return &watch
}

// ProcessWatchInfo processes watch information from a packet
func (wp *WatchesPeer) ProcessWatchInfo(watchInfo ds.WatchInfo) {
	wp.lock.Lock()
	defer wp.lock.Unlock()

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
	for _, decl := range watchInfo.Decls {
		wp.getOrCreateWatch_nolock(decl)
	}

	// Process watch samples
	for _, sample := range watchInfo.Watches {
		watch := wp.getWatchByName_nolock(sample.Name)
		if watch == nil {
			logKey := fmt.Sprintf("watches-nosample-%s", wp.appRunId)
			logutil.LogfOnce(logKey, "WARNING: [AppRun: %s] No watch found for sample %s in watch info\n", wp.appRunId, sample.Name)
			continue // Skip this sample if no watch is found
		}
		// Handle watch value updates based on whether the sample is marked as "same"
		if watchInfo.Delta && sample.Same {
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
			// Full update or changed sample, write the sample directly
			watch.WatchVals.Write(sample)
		}
	}
}

// GetActiveWatchCount returns the number of active watches
func (wp *WatchesPeer) GetActiveWatchCount() int {
	return wp.GetTotalWatchCount()
}

// GetTotalWatchCount returns the total number of watches (active and inactive)
func (wp *WatchesPeer) GetTotalWatchCount() int {
	return len(wp.watches.Keys())
}

// GetAllWatches returns all watches with their most recent values as combined samples
func (wp *WatchesPeer) GetAllWatches() []rpctypes.CombinedWatchSample {
	watchNums := wp.watches.Keys()
	return wp.GetWatchesByIds(watchNums)
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

// GetWatchesByIds returns watches for specific watch IDs
func (wp *WatchesPeer) GetWatchesByIds(watchIds []int64) []rpctypes.CombinedWatchSample {
	result := make([]rpctypes.CombinedWatchSample, 0, len(watchIds))
	for _, watchId := range watchIds {
		watch, exists := wp.watches.GetEx(watchId)
		if !exists {
			continue
		}
		latestSample, _, exists := watch.WatchVals.GetLast()
		if !exists {
			continue
		}
		result = append(result, rpctypes.CombinedWatchSample{
			WatchNum: watch.WatchNum,
			Decl:     watch.Decl,
			Sample:   latestSample,
		})
	}
	return result
}
