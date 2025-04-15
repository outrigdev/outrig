// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/ds"
)

// getDefaultConfig returns a default configuration with the specified dev mode
func getDefaultConfig(isDev bool) *ds.Config {
	wrapStdout := true
	wrapStderr := true

	if os.Getenv(base.ExternalLogCaptureEnvName) != "" {
		wrapStdout = false
		wrapStderr = false
	}

	return &ds.Config{
		DomainSocketPath: base.GetDomainSocketNameForClient(isDev),
		AppName:          "",
		ModuleName:       "",
		Dev:              isDev,
		ConnectOnInit:    true,
		LogProcessorConfig: ds.LogProcessorConfig{
			Enabled:    true,
			WrapStdout: wrapStdout,
			WrapStderr: wrapStderr,
		},
		WatchConfig: ds.WatchConfig{
			Enabled: true,
		},
		GoRoutineConfig: ds.GoRoutineConfig{
			Enabled: true,
		},
		RuntimeStatsConfig: ds.RuntimeStatsConfig{
			Enabled: true,
		},
	}
}

// DefaultConfig returns the default configuration for normal usage
func DefaultConfig() *ds.Config {
	return getDefaultConfig(false)
}

// DefaultConfigForOutrigDevelopment returns a configuration specifically for Outrig internal development
// This is only used for internal Outrig development and not intended for general SDK users
func DefaultConfigForOutrigDevelopment() *ds.Config {
	return getDefaultConfig(true)
}