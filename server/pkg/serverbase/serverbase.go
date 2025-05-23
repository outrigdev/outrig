// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package serverbase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// OutrigServerVersion is the current version of Outrig
// This gets set from main-server.go during initialization
var OutrigServerVersion = "v0.5.11"

// OutrigBuildTime is the build timestamp of Outrig
// This gets set from main-server.go during initialization
var OutrigBuildTime = ""

// OutrigCommit is the git commit hash of this build
// This gets set from main-server.go during initialization
var OutrigCommit = ""

// OutrigId is the unique identifier for this Outrig server instance
var OutrigId string

// OutrigFirstRun indicates if this is the first run of this Outrig server instance
var OutrigFirstRun bool

const OutrigLockFile = "outrig.lock"
const OutrigIdFile = "outrig.id"
const OutrigDataDir = "data"
const OutrigDevEnvName = "OUTRIG_DEV"
const OutrigTEventsFile = "tevents.jsonl"

// Default production port for server
const ProdWebServerPort = 5005

// Development port for server
const DevWebServerPort = 6005

type FDLock interface {
	Close() error
}

// IsDev returns true if the server is running in development mode
func IsDev() bool {
	return os.Getenv(OutrigDevEnvName) == "1"
}

// GetOutrigHome returns the appropriate home directory based on mode
func GetOutrigHome() string {
	if IsDev() {
		return base.DevOutrigHome
	}
	return base.OutrigHome
}

// GetDomainSocketName returns the full domain socket path
func GetDomainSocketName() string {
	return GetOutrigHome() + base.DefaultDomainSocketName
}

// GetWebServerPort returns the appropriate web server port based on mode
func GetWebServerPort() int {
	if IsDev() {
		return DevWebServerPort
	}
	return ProdWebServerPort
}

// EnsureOutrigId ensures that the outrig.id file exists and contains a valid UUID.
// If the file doesn't exist, it creates it with a new UUID.
// If the file exists but contains an invalid UUID, it overwrites it with a new UUID.
// Returns:
// - The UUID (either read from the file or newly generated)
// - A boolean indicating if a new UUID was generated (true) or read from an existing file (false)
// - An error if one occurred during the process
func EnsureOutrigId() (string, bool, error) {
	// Get the path to the outrig.id file
	idFilePath := filepath.Join(utilfn.ExpandHomeDir(GetOutrigHome()), OutrigIdFile)

	// Try to read the existing file
	content, err := os.ReadFile(idFilePath)
	if err == nil {
		// File exists, check if it contains a valid UUID
		idStr := strings.TrimSpace(string(content))
		_, err := uuid.Parse(idStr)
		if err == nil {
			// Valid UUID found
			return idStr, false, nil
		}
		// Invalid UUID, will generate a new one
	}

	// Generate a new UUID (v7)
	newUuid, err := uuid.NewV7()
	if err != nil {
		return "", false, fmt.Errorf("failed to generate outrig ID: %w", err)
	}
	newId := newUuid.String()

	// Write the new UUID to the file
	err = os.WriteFile(idFilePath, []byte(newId), 0644)
	if err != nil {
		return "", false, fmt.Errorf("failed to write outrig.id file: %w", err)
	}

	return newId, true, nil
}

// GetOutrigDataDir returns the path to the data directory
func GetOutrigDataDir() string {
	return filepath.Join(GetOutrigHome(), OutrigDataDir)
}

func EnsureHomeDir() error {
	outrigHomeDir := utilfn.ExpandHomeDir(GetOutrigHome())
	return os.MkdirAll(outrigHomeDir, 0755)
}

func EnsureDataDir() error {
	dataDir := utilfn.ExpandHomeDir(GetOutrigDataDir())
	return os.MkdirAll(dataDir, 0755)
}

// GetTEventsFilePath returns the full path to the tevents.jsonl file
func GetTEventsFilePath() string {
	return filepath.Join(GetOutrigDataDir(), OutrigTEventsFile)
}
