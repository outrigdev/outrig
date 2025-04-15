// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tevent

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig"
)

var (
	eventsUploaded atomic.Int64
	uploadAttempts atomic.Int64
)

func init() {
	// Register counters with Outrig
	outrig.WatchAtomicCounter("tevent:eventsUploaded", &eventsUploaded)
	outrig.WatchAtomicCounter("tevent:uploadAttempts", &uploadAttempts)
}

// UploadEvents grabs events from the buffer and uploads them to the server
// It also updates the lastFlushTime at the beginning of the upload
func UploadEvents() error {
	if Disabled.Load() {
		return nil
	}

	// Increment upload attempts counter
	uploadAttempts.Add(1)

	// Update the last flush time at the beginning of the upload
	now := time.Now()
	atomic.StoreInt64(&lastFlushTime, now.UnixMilli())

	// Grab events from the buffer
	events := GrabEvents()

	if len(events) == 0 {
		// Update status even if no events were uploaded
		updateTelemetryStatus(now, 0, nil)
		return nil
	}

	// TODO upload to the server
	time.Sleep(500 * time.Millisecond)

	// Increment uploaded events counter
	eventsUploaded.Add(int64(len(events)))

	// Update telemetry status with upload information
	updateTelemetryStatus(now, len(events), nil)

	return nil
}

// UploadEventsAsync calls UploadEvents in a separate goroutine
func UploadEventsAsync() {
	go func() {
		outrig.SetGoRoutineName("TEventUploader")
		_ = UploadEvents() // ignore error, written to status
	}()
}

// updateTelemetryStatus updates the telemetry status with upload information
func updateTelemetryStatus(uploadTime time.Time, eventCount int, err error) {
	// Format the last upload time
	lastUploadStr := uploadTime.Format(time.RFC3339)

	// Calculate the next upload time (1 hour from now)
	nextUploadTime := uploadTime.Add(flushInterval)
	nextUploadStr := nextUploadTime.Format(time.RFC3339)

	// Format the status string
	var status string
	if err != nil {
		status = fmt.Sprintf("Telemetry Upload Error at %s: %v\nNext Telemetry Upload at %s",
			lastUploadStr, err, nextUploadStr)
	} else if eventCount > 0 {
		status = fmt.Sprintf("Last Telemetry Uploaded at %s (%d events)\nNext Telemetry Upload at %s",
			lastUploadStr, eventCount, nextUploadStr)
	} else {
		status = fmt.Sprintf("Last Telemetry Check at %s (no events)\nNext Telemetry Upload at %s",
			lastUploadStr, nextUploadStr)
	}

	// Track the status value
	outrig.TrackValue("tevent:status", status)
}
