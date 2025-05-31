// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package democontroller

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
)

const (
	StatusStopped = "stopped"
	StatusRunning = "running"
	StatusError   = "error"
)

type DemoController struct {
	mu              sync.RWMutex
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	status          string
	err             error
	intentionalKill bool
}

var globalController = &DemoController{
	status: StatusStopped,
}

func init() {
	// Set up watch for demo controller status
	outrig.NewWatch("demo-controller").AsJSON().PollFunc(GetDemoAppStatusForWatch)
}

func startDemoApp() error {
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

	cmd := exec.Command(executable, "demo", "--no-browser-launch", "--close-on-stdin")

	// If server is in dev mode, set OUTRIG_DEVCONFIG for the demo app
	if serverbase.IsDev() {
		cmd.Env = append(os.Environ(), "OUTRIG_DEVCONFIG=1")
	}

	// Set up stdin pipe so the demo will exit when the server dies
	stdin, err := cmd.StdinPipe()
	if err != nil {
		globalController.status = StatusError
		globalController.err = err
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		stdin.Close()
		globalController.status = StatusError
		globalController.err = err
		return fmt.Errorf("failed to start demo app: %w", err)
	}

	globalController.cmd = cmd
	globalController.stdin = stdin
	globalController.status = StatusRunning
	globalController.err = nil
	globalController.intentionalKill = false

	go func() {
		err := cmd.Wait()
		globalController.mu.Lock()
		defer globalController.mu.Unlock()

		if err != nil && !globalController.intentionalKill {
			globalController.status = StatusError
			globalController.err = err
		} else {
			globalController.status = StatusStopped
			globalController.err = nil
		}
		globalController.cmd = nil
		globalController.stdin = nil
		globalController.intentionalKill = false
	}()

	return nil
}

func LaunchDemoApp() error {
	err := startDemoApp()
	if err != nil {
		return err
	}

	// Wait 500ms to see if the process exits immediately (e.g., port already in use)
	time.Sleep(500 * time.Millisecond)

	// Check if the process has already exited
	globalController.mu.RLock()
	status := globalController.status
	cmdErr := globalController.err
	globalController.mu.RUnlock()

	if status == StatusError {
		return fmt.Errorf("demo app failed to start: %w", cmdErr)
	}

	return nil
}

func KillDemoApp() error {
	globalController.mu.Lock()
	defer globalController.mu.Unlock()

	if globalController.cmd == nil || globalController.cmd.Process == nil {
		return fmt.Errorf("demo app is not running")
	}

	globalController.intentionalKill = true
	err := globalController.cmd.Process.Kill()
	if err != nil {
		globalController.intentionalKill = false
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

// GetDemoAppStatusForWatch returns the demo controller status in a format suitable for Outrig watches
func GetDemoAppStatusForWatch() map[string]interface{} {
	globalController.mu.RLock()
	defer globalController.mu.RUnlock()

	result := map[string]interface{}{
		"status": globalController.status,
	}
	
	if globalController.err != nil {
		result["error"] = globalController.err.Error()
	}
	
	if globalController.cmd != nil && globalController.cmd.Process != nil {
		result["pid"] = globalController.cmd.Process.Pid
	}
	
	return result
}
