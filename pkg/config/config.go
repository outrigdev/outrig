// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"log"
	"os"

	"github.com/outrigdev/outrig/pkg/base"
)

type Config struct {
	Quiet bool // If true, suppresses init, connect, and disconnect messages

	// DomainSocketPath is the path to the Unix domain socket. If "" => use default.
	// If "-" => disable domain socket.
	DomainSocketPath string

	// TcpAddr is the TCP address to connect to the Outrig server.  If "" => use default.
	// If "-" => disable TCP connection. Domain socket will be tried first (except on Windows where domain sockets are not supported).)
	TcpAddr string

	// By default the SDK will probe host.docker.internal:5005 to see if the Outrig monitor is running on the host machine
	// We do an initial DNS lookup at startup and only try this host/port if the DNS lookup succeeds.
	// Setting this to true will disable the initial probe.
	DisableDockerProbe bool

	// ModuleName is the name of the Go module. If not specified, it will be determined
	// from the go.mod file.
	ModuleName string

	// Dev indicates whether the client is in development mode
	Dev bool

	// If true, try to synchronously connect to the server on Init
	ConnectOnInit bool

	// Collector configurations
	LogProcessorConfig LogProcessorConfig
	WatchConfig        WatchConfig
	GoRoutineConfig    GoRoutineConfig
	RuntimeStatsConfig RuntimeStatsConfig
}

type LogProcessorConfig struct {
	// Enabled indicates whether the log processor is enabled
	Enabled    bool
	WrapStdout bool
	WrapStderr bool
	// OutrigPath is the full path to the outrig executable (including the executable name)
	// If empty, the system will look for "outrig" in the PATH
	OutrigPath string
	// AdditionalArgs are additional arguments to pass to the outrig command
	// These are inserted before the "capturelogs" argument
	AdditionalArgs []string
}

type WatchConfig struct {
	// Enabled indicates whether the watch collector is enabled
	Enabled bool
}

type GoRoutineConfig struct {
	// Enabled indicates whether the goroutine collector is enabled
	Enabled bool
}

type RuntimeStatsConfig struct {
	// Enabled indicates whether the runtime stats collector is enabled
	Enabled bool
}

// getDefaultConfig returns a default configuration with the specified dev mode
func getDefaultConfig(isDev bool) *Config {
	wrapStdout := true
	wrapStderr := true

	if os.Getenv(base.ExternalLogCaptureEnvName) != "" {
		wrapStdout = false
		wrapStderr = false
	}

	return &Config{
		DomainSocketPath: base.GetDomainSocketNameForClient(isDev),
		TcpAddr:          base.GetTcpAddrForClient(isDev),
		ModuleName:       "",
		Dev:              isDev,
		ConnectOnInit:    true,
		LogProcessorConfig: LogProcessorConfig{
			Enabled:    true,
			WrapStdout: wrapStdout,
			WrapStderr: wrapStderr,
		},
		WatchConfig: WatchConfig{
			Enabled: true,
		},
		GoRoutineConfig: GoRoutineConfig{
			Enabled: true,
		},
		RuntimeStatsConfig: RuntimeStatsConfig{
			Enabled: true,
		},
	}
}

// DefaultConfig returns the default configuration for normal usage
func DefaultConfig() *Config {
	if os.Getenv("OUTRIG_DEVCONFIG") != "" {
		log.Printf("OUTRIG_DEVCONFIG environment variable is set, switching to development configuration")
		return getDefaultConfig(true)
	}
	return getDefaultConfig(false)
}

// DefaultConfigForOutrigDevelopment returns a configuration specifically for Outrig internal development
// This is only used for internal Outrig development and not intended for general SDK users
func DefaultConfigForOutrigDevelopment() *Config {
	return getDefaultConfig(true)
}
