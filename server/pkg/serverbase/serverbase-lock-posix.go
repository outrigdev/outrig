// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package serverbase

import (
	"log"
	"os"
	"path/filepath"

	"github.com/outrigdev/outrig/pkg/utilfn"
	"golang.org/x/sys/unix"
)

func AcquireOutrigServerLock() (FDLock, error) {
	outrigHome := utilfn.ExpandHomeDir(GetOutrigHome())
	lockFileName := filepath.Join(outrigHome, OutrigLockFile)
	log.Printf("#base acquiring lock on %s\n", lockFileName)
	fd, err := os.OpenFile(lockFileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	err = unix.Flock(int(fd.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		fd.Close()
		return nil, err
	}
	return fd, nil
}
