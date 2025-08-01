// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFileName = "outrig.json"

// LoadConfig loads configuration from various sources in priority order.
// The overrideFileName parameter, if provided, takes highest priority and overrides all other sources.
// This is typically used when a config file is explicitly specified via CLI arguments.
// The cwd parameter specifies the working directory to use for config discovery. If empty, uses os.Getwd().
//
// Configuration loading priority (highest to lowest):
//  1. overrideFileName parameter (if not empty) - returns error if file doesn't exist
//  2. OUTRIG_CONFIGJSON environment variable - JSON string
//  3. OUTRIG_CONFIGFILE environment variable - file path
//  4. outrig.json files found by walking up directory tree from specified working directory,
//     stopping at project root markers (go.mod, .git) or home directory
//
// Returns nil config (not an error) if no configuration is found through automatic discovery.
// Returns an error if an explicitly specified config source fails to load or parse.
func LoadConfig(overrideFileName string, cwd string) (*Config, string, error) {
	// 1. Check explicit filename parameter first (overrides everything)
	if overrideFileName != "" {
		cfg, err := tryLoadConfig(overrideFileName)
		if err != nil {
			return nil, "", err
		}
		if cfg != nil {
			return cfg, fmt.Sprintf("file:%q", overrideFileName), nil
		}
		// If explicitly set but file doesn't exist, that's an error
		return nil, "", fmt.Errorf("config file does not exist: %s", overrideFileName)
	}

	// 2. Check explicit JSON env var
	if configJson := os.Getenv(ConfigJsonEnvName); configJson != "" {
		var cfg Config
		if err := json.Unmarshal([]byte(configJson), &cfg); err != nil {
			return nil, "", fmt.Errorf("failed to parse JSON from %s env var: %w", ConfigJsonEnvName, err)
		}
		return &cfg, fmt.Sprintf("env:%s", ConfigJsonEnvName), nil
	}

	// 3. Check explicit config file env var
	if configFile := os.Getenv(ConfigFileEnvName); configFile != "" {
		cfg, err := tryLoadConfig(configFile)
		if err != nil {
			return nil, "", err
		}
		if cfg != nil {
			return cfg, fmt.Sprintf("file:%q", configFile), nil
		}
		// If explicitly set but file doesn't exist, that's an error
		return nil, "", fmt.Errorf("config file does not exist (from %s env var): %s", ConfigFileEnvName, configFile)
	}

	// 4. Walk up directories looking for project root (includes current dir)
	cfg, configPath, err := findConfigInParents(cwd)
	if err != nil {
		return nil, "", err
	}
	if cfg != nil {
		return cfg, fmt.Sprintf("file:%q", configPath), nil
	}

	// 5. No config found (not an error)
	return nil, "", nil
}

// findConfigInParents searches for a config file by walking up the directory tree.
// Returns the config, the path where it was found, and any error.
func findConfigInParents(cwd string) (*Config, string, error) {
	var dir string
	var err error

	if cwd != "" {
		dir = cwd
	} else {
		dir, err = os.Getwd()
		if err != nil {
			return nil, "", err
		}
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, "", err
	}

	homeDir, _ := os.UserHomeDir()

	for {
		// Check for config file in current dir
		path := filepath.Join(dir, ConfigFileName)
		cfg, err := tryLoadConfig(path)
		if err != nil {
			return nil, "", err
		}
		if cfg != nil {
			return cfg, path, nil
		}

		// Stop at project root markers
		if hasProjectRoot(dir) {
			break
		}

		// Stop at home directory
		if homeDir != "" && dir == homeDir {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir || parent == "/" { // reached filesystem root or about to traverse to it
			break
		}

		dir = parent
	}

	return nil, "", nil
}

func hasProjectRoot(dir string) bool {
	markers := []string{".git", "go.mod"}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

func tryLoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File not found is not an error for search
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON in config file %s: %w", path, err)
	}

	return &cfg, nil
}
