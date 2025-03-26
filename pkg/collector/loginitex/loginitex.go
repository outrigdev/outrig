// Package loginitex provides external process-based log capture functionality
package loginitex

import (
	"github.com/outrigdev/outrig/pkg/ds"
)

// EnableExternalLogWrap redirects stdout and stderr to an external outrig capturelogs process
// isDev specifies whether to run the process in development mode
// config specifies which streams to wrap (stdout/stderr)
func EnableExternalLogWrap(isDev bool, config ds.LogProcessorConfig) error {
	// Platform-specific implementation will be provided
	return enableExternalLogWrapImpl(isDev, config)
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
