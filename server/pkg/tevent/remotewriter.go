// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tevent

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig"
)

// UploadEvents grabs events from the buffer and uploads them to the server
// It also updates the lastFlushTime at the beginning of the upload
func UploadEvents() error {
	if Disabled.Load() {
		return nil
	}
	// Update the last flush time at the beginning of the upload
	now := time.Now()
	atomic.StoreInt64(&lastFlushTime, now.UnixMilli())

	// Grab events from the buffer
	events := GrabEvents()

	if len(events) == 0 {
		// Update status even if no events were uploaded
		updateTelemetryStatus(now, 0)
		return nil
	}

	// TODO upload to the server
	log.Printf("(pretending to upload telemetry events, no upload implemented yet)")
	time.Sleep(500 * time.Millisecond)

	// Update telemetry status with upload information
	updateTelemetryStatus(now, len(events))

	log.Printf("Uploaded %d telemetry events", len(events))

	return nil
}

// UploadEventsAsync calls UploadEvents in a separate goroutine
func UploadEventsAsync() {
	go func() {
		outrig.SetGoRoutineName("TEventUploader")
		err := UploadEvents()
		if err != nil {
			log.Printf("Failed to upload telemetry: %v", err)
		}
	}()
}

// updateTelemetryStatus updates the telemetry status with upload information
func updateTelemetryStatus(uploadTime time.Time, eventCount int) {
	// Format the last upload time
	lastUploadStr := uploadTime.Format(time.RFC3339)

	// Calculate the next upload time (1 hour from now)
	nextUploadTime := uploadTime.Add(flushInterval)
	nextUploadStr := nextUploadTime.Format(time.RFC3339)

	// Format the status string
	var status string
	if eventCount > 0 {
		status = fmt.Sprintf("Last Telemetry Uploaded at %s (%d events)\nNext Telemetry Upload at %s",
			lastUploadStr, eventCount, nextUploadStr)
	} else {
		status = fmt.Sprintf("Last Telemetry Check at %s (no events)\nNext Telemetry Upload at %s",
			lastUploadStr, nextUploadStr)
	}

	// Track the status value
	outrig.TrackValue("tevent:status", status)
}
