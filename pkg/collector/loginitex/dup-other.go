//go:build !linux && !windows

// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loginitex

import (
	"syscall"
)

// dup2Wrap on non-Linux systems uses dup2
func dup2Wrap(oldfd, newfd int) error {
	err := syscall.Dup2(oldfd, newfd)
	if err != nil {
		return err
	}
	syscall.CloseOnExec(newfd)
	return nil
}
