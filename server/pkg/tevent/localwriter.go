// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tevent

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig"
)

const (
	channelBufferSize = 100
)

var (
	eventFile     *os.File
	errorLogged   sync.Once
	eventChan     chan TEvent
	writerOnce    sync.Once
	eventsWritten atomic.Int64
)

func initEventFile() {
	if Disabled.Load() {
		return
	}

	if err := serverbase.EnsureDataDir(); err != nil {
		log.Printf("Failed to ensure tevent data directory: %v", err)
		return
	}

	filePath := utilfn.ExpandHomeDir(serverbase.GetTEventsFilePath())

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open local tevent file: %v", err)
		return
	}
	eventFile = file
	eventChan = make(chan TEvent, channelBufferSize)
	
	// Register the events written counter with Outrig
	outrig.WatchAtomicCounter("tevent:eventsWritten", &eventsWritten)
	
	// Start the writer goroutine
	go func() {
		outrig.SetGoRoutineName("TEventWriter")
		eventWriter()
	}()
}

func eventWriter() {
	for event := range eventChan {
		writeEventToFile(event)
	}
}

func writeEventToFile(event TEvent) {
	if eventFile == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		errorLogged.Do(func() {
			log.Printf("Failed to marshal tevent: %v", err)
		})
		return
	}

	data = append(data, '\n')

	_, err = eventFile.Write(data)
	if err != nil {
		errorLogged.Do(func() {
			log.Printf("Failed to write tevent to file: %v", err)
		})
		return
	}
	
	// Increment the events written counter
	eventsWritten.Add(1)
}

// WriteTEvent appends a telemetry event to the local JSONL file
func WriteTEvent(event TEvent) {
	if Disabled.Load() {
		return
	}

	// Initialize the file and writer goroutine if not already done
	writerOnce.Do(initEventFile)

	// Ensure timestamps are set
	event.EnsureTimestamps()

	// Try to send to channel, drop if full (non-blocking)
	select {
	case eventChan <- event:
		// Successfully sent to channel
	default:
		// Channel is full, drop the event
	}
}
