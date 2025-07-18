// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const ConfigFileName = "outrig.json"

func LoadConfig() (*Config, error) {
	// 1. Check explicit JSON env var first
	if configJson := os.Getenv(ConfigJsonEnvName); configJson != "" {
		var cfg Config
		if err := json.Unmarshal([]byte(configJson), &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}
	
	// 2. Check explicit config file env var
	if configFile := os.Getenv(ConfigFileEnvName); configFile != "" {
		cfg, err := tryLoadConfig(configFile)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			return cfg, nil
		}
		// If explicitly set but file doesn't exist, that's an error
		return nil, os.ErrNotExist
	}
	
	// 3. Walk up directories looking for project root (includes current dir)
	cfg, err := findConfigInParents()
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		return cfg, nil
	}
	
	// 4. No config found (not an error)
	return nil, nil
}

func findConfigInParents() (*Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	
	homeDir, _ := os.UserHomeDir()
	
	for {
		// Check for config file in current dir
		path := filepath.Join(dir, ConfigFileName)
		cfg, err := tryLoadConfig(path)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			return cfg, nil
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
	
	return nil, nil
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
		return nil, err
	}
	
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err // JSON parse error is always an error
	}
	
	return &cfg, nil
}
