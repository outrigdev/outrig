// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig"
	configpkg "github.com/outrigdev/outrig/pkg/config"
)

// Command line flags
var crashFlag = flag.Bool("crash", false, "Cause an unhandled panic after a few seconds")

// Log levels for more diverse output
const (
	LogLevelInfo    = "INFO"
	LogLevelWarning = "WARNING"
	LogLevelError   = "ERROR"
	LogLevelDebug   = "DEBUG"
)

var LogCounter int
var LogCounterLock = &sync.Mutex{}

var LogCounterAtomic atomic.Int64

// Sample words for random log generation
var (
	subjects = []string{"server", "client", "database", "cache", "request", "response", "connection", "session", "user", "file", "process", "thread", "goroutine", "worker", "job"}
	verbs    = []string{"started", "stopped", "created", "deleted", "updated", "processed", "received", "sent", "loaded", "saved", "connected", "disconnected", "initialized", "terminated", "failed"}
	objects  = []string{"data", "request", "response", "file", "connection", "session", "transaction", "record", "document", "message", "event", "task", "operation", "resource", "service"}
	adverbs  = []string{"successfully", "quickly", "slowly", "unexpectedly", "partially", "completely", "temporarily", "permanently", "automatically", "manually", "correctly", "incorrectly", "efficiently", "poorly", "silently"}

	// Special patterns for testing search functionality
	patterns = []string{
		"ID: %s-%d",
		"User-%d logged in from %s",
		"Transaction %s completed in %dms",
		"Error code: E%d - %s",
		"API call to /api/%s returned status %d",
		"Memory usage: %d MB",
		"CPU load: %d%%",
		"[%s] %s",
		"key=%s value=%s",
		"{\"type\": \"%s\", \"id\": %d, \"status\": \"%s\"}",
	}

	// Sample IPs and endpoints for pattern substitution
	ips       = []string{"192.168.1.1", "10.0.0.5", "172.16.0.10", "127.0.0.1", "8.8.8.8"}
	endpoints = []string{"users", "products", "orders", "auth", "settings", "metrics", "health", "status", "config", "logs"}
	statuses  = []string{"success", "pending", "failed", "timeout", "rejected", "approved", "processing", "queued", "completed", "canceled"}
	errorMsgs = []string{"not found", "permission denied", "timeout", "invalid input", "connection lost", "out of memory", "internal error", "bad request", "unauthorized", "service unavailable"}
)

var counter int

// generateRandomWord returns a random word from the given slice
func randomElement(slice []string) string {
	counter++
	if counter%5000 == 10 {
		debug.PrintStack()
	}
	return slice[rand.Intn(len(slice))]
}

// generateRandomSentence creates a random log message
func generateRandomSentence() string {
	subject := randomElement(subjects)
	verb := randomElement(verbs)
	object := randomElement(objects)
	adverb := randomElement(adverbs)

	// Different sentence structures for variety
	switch rand.Intn(4) {
	case 0:
		return fmt.Sprintf("The %s %s the %s %s", subject, verb, object, adverb)
	case 1:
		return fmt.Sprintf("%s %s %s", subject, verb, adverb)
	case 2:
		return fmt.Sprintf("%s %s %s", subject, adverb, verb)
	default:
		return fmt.Sprintf("%s %s", subject, verb)
	}
}

// generatePatternLog creates a log message using one of the predefined patterns
func generatePatternLog() string {
	pattern := randomElement(patterns)

	switch pattern {
	case "ID: %s-%d":
		return fmt.Sprintf(pattern, randomElement(subjects), rand.Intn(10000))
	case "User-%d logged in from %s":
		return fmt.Sprintf(pattern, rand.Intn(1000), randomElement(ips))
	case "Transaction %s completed in %dms":
		return fmt.Sprintf(pattern, fmt.Sprintf("TX%d", rand.Intn(10000)), rand.Intn(500))
	case "Error code: E%d - %s":
		return fmt.Sprintf(pattern, rand.Intn(500), randomElement(errorMsgs))
	case "API call to /api/%s returned status %d":
		return fmt.Sprintf(pattern, randomElement(endpoints), []int{200, 201, 400, 401, 403, 404, 500}[rand.Intn(7)])
	case "Memory usage: %d MB":
		return fmt.Sprintf(pattern, rand.Intn(1024))
	case "CPU load: %d%%":
		return fmt.Sprintf(pattern, rand.Intn(100))
	case "[%s] %s":
		return fmt.Sprintf(pattern, randomElement(subjects), generateRandomSentence())
	case "key=%s value=%s":
		return fmt.Sprintf(pattern, randomElement(subjects), randomElement(objects))
	case "{\"type\": \"%s\", \"id\": %d, \"status\": \"%s\"}":
		return fmt.Sprintf(pattern, randomElement(subjects), rand.Intn(1000), randomElement(statuses))
	default:
		return pattern
	}
}

