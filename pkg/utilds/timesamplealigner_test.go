// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import (
	"math/rand"
	"testing"
)

func TestTimeSampleAlignerNormalSequence(t *testing.T) {
	tsa := MakeTimeSampleAligner(200)
	baseTime := int64(10500)

	// Add 100 samples, each one roughly 1 second apart but with ±100ms jitter
	for i := 0; i < 100; i++ {
		// Calculate expected time: baseTime + i*1000
		expectedTime := baseTime + int64(i*1000)

		// Add random jitter: ±100ms
		jitter := rand.Int63n(200) - 100 // Random number from -100 to +99
		actualTime := expectedTime + jitter

		logical, err := tsa.AddSample(actualTime)

		if err != nil {
			t.Errorf("Sample %d (time %d) should not error: %v", i, actualTime, err)
		}

		if logical != i {
			t.Errorf("Sample %d: expected logical time %d, got %d", i, i, logical)
		}
	}

	// Verify final state
	maxLogical := tsa.GetMaxLogicalTime()
	if maxLogical != 99 {
		t.Errorf("Expected max logical time 99, got %d", maxLogical)
	}
}

func TestTimeSampleAlignerMappingAndBoundaries(t *testing.T) {
	tsa := MakeTimeSampleAligner(50)

	// Add controlled samples that map to logical times 0, 1, 2, 3
	samples := []struct {
		timestamp       int64
		expectedLogical int
	}{
		{10500, 0}, // First sample always maps to 0
		{11600, 1}, // 1.1 seconds later
		{12400, 2}, // 0.8 seconds later (total 1.9s from first)
		{13550, 3}, // 1.15 seconds later (total 3.05s from first)
	}

	// Add the samples and verify they map to expected logical times
	for i, sample := range samples {
		logical, err := tsa.AddSample(sample.timestamp)
		if err != nil {
			t.Errorf("Sample %d (timestamp %d) should not error: %v", i, sample.timestamp, err)
		}
		if logical != sample.expectedLogical {
			t.Errorf("Sample %d: expected logical time %d, got %d", i, sample.expectedLogical, logical)
		}
	}

	// Test that we can retrieve the exact timestamps back
	for _, sample := range samples {
		retrievedTs, exists := tsa.GetTimestamp(sample.expectedLogical)
		if !exists {
			t.Errorf("Should be able to retrieve timestamp for logical time %d", sample.expectedLogical)
		}
		if retrievedTs != sample.timestamp {
			t.Errorf("Logical time %d: expected timestamp %d, got %d", sample.expectedLogical, sample.timestamp, retrievedTs)
		}
	}

	// Test boundary conditions - timestamps that fall between logical intervals
	boundaryTests := []struct {
		timestamp       int64
		expectedLogical int
		description     string
	}{
		{11510, 0, "11510 should map to logical 0 (between 0 and 1)"},          // More than 1s past [0] but before [1]
		{12450, 2, "12450 should map to logical 2 (between 2 and 3)"},          // Between [2] and [3]
		{11000, 0, "11000 should map to logical 0 (exactly 0.5s after first)"}, // Exactly 0.5s after first
		{12000, 1, "12000 should map to logical 1 (between 1 and 2)"},          // Between [1] and [2]
		{13000, 2, "13000 should map to logical 2 (between 2 and 3)"},          // Between [2] and [3]
	}

	for _, test := range boundaryTests {
		logical := tsa.GetLogicalTime(test.timestamp)
		if logical != test.expectedLogical {
			t.Errorf("%s: expected logical time %d, got %d", test.description, test.expectedLogical, logical)
		}
	}
}

func TestTimeSampleAlignerGapFilling(t *testing.T) {
	tsa := MakeTimeSampleAligner(50)

	// Add samples with a large gap
	samples := []struct {
		timestamp       int64
		expectedLogical int
	}{
		{10500, 0}, // First sample
		{11400, 1}, // 0.9 seconds later
		{18250, 8}, // 6.85 seconds later (should fill gap and map to logical 8)
	}

	// Add the samples
	for i, sample := range samples {
		logical, err := tsa.AddSample(sample.timestamp)
		if err != nil {
			t.Errorf("Sample %d (timestamp %d) should not error: %v", i, sample.timestamp, err)
		}
		if logical != sample.expectedLogical {
			t.Errorf("Sample %d: expected logical time %d, got %d", i, sample.expectedLogical, logical)
		}
	}

	// Verify that synthetic timestamps were created to fill the gap
	// The gap should be filled with: 12400, 13400, 14400, 15400, 16400, 17400
	expectedSyntheticTimestamps := []struct {
		logical    int
		expectedTs int64
	}{
		{2, 12400}, // 11400 + 1000
		{3, 13400}, // 11400 + 2000
		{4, 14400}, // 11400 + 3000
		{5, 15400}, // 11400 + 4000
		{6, 16400}, // 11400 + 5000
		{7, 17400}, // 11400 + 6000
	}

	for _, expected := range expectedSyntheticTimestamps {
		ts, exists := tsa.GetTimestamp(expected.logical)
		if !exists {
			t.Errorf("Should have synthetic timestamp for logical time %d", expected.logical)
		}
		if ts != expected.expectedTs {
			t.Errorf("Logical time %d: expected synthetic timestamp %d, got %d", expected.logical, expected.expectedTs, ts)
		}
	}

	// Test that timestamps in the gap map to correct logical times
	// Intervals work as [start, next) - so logical N covers from timestamp N to timestamp N+1 (exclusive)
	gapTests := []struct {
		timestamp       int64
		expectedLogical int
		description     string
	}{
		{13450, 3, "13450 should map to logical 3 (in interval [13400, 14400))"},
		{14350, 3, "14350 should map to logical 3 (in interval [13400, 14400))"},
		{14400, 4, "14400 should map to logical 4 (at start of interval [14400, 15400))"},
		{16500, 6, "16500 should map to logical 6 (in interval [16400, 17400))"},
		{17500, 7, "17500 should map to logical 7 (in interval [17400, 18250))"},
	}

	for _, test := range gapTests {
		logical := tsa.GetLogicalTime(test.timestamp)
		if logical != test.expectedLogical {
			t.Errorf("%s: expected logical time %d, got %d", test.description, test.expectedLogical, logical)
		}
	}
}

