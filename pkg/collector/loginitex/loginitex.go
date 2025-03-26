// Package loginitex provides external process-based log capture functionality
package loginitex

// EnableExternalLogWrap redirects stdout and stderr to an external outrig capturelogs process
// isDev specifies whether to run the process in development mode
func EnableExternalLogWrap(isDev bool) error {
	// Platform-specific implementation will be provided
	return enableExternalLogWrapImpl(isDev)
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
