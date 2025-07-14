// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import (
	"errors"
	"fmt"
	"sync"
)

var ErrSampleTooClose = errors.New("sample too close to previous timestamp")
var ErrSampleSkipped = errors.New("sample skipped due to timing skew")

// TimeSampleAligner converts real-world timestamps (which may have skew, gaps, or
// timing drift) into clean logical time indices for frontend consumption.
//
// Key features:
//   - Maps incoming samples to logical time slots (0, 1, 2, 3...)
//   - Maintains timing accuracy within ±1000ms of real timestamps
//   - Handles clock drift and power-sleep gaps gracefully
//   - Uses ring buffer for efficient memory management
//   - Provides both logical indices and real timestamps for graphing
//
// Algorithm overview:
//   - Tracks running time skew between expected and actual sample timing
//   - When skew exceeds ±1000ms, corrects by either filling gaps or dropping samples
//   - For gaps (skew >= +1000ms): interpolates synthetic timestamps and advances logical time
//   - For samples arriving too fast (skew <= -1000ms): drops sample and resets skew
//   - Maintains invariant: abs(timeSkew) < 1000ms for timing accuracy
//
// Usage pattern:
//
//	aligner := MakeTimeSampleAligner2(maxSamples)
//	logicalTime, err := aligner.AddSample(timestampMs)
//	baseLogical, timestamps := aligner.GetTimestamps()
//	// Frontend can use logical indices 0,1,2... while mapping back to real timestamps
type TimeSampleAligner struct {
	lock           sync.Mutex
	firstTs        int64
	lastRealTs     int64
	timeSkew       int64
	logicalCounter int
	maxSamples     int
	ringBuffer     []int64
	baseLogical    int
}

func MakeTimeSampleAligner(maxSamples int) *TimeSampleAligner {
	return &TimeSampleAligner{
		maxSamples:  maxSamples,
		ringBuffer:  make([]int64, 0, maxSamples),
		baseLogical: 0,
	}
}

func (tsa *TimeSampleAligner) addSkipTs(ts int64) (int, error) {
	ideal := (ts - tsa.firstTs + 500) / 1000
	idealSlot := int(ideal) + tsa.baseLogical
	idealTs := tsa.firstTs + int64(idealSlot-tsa.baseLogical)*1000
	newSkew := ts - idealTs

	if idealSlot == tsa.logicalCounter {
		// sanity check, but the math should guarantee that this never happens
		return tsa.logicalCounter, fmt.Errorf("invalid ideal slot %d, already at %d", idealSlot, tsa.logicalCounter)
	}

	skipSlots := idealSlot - tsa.logicalCounter - 1
	if skipSlots > 0 {
		// Interpolate intermediate timestamps across the gap
		timeGap := ts - tsa.lastRealTs
		totalSlots := skipSlots + 1 // +1 to include the final slot

		for i := 1; i <= skipSlots; i++ {
			interpolatedTs := tsa.lastRealTs + (timeGap*int64(i))/int64(totalSlots)
			tsa.appendSlot(tsa.logicalCounter+i, interpolatedTs)
		}
	}
	// because idealSlot > ts.logicalCounter we should always be able to append
	tsa.appendSlot(idealSlot, ts)
	tsa.logicalCounter = idealSlot
	tsa.lastRealTs = ts
	tsa.timeSkew = newSkew

	return tsa.logicalCounter, nil
}

func (tsa *TimeSampleAligner) AddSample(ts int64) (int, error) {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()

	if len(tsa.ringBuffer) == 0 {
		tsa.firstTs = ts
		tsa.lastRealTs = ts
		tsa.appendSlot(0, ts)
		return 0, nil
	}
	if ts < tsa.lastRealTs {
		return 0, fmt.Errorf("timestamp %d < previous %d", ts, tsa.lastRealTs)
	}
	if ts-tsa.lastRealTs < 500 {
		return tsa.logicalCounter, ErrSampleTooClose
	}
	delta := ts - tsa.lastRealTs
	newSkew := tsa.timeSkew + delta - 1000
	if newSkew >= 1000 {
		return tsa.addSkipTs(ts)
	}
	if newSkew <= -1000 {
		// skip and reset skew
		tsa.timeSkew = newSkew + 1000
		return tsa.logicalCounter, ErrSampleSkipped
	}
	// normal case, just update the last timestamp
	tsa.appendSlot(tsa.logicalCounter+1, ts)
	tsa.logicalCounter++
	tsa.lastRealTs = ts
	tsa.timeSkew = newSkew
	return tsa.logicalCounter, nil
}

func (tsa *TimeSampleAligner) appendSlot(logical int, realTs int64) {
	tsa.ringBuffer = append(tsa.ringBuffer, realTs)
	if len(tsa.ringBuffer) > tsa.maxSamples {
		tsa.ringBuffer = tsa.ringBuffer[1:]
		tsa.baseLogical++
	}
}

func (tsa *TimeSampleAligner) GetTimestamps() (int, []int64) {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()
	timestamps := make([]int64, len(tsa.ringBuffer))
	copy(timestamps, tsa.ringBuffer)
	return tsa.baseLogical, timestamps
}

func (tsa *TimeSampleAligner) GetMaxLogicalTime() int {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()
	if len(tsa.ringBuffer) == 0 {
		return 0
	}
	return tsa.baseLogical + len(tsa.ringBuffer) - 1
}

func (tsa *TimeSampleAligner) GetRealTimestampFromLogical(logical int) int64 {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()
	// if not in the ring buffer, return just a stright up delta from baseTs
	if logical < tsa.baseLogical || logical >= tsa.baseLogical+len(tsa.ringBuffer) {
		return tsa.firstTs + int64(logical-tsa.baseLogical)*1000
	}
	// find the timestamp in the ring buffer
	return tsa.ringBuffer[logical-tsa.baseLogical]
}

func (tsa *TimeSampleAligner) GetLogicalTimeFromRealTimestamp(ts int64) int {
	tsa.lock.Lock()
	defer tsa.lock.Unlock()

	// No samples yet
	if len(tsa.ringBuffer) == 0 {
		return 0
	}

	// Calculate ideal slot based on firstTs
	idealSlot := int((ts - tsa.firstTs) / 1000)

	// If outside ring buffer range, return calculated ideal
	if idealSlot < tsa.baseLogical {
		return idealSlot
	}
	if idealSlot >= tsa.baseLogical+len(tsa.ringBuffer) {
		return idealSlot
	}

	// Search within ring buffer, starting near the ideal position
	startIdx := max(idealSlot-tsa.baseLogical-1, 0)

	for i := startIdx; i < len(tsa.ringBuffer)-1; i++ {
		if ts >= tsa.ringBuffer[i] && ts < tsa.ringBuffer[i+1] {
			return tsa.baseLogical + i
		}
	}

	// ts >= all timestamps in buffer
	return tsa.baseLogical + len(tsa.ringBuffer) - 1
}
