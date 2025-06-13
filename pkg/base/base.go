// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package base

import "strconv"

// Home directory paths
const OutrigHome = "~/.config/outrig"
const DevOutrigHome = "~/.config/outrig-dev"

// Domain socket name (just the filename part)
const DefaultDomainSocketName = "/outrig.sock"

// Environment variables
const ExternalLogCaptureEnvName = "OUTRIG_EXTERNALLOGCAPTURE"
const AppRunIdEnvName = "OUTRIG_APPRUNID"

const OutrigSDKVersion = "v0.7.4"

// Default ports for the server (should match serverbase)
const (
	ProdWebServerPort = 5005
	DevWebServerPort  = 6005
)

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

func GetTcpAddrForClient(isDev bool) string {
	return "127.0.0.1:" + strconv.Itoa(GetMonitorPort(isDev))
}

func GetMonitorPort(isDev bool) int {
	if isDev {
		return DevWebServerPort
	}
	return ProdWebServerPort
}
