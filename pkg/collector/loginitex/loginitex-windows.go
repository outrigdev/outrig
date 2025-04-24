//go:build windows

// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loginitex

import (
	"errors"
	"os"

	"github.com/outrigdev/outrig/pkg/ds"
)

func enableExternalLogWrapImpl(_ string, _ ds.LogProcessorConfig, _ bool) error {
	return errors.New("external log wrapping not supported on Windows")
}

func disableExternalLogWrapImpl() {
	// No-op on Windows
}

func isExternalLogWrapActiveImpl() bool {
	// External log wrapping is not supported on Windows
	return false
}

func OrigStdout() *os.File {
	return os.Stdout
}

func OrigStderr() *os.File {
	return os.Stderr
}
