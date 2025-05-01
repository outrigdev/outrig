// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package updatecheck

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
)

const (
	// GitHubReleasesURL is the URL to check for the latest release
	GitHubReleasesURL = "https://api.github.com/repos/outrigdev/outrig/releases/latest"

	// InitialDelay is the delay before the first update check (10 seconds)
	InitialDelay = 10 * time.Second

	// CheckInterval is the interval between update checks (5 minutes)
	CheckInterval = 5 * time.Minute

	// UpdateCheckPeriod is how often we actually perform the check (24 hours)
	UpdateCheckPeriod = 24 * time.Hour
)

var (
	// Disabled is a flag to disable update checking
	Disabled atomic.Bool

	// newerVersion stores the newer version if one is found
	newerVersion string

	// mutex protects access to newerVersion
	mutex sync.RWMutex

	// lastCheckTime is the time of the last update check
	lastCheckTime int64
)

// GitHubRelease represents the GitHub release API response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// StartUpdateChecker starts the update checker routine
func StartUpdateChecker() {
	// Don't start the update checker if it's disabled
	if Disabled.Load() {
		log.Printf("Update checker is disabled, not starting")
		return
	}

	// Set the initial last check time
	atomic.StoreInt64(&lastCheckTime, time.Now().UnixMilli())

	// Start the update checker goroutine
	go func() {
		outrig.SetGoRoutineName("UpdateChecker")
		// Wait for the initial delay before the first check
		time.Sleep(InitialDelay)

		// Perform the first check
		checkForUpdates()

		// Start the ticker for periodic checks
		ticker := time.NewTicker(CheckInterval)
		defer ticker.Stop()

		for range ticker.C {
			// Check if it's time to perform an update check
			now := time.Now().UnixMilli()
			lastCheck := atomic.LoadInt64(&lastCheckTime)

			if now-lastCheck >= UpdateCheckPeriod.Milliseconds() {
				checkForUpdates()
				atomic.StoreInt64(&lastCheckTime, now)
			}
		}
	}()
}

// checkForUpdates checks if there's a newer version available
func checkForUpdates() {
	if Disabled.Load() {
		return
	}

	log.Printf("Checking for Outrig updates...")

	// Get the current version
	currentVersion := serverbase.OutrigServerVersion

	// Get the latest release from GitHub
	latestVersion, err := getLatestRelease()
	if err != nil {
		log.Printf("Error checking for updates: %v", err)
		// Track the error
		outrig.TrackValue("updatecheck.latestreleasecheck", fmt.Sprintf("error: %v", err))
		return
	}

	// Track the successful result
	outrig.TrackValue("updatecheck.latestreleasecheck", latestVersion)

	// Compare versions
	newer, err := isNewerVersion(currentVersion, latestVersion)
	if err != nil {
		log.Printf("Error comparing versions: %v", err)
		return
	}

	if newer {
		log.Printf("New version available: %s (current: %s)", latestVersion, currentVersion)
		setNewerVersion(latestVersion)
	} else {
		log.Printf("No new version available (current: %s, latest: %s)", currentVersion, latestVersion)
		setNewerVersion("")
	}
}

// getLatestRelease gets the latest release from GitHub
func getLatestRelease() (string, error) {
	// Create a new HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create a new request
	req, err := http.NewRequest("GET", GitHubReleasesURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set the User-Agent header
	req.Header.Set("User-Agent", "Outrig-UpdateChecker")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Parse the JSON response
	var release GitHubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		return "", fmt.Errorf("error parsing JSON response: %w", err)
	}

	// Return the tag name
	return release.TagName, nil
}

// isNewerVersion checks if the latest version is newer than the current version
func isNewerVersion(currentVersion, latestVersion string) (bool, error) {
	// Parse the versions
	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return false, fmt.Errorf("error parsing current version: %w", err)
	}

	latest, err := semver.NewVersion(latestVersion)
	if err != nil {
		return false, fmt.Errorf("error parsing latest version: %w", err)
	}

	// Compare the versions
	return latest.GreaterThan(current), nil
}

// setNewerVersion sets the newer version with proper locking
func setNewerVersion(version string) {
	mutex.Lock()
	defer mutex.Unlock()
	newerVersion = version
}

// GetUpdatedVersion returns the newer version if one is available
func GetUpdatedVersion() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return newerVersion
}
