//go:build windows

package loginitex

import (
	"errors"
)

func enableExternalLogWrapImpl(isDev bool) error {
	return errors.New("external log wrapping not supported on Windows")
}

func disableExternalLogWrapImpl() {
	// No-op on Windows
}

func isExternalLogWrapActiveImpl() bool {
	// External log wrapping is not supported on Windows
	return false
}
