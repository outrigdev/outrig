// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import (
	"fmt"
	"sync"
)

// TimeSampleAligner maps real timestamps to logical time indices while maintaining
// timing accuracy. Logical time starts at 0 and increments by 1 for each second.
// It tracks global skew to handle timing drift and boundary conditions.
type TimeSampleAligner struct {
	lock           sync.Mutex
	logicalCounter int                // Current logical time counter
	timestamps     map[int]int64      // Map of logical time -> real timestamp
	globalSkew     int64              // Accumulated timing drift (milliseconds)
	lastRealTs     int64              // Last real timestamp received
	firstTs        int64              // First timestamp (never cleaned up)
	maxSamples     int                // Maximum number of logical samples to keep
	hasFirstSample bool               // Whether we've received the first sample
}

// MakeTimeSampleAligner creates a new TimeSampleAligner instance
func MakeTimeSampleAligner(maxSamples int) *TimeSampleAligner {
	return &TimeSampleAligner{
		timestamps: make(map[int]int64),
		maxSamples: maxSamples,
	}
}

// AddSample takes a timestamp in milliseconds and returns its logical time.
// The first call always returns 0. Subsequent calls use skew-based algorithm
// to maintain timing accuracy while handling gaps and timing drift.
// Returns an error if sample is dropped or timestamp goes backward.
func (tsa *TimeSampleAligner) AddSample(ts int64) (int, error) {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()
	defer tsa.cleanupOldSamples()

	// First sample always maps to logical time 0
	if !tsa.hasFirstSample {
		tsa.hasFirstSample = true
		tsa.logicalCounter = 0
		tsa.timestamps[0] = ts
		tsa.lastRealTs = ts
		tsa.firstTs = ts
		tsa.globalSkew = 0
		return 0, nil
	}

	// Reject samples that go backward in time
	if ts < tsa.lastRealTs {
		return 0, fmt.Errorf("timestamp %d is less than previous timestamp %d", ts, tsa.lastRealTs)
	}

	// Drop samples that arrive < 500ms after previous sample
	if ts-tsa.lastRealTs < 500 {
		return 0, fmt.Errorf("sample dropped: timestamp %d is too close to previous timestamp %d", ts, tsa.lastRealTs)
	}

	// Calculate time difference and how many logical slots to advance
	timeDiff := ts - tsa.lastRealTs
	logicalSlotsToAdvance := int((timeDiff + 500) / 1000) // Round to nearest second

	// Fill gaps with synthetic timestamps for missing logical slots
	// Don't advance logicalCounter yet - just fill the gaps
	for i := 1; i <= logicalSlotsToAdvance; i++ {
		gapLogical := tsa.logicalCounter + i
		syntheticTs := tsa.lastRealTs + int64(i*1000)
		tsa.timestamps[gapLogical] = syntheticTs
	}

	// Calculate expected timestamp for this sample based on logical advancement
	expectedTs := tsa.lastRealTs + int64(logicalSlotsToAdvance*1000)
	localSkew := ts - expectedTs

	// Accumulate local skew into global skew
	tsa.globalSkew += localSkew

	// Determine where to place this sample based on global skew
	targetLogical := tsa.logicalCounter + logicalSlotsToAdvance

	// Handle global skew corrections
	if tsa.globalSkew >= 1000 {
		// Too far ahead - insert synthetic timestamp to slow down, then place sample
		tsa.globalSkew -= 1000
		targetLogical++
		syntheticTs := (tsa.lastRealTs + ts) / 2 // Average of last and current
		tsa.timestamps[targetLogical-1] = syntheticTs
		tsa.timestamps[targetLogical] = ts
		tsa.logicalCounter = targetLogical
		tsa.lastRealTs = ts
		return targetLogical, nil
	} else if tsa.globalSkew <= -1000 {
		// Too far behind - drop sample and reset skew (don't advance logical counter)
		tsa.globalSkew += 1000
		tsa.lastRealTs = ts
		return 0, fmt.Errorf("sample dropped: global skew correction (too far behind)")
	}

	// Normal case - place the actual sample at the target logical time
	tsa.logicalCounter = targetLogical
	tsa.timestamps[targetLogical] = ts
	tsa.lastRealTs = ts
	return targetLogical, nil
}

// cleanupOldSamples removes old logical time entries to stay under maxSamples
func (tsa *TimeSampleAligner) cleanupOldSamples() {
	if len(tsa.timestamps) <= tsa.maxSamples {
		return
	}

	// Find the oldest logical time to keep
	keepFromLogical := tsa.logicalCounter - tsa.maxSamples + 1
	
	// Remove entries older than keepFromLogical
	for logical := range tsa.timestamps {
		if logical < keepFromLogical {
			delete(tsa.timestamps, logical)
		}
	}
}

// GetTimestamp returns the real timestamp for a given logical time
func (tsa *TimeSampleAligner) GetTimestamp(logicalTime int) (int64, bool) {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()
	
	if !tsa.hasFirstSample {
		return 0, false
	}
	
	// If we have the actual timestamp, return it
	if ts, exists := tsa.timestamps[logicalTime]; exists {
		return ts, true
	}
	
	// Otherwise calculate it from first timestamp
	calculatedTs := tsa.firstTs + int64(logicalTime*1000)
	return calculatedTs, true
}

// GetLogicalTime returns the logical time for a given real timestamp
// Uses the constraint that logical time N is within ±1000ms of N seconds from first sample
func (tsa *TimeSampleAligner) GetLogicalTime(timestamp int64) int {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()
	
	if !tsa.hasFirstSample {
		return 0
	}
	
	// Calculate approximate logical time (within ±1 due to skew constraint)
	approxLogical := int((timestamp - tsa.firstTs) / 1000)
	
	// Check a small range around the approximate logical time
	for logical := approxLogical - 1; logical <= approxLogical + 1; logical++ {
		if logical < 0 || logical > tsa.logicalCounter {
			continue
		}
		
		currentTs, exists := tsa.timestamps[logical]
		if !exists {
			continue // Skip cleaned up entries
		}
		
		// Check if timestamp falls in this logical interval
		if logical == tsa.logicalCounter {
			// Last logical time - timestamp must be >= currentTs
			if timestamp >= currentTs {
				return logical
			}
		} else {
			// Find the next timestamp boundary
			var nextTs int64
			if nextTimestamp, nextExists := tsa.timestamps[logical+1]; nextExists {
				nextTs = nextTimestamp
			} else {
				// Next was cleaned up, estimate based on current + 1s
				nextTs = currentTs + 1000
			}
			
			// Check if timestamp falls in [currentTs, nextTs)
			if timestamp >= currentTs && timestamp < nextTs {
				return logical
			}
		}
	}
	
	// If we get here, timestamp doesn't fall in any retained interval
	// Fall back to simple calculation
	return int((timestamp - tsa.firstTs) / 1000)
}

// GetMaxLogicalTime returns the current maximum logical time
func (tsa *TimeSampleAligner) GetMaxLogicalTime() int {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()
	
	return tsa.logicalCounter
}



