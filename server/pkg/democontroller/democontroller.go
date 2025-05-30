// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package democontroller

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

const (
	StatusStopped = "stopped"
	StatusRunning = "running"
	StatusError   = "error"
)

type DemoController struct {
	mu     sync.RWMutex
	cmd    *exec.Cmd
	status string
	err    error
}

var globalController = &DemoController{
	status: StatusStopped,
}

func LaunchDemoApp() error {
	globalController.mu.Lock()
	defer globalController.mu.Unlock()

	if globalController.cmd != nil && globalController.cmd.Process != nil {
		return fmt.Errorf("demo app is already running")
	}

	executable, err := os.Executable()
	if err != nil {
		globalController.status = StatusError
		globalController.err = err
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command(executable, "demo", "--no-browser-launch")
	
	err = cmd.Start()
	if err != nil {
		globalController.status = StatusError
		globalController.err = err
		return fmt.Errorf("failed to start demo app: %w", err)
	}

	globalController.cmd = cmd
	globalController.status = StatusRunning
	globalController.err = nil

	go func() {
		err := cmd.Wait()
		globalController.mu.Lock()
		defer globalController.mu.Unlock()
		
		if err != nil {
			globalController.status = StatusError
			globalController.err = err
		} else {
			globalController.status = StatusStopped
			globalController.err = nil
		}
		globalController.cmd = nil
	}()

	return nil
}

func KillDemoApp() error {
	globalController.mu.Lock()
	defer globalController.mu.Unlock()

	if globalController.cmd == nil || globalController.cmd.Process == nil {
		return fmt.Errorf("demo app is not running")
	}

	err := globalController.cmd.Process.Kill()
	if err != nil {
		globalController.status = StatusError
		globalController.err = err
		return fmt.Errorf("failed to kill demo app: %w", err)
	}

	globalController.status = StatusStopped
	globalController.err = nil
	globalController.cmd = nil

	return nil
}

func GetDemoAppStatus() (string, error) {
	globalController.mu.RLock()
	defer globalController.mu.RUnlock()
	
	return globalController.status, globalController.err
}