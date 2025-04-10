// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Package loginitex provides external process-based log capture functionality
package loginitex

import (
	"github.com/outrigdev/outrig/pkg/ds"
)

// EnableExternalLogWrap redirects stdout and stderr to an external outrig capturelogs process
// appRunId is the unique identifier for the application run
// config specifies which streams to wrap (stdout/stderr)
// isDev specifies whether to run the process in development mode
func EnableExternalLogWrap(appRunId string, config ds.LogProcessorConfig, isDev bool) error {
	// Platform-specific implementation will be provided
	return enableExternalLogWrapImpl(appRunId, config, isDev)
}

// DisableExternalLogWrap stops the external log capture process and restores original file descriptors
func DisableExternalLogWrap() {
	// Platform-specific implementation will be provided
	disableExternalLogWrapImpl()
}

// IsExternalLogWrapActive returns whether external log wrapping is currently active
func IsExternalLogWrapActive() bool {
	return isExternalLogWrapActiveImpl()
}
