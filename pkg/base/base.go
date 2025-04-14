// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package base

// Connection mode constants
const (
	ConnectionModePacket = "packet"
	ConnectionModeLog    = "log"
)

// Home directory paths
const OutrigHome = "~/.outrig"
const DevOutrigHome = "~/.outrig-dev"

// Domain socket name (just the filename part)
const DefaultDomainSocketName = "/outrig.sock"

// Environment variables
const ExternalLogCaptureEnvName = "OUTRIG_EXTERNALLOGCAPTURE"
const AppRunIdEnvName = "OUTRIG_APPRUNID"

// Client-specific functions that use the client's Dev flag

// GetOutrigHomeForClient returns the appropriate home directory based on client config
func GetOutrigHomeForClient(isDev bool) string {
	if isDev {
		return DevOutrigHome
	}
	return OutrigHome
}

// GetDomainSocketNameForClient returns the full domain socket path for client
func GetDomainSocketNameForClient(isDev bool) string {
	return GetOutrigHomeForClient(isDev) + DefaultDomainSocketName
}
