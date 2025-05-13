// Package logutil provides logging utilities for the Outrig server.
package logutil

import (
	"log"
	"sync"
)

var (
	// loggedKeys tracks which keys have already been logged
	loggedKeys = make(map[string]struct{})
	// mutex protects access to the loggedKeys map
	mutex sync.Mutex
)

// shouldLog checks if a message with the given key should be logged
// and marks the key as logged if it hasn't been seen before.
// Returns true if the message should be logged, false otherwise.
func shouldLog(key string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	if _, exists := loggedKeys[key]; exists {
		return false
	}
	loggedKeys[key] = struct{}{}
	return true
}

// LogOnce logs a message with the given key only once.
// If a message with the same key has already been logged, this function does nothing.
// The format and args parameters work the same as log.Printf
func LogfOnce(key string, format string, args ...interface{}) {
	// Only log if this key hasn't been seen before
	if !shouldLog(key) {
		return
	}
	log.Printf(format, args...)
}
