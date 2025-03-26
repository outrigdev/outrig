//go:build windows

package loginitex

import (
	"errors"

	"github.com/outrigdev/outrig/pkg/ds"
)

func enableExternalLogWrapImpl(_ bool, _ ds.LogProcessorConfig) error {
	return errors.New("external log wrapping not supported on Windows")
}

func disableExternalLogWrapImpl() {
	// No-op on Windows
}

func isExternalLogWrapActiveImpl() bool {
	// External log wrapping is not supported on Windows
	return false
}
