//go:build linux

// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loginitex

import "syscall"

// dup2Wrap on Linux uses Dup3 with flags 0 (mimicking dup2)
func dup2Wrap(oldfd, newfd int) error {
	return syscall.Dup3(oldfd, newfd, 0)
}
