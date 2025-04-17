// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tevent

import (
	"sync"
	"sync/atomic"

	"github.com/outrigdev/outrig"
)

const (
	// Hard maximum buffer size - events will be dropped if this is exceeded
	hardMaxBufferSize = 1000
)

var (
	eventBuffer     []TEvent
	eventBufferLock sync.Mutex
	writerOnce      sync.Once
	eventsWritten   atomic.Int64
	eventsInBuffer  atomic.Int64
)

func initEventBuffer() {
	if Disabled.Load() {
		return
	}

	// Initialize the buffer
	eventBufferLock.Lock()
	eventBuffer = make([]TEvent, 0)
	eventBufferLock.Unlock()

	// Register counters with Outrig
	outrig.WatchAtomicCounter("tevent:eventsWritten", &eventsWritten)
	outrig.WatchAtomicCounter("tevent:eventsInBuffer", &eventsInBuffer)
}

// GrabEvents takes the lock, gets up to maxSize events from the buffer, and returns them
// If maxSize <= 0, it returns all events
func GrabEvents(maxSize int) []TEvent {
	eventBufferLock.Lock()
	defer eventBufferLock.Unlock()

	if len(eventBuffer) == 0 {
		return nil
	}

	var events []TEvent
	if maxSize <= 0 || maxSize >= len(eventBuffer) {
		// Take all events
		events = eventBuffer
		eventsInBuffer.Store(0)
		eventBuffer = make([]TEvent, 0)
	} else {
		// Take only maxSize events
		events = eventBuffer[:maxSize]
		eventBuffer = eventBuffer[maxSize:]
		eventsInBuffer.Store(int64(len(eventBuffer)))
	}

	return events
}

// GetEventBufferLength returns the current number of events in the buffer
// with proper locking
func GetEventBufferLength() int {
	eventBufferLock.Lock()
	defer eventBufferLock.Unlock()
	return len(eventBuffer)
}

// WriteTEvent adds a telemetry event to the in-memory buffer
// If the buffer exceeds hardMaxBufferSize, the event will be dropped
func WriteTEvent(event TEvent) {
	if Disabled.Load() {
		return
	}

	// Initialize the buffer if not already done
	writerOnce.Do(initEventBuffer)

	// Ensure timestamps are set
	event.EnsureTimestamps()

	// Lock the buffer and ensure it's unlocked when we're done
	eventBufferLock.Lock()
	defer eventBufferLock.Unlock()

	currentSize := len(eventBuffer)
	if currentSize >= hardMaxBufferSize {
		return
	}
	eventBuffer = append(eventBuffer, event)
	currentSize++
	eventsWritten.Add(1)
	eventsInBuffer.Add(1)
}
