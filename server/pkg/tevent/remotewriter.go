// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tevent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
)

const TEventsUrl = "/tevents"

var (
	eventsUploaded atomic.Int64
	uploadAttempts atomic.Int64
	lastErrorTime  atomic.Int64
)

func init() {
	// Register counters with Outrig
	outrig.WatchAtomicCounter("tevent:eventsUploaded", &eventsUploaded)
	outrig.WatchAtomicCounter("tevent:uploadAttempts", &uploadAttempts)
}

const TEventsBatchSize = 200
const TEventsMaxBatches = 10
const CloudDefaultTimeout = 5 * time.Second
const ErrorCooldownPeriod = time.Hour // Wait 1 hour after an error before trying again

type TEventsInputType struct {
	ClientId string   `json:"clientid"`
	Events   []TEvent `json:"events"`
}

func makeOutrigApiPostReq(ctx context.Context, url string, input TEventsInputType) (*http.Request, error) {
	// Determine the base URL based on dev mode
	baseURL := "https://api.outrig.run"
	if serverbase.IsDev() {
		baseURL = "https://api-dev.outrig.run"
	}

	// Combine the base URL with the provided path
	fullURL := baseURL + url

	// Marshal the input to JSON
	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TEventsInputType: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Outrig/"+serverbase.OutrigServerVersion)
	req.Header.Set("X-OutrigAPIVersion", "1")

	return req, nil
}

func doRequest(req *http.Request, resp interface{}) (*http.Response, error) {
	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: CloudDefaultTimeout,
	}

	// Execute the request
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Always read the body to completion for connection reuse
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return httpResp, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-2xx status codes
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return httpResp, fmt.Errorf("HTTP request returned non-success status: %d %s - %s",
			httpResp.StatusCode, httpResp.Status, string(body))
	}

	// If a response struct was provided, unmarshal the response body into it
	if resp != nil && len(body) > 0 {
		err = json.Unmarshal(body, resp)
		if err != nil {
			return httpResp, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return httpResp, nil
}

// returns (done, num-sent, error)
func sendTEventsBatch(clientId string, events []TEvent) (bool, int, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), CloudDefaultTimeout)
	defer cancelFn()
	input := TEventsInputType{
		ClientId: clientId,
		Events:   events,
	}
	req, err := makeOutrigApiPostReq(ctx, TEventsUrl, input)
	if err != nil {
		return true, 0, err
	}
	_, err = doRequest(req, nil)
	if err != nil {
		return true, 0, err
	}
	return len(events) < TEventsBatchSize, len(events), nil
}

func sendTEvents(clientId string) (int, error) {
	numIters := 0
	totalEvents := 0

	for numIters < TEventsMaxBatches {
		numIters++

		// Grab a batch of events
		batch := GrabEvents(TEventsBatchSize)
		if len(batch) == 0 {
			break
		}

		done, numEvents, err := sendTEventsBatch(clientId, batch)
		if err != nil {
			// Set the last error time
			lastErrorTime.Store(time.Now().UnixMilli())
			// Error occurred, restore events to the buffer
			for _, event := range batch {
				WriteTEvent(event)
			}
			return totalEvents, err
		}

		totalEvents += numEvents

		if done {
			break
		}

		if numIters >= TEventsMaxBatches {
			break
		}
	}

	return totalEvents, nil
}

// inErrorCooldown checks if we're in the cooldown period after an error
func inErrorCooldown() (bool, time.Time) {
	lastError := lastErrorTime.Load()
	if lastError > 0 {
		cooldownEndTime := time.UnixMilli(lastError).Add(ErrorCooldownPeriod)
		if time.Now().Before(cooldownEndTime) {
			return true, cooldownEndTime
		}
	}
	return false, time.Time{}
}

// UploadEvents uploads events to the server
// UploadEvents uploads events to the server
func UploadEvents() error {
	if Disabled.Load() {
		return nil
	}
	if inCooldown, _ := inErrorCooldown(); inCooldown {
		return nil
	}
	uploadAttempts.Add(1)
	now := time.Now()
	eventsSent, err := sendTEvents(serverbase.OutrigId)
	if err != nil {
		updateTelemetryStatus(now, eventsSent, err)
		return err
	}
	if eventsSent == 0 {
		updateTelemetryStatus(now, 0, nil)
		return nil
	}
	eventsUploaded.Add(int64(eventsSent))
	updateTelemetryStatus(now, eventsSent, nil)
	return nil
}

// UploadEventsAsync calls UploadEvents in a separate goroutine
// It first checks if we're in the cooldown period to avoid spawning unnecessary goroutines
func UploadEventsAsync() {
	if inCooldown, _ := inErrorCooldown(); inCooldown {
		return
	}
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
	nextUploadTime := uploadTime.Add(time.Hour)
	nextUploadStr := nextUploadTime.Format(time.RFC3339)

	// Check if we're in error cooldown
	inCooldown, cooldownEndTime := inErrorCooldown()

	// Format the status string
	var status string
	if err != nil {
		if inCooldown {
			status = fmt.Sprintf("Telemetry Upload Error at %s: %v\nIn error cooldown until %s\nNext Telemetry Upload at %s",
				lastUploadStr, err, cooldownEndTime.Format(time.RFC3339), nextUploadStr)
		} else {
			status = fmt.Sprintf("Telemetry Upload Error at %s: %v\nNext Telemetry Upload at %s",
				lastUploadStr, err, nextUploadStr)
		}
	} else if inCooldown {
		uploadType := "Check"
		if eventCount > 0 {
			uploadType = "Uploaded"
		}
		status = fmt.Sprintf("Last Telemetry %s at %s (%d events)\nIn error cooldown until %s\nNext Telemetry Upload at %s",
			uploadType, lastUploadStr, eventCount, cooldownEndTime.Format(time.RFC3339), nextUploadStr)
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