// generateStructuredLog creates a JSON log entry
func generateStructuredLog() string {
	data := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     randomElement([]string{LogLevelInfo, LogLevelWarning, LogLevelError, LogLevelDebug}),
		"component": randomElement(subjects),
		"action":    randomElement(verbs),
		"status":    randomElement(statuses),
		"duration":  rand.Intn(500),
		"metadata": map[string]interface{}{
			"id":      rand.Intn(10000),
			"source":  randomElement(ips),
			"success": rand.Intn(2) == 1,
		},
	}

	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func testForGoRoutine() {
	outrig.SetGoRoutineName("test-goroutine")
	time.Sleep(10 * time.Second)
}

func main() {
	// Parse command line flags
	flag.Parse()

	fmt.Printf("log before init\n")

	config := configpkg.DefaultConfigForOutrigDevelopment()
	config.LogProcessorConfig.OutrigPath = "go"
	config.LogProcessorConfig.AdditionalArgs = []string{"run", "server/main-server.go"}
	outrig.Init(config)
	defer outrig.AppDone()

	go testForGoRoutine()

	// Set up crash timer if flag is enabled
	if *crashFlag {
		fmt.Println("WARNING: --crash flag detected, will panic in 3 seconds")
		go func() {
			time.Sleep(1 * time.Second)
			panic("Intentional crash triggered by --crash flag")
		}()
	}
	outrig.WatchSync("test-count", LogCounterLock, &LogCounter)
	outrig.WatchAtomicCounter("test-count-atomic", &LogCounterAtomic)
	fmt.Printf("hello outrig!\n")
	time.Sleep(200 * time.Millisecond)
	outrig.Disable(false)
	fmt.Printf("during disable\n")
	time.Sleep(100 * time.Millisecond)
	outrig.Enable()
	fmt.Printf("after enable\n")
	fmt.Printf("again\n")

	outrig.TrackValue("push1 #tag1 #push", "hello world!")
	outrig.TrackValue("push2 #push", 55.23)

	// Output some initial structured logs for testing
	fmt.Printf("--- Starting log generation ---\n")
	fmt.Printf("This test program generates various log formats to help test search functionality\n")
	fmt.Printf("You can search for specific words, patterns, or JSON fields\n")
	fmt.Printf("Counter logs are still included with format: 'Counter: X'\n")

	// Loop that outputs diverse log lines
	counter := 0
	for {
		LogCounterLock.Lock()
		LogCounter = counter
		LogCounterLock.Unlock()
		LogCounterAtomic.Store(int64(counter))

		// Every 20th message is still the counter for continuity
		if counter%20 == 0 {
			fmt.Printf("Counter: %d\n", counter)
		} else {
			// Choose a log type randomly
			logType := rand.Intn(10)
			level := ""

			// Assign a log level
			switch rand.Intn(10) {
			case 0:
				level = LogLevelError + ": "
			case 1, 2:
				level = LogLevelWarning + ": "
			case 3, 4, 5:
				level = LogLevelDebug + ": "
			default:
				level = LogLevelInfo + ": "
			}

			// Generate different types of logs
			switch logType {
			case 0, 1, 2, 3: // 40% chance for random sentences
				fmt.Printf("%s%s\n", level, generateRandomSentence())
			case 4, 5, 6: // 30% chance for pattern logs
				fmt.Printf("%s%s\n", level, generatePatternLog())
			case 7: // 10% chance for structured JSON logs
				fmt.Printf("%s\n", generateStructuredLog())
			default: // 20% chance for combined logs
				fmt.Printf("%s%s | %s\n", level, generateRandomSentence(), generatePatternLog())
			}
		}

		counter++
		time.Sleep(5 * time.Millisecond)
	}
}
