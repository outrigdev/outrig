// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tevent

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig"
)

const (
	// Time between automatic flushes (1 hour)
	flushInterval = time.Hour

	// Time between ticker checks (5 minutes)
	tickInterval = 5 * time.Minute

	// Maximum number of events to buffer before forcing a flush
	maxBufferSize = 300
)

var (
	eventBuffer     []TEvent
	eventBufferLock sync.Mutex
	writerOnce      sync.Once
	eventsWritten   atomic.Int64
	eventsInBuffer  atomic.Int64
	lastFlushTime   int64
	ticker          *time.Ticker
)

func initEventBuffer() {
	if Disabled.Load() {
		return
	}

	// Initialize the buffer
	eventBufferLock.Lock()
	eventBuffer = make([]TEvent, 0, maxBufferSize)
	eventBufferLock.Unlock()

	// Register counters with Outrig
	outrig.WatchAtomicCounter("tevent:eventsWritten", &eventsWritten)
	outrig.WatchAtomicCounter("tevent:eventsInBuffer", &eventsInBuffer)

	// Set initial flush time
	atomic.StoreInt64(&lastFlushTime, time.Now().UnixMilli())

	// Start the ticker for periodic checks
	ticker = time.NewTicker(tickInterval)
	go func() {
		outrig.SetGoRoutineName("TEventTicker")
		for range ticker.C {
			checkAndFlush()
		}
	}()
}

// checkAndFlush checks if it's time to flush events based on time elapsed or buffer size
func checkAndFlush() {
	now := time.Now().UnixMilli()

	// Check if an hour has passed since the last flush
	if now-atomic.LoadInt64(&lastFlushTime) >= flushInterval.Milliseconds() {
		UploadEventsAsync()
	}
}

// GrabEvents takes the lock, gets the current events, clears the buffer, and returns the events
func GrabEvents() []TEvent {
	eventBufferLock.Lock()
	defer eventBufferLock.Unlock()
	if len(eventBuffer) == 0 {
		return nil
	}
	events := eventBuffer
	eventsInBuffer.Store(0)
	eventBuffer = make([]TEvent, 0, maxBufferSize)
	return events
}

// WriteTEvent adds a telemetry event to the in-memory buffer
func WriteTEvent(event TEvent) {
	if Disabled.Load() {
		return
	}

	// Initialize the buffer if not already done
	writerOnce.Do(initEventBuffer)

	// Ensure timestamps are set
	event.EnsureTimestamps()

	// Add to buffer with lock protection
	eventBufferLock.Lock()
	eventBuffer = append(eventBuffer, event)
	currentSize := len(eventBuffer)
	eventBufferLock.Unlock()

	// Increment counters
	eventsWritten.Add(1)
	eventsInBuffer.Add(1)

	// Check if we need to flush due to buffer size
	if currentSize >= maxBufferSize {
		UploadEventsAsync()
	}
}
