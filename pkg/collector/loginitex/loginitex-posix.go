//go:build !windows

// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loginitex

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

var (
	origStdoutFD, origStderrFD int
	origStdout, origStderr     *os.File // Store original file structures
	externalCaptureLock        sync.Mutex
	externalCaptureActive      bool
	wrapStdout, wrapStderr     bool // Track which streams are being wrapped

	activeExtProc *extCaptureProc // Store the external process reference
)

type extCaptureProc struct {
	stdoutPipeW, stderrPipeW *os.File
	externalCaptureContext   context.Context
	externalCaptureCancel    context.CancelFunc
	externalCaptureExitChan  chan struct{} // Reference to the exit channel
	cmd                      *exec.Cmd     // Reference to the command
	closing                  atomic.Bool   // Flag to indicate that disableExternalLogWrapImpl is running
}

func enableExternalLogWrapImpl(appRunId string, config config.LogProcessorConfig, isDev bool) error {
	externalCaptureLock.Lock()
	defer externalCaptureLock.Unlock()

	if externalCaptureActive {
		return nil // Already active
	}

	// If both stdout and stderr wrapping are disabled, do nothing
	if !config.WrapStdout && !config.WrapStderr {
		return nil
	}

	// Set which streams to wrap
	wrapStdout = config.WrapStdout
	wrapStderr = config.WrapStderr

	// Determine the outrig executable path
	outrigPath, err := resolveOutrigPath(config)
	if err != nil {
		return err
	}

	// Duplicate original file descriptors to save them
	origStdoutFD, err = syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("failed to duplicate stdout fd: %w", err)
	}

	origStderrFD, err = syscall.Dup(int(os.Stderr.Fd()))
	if err != nil {
		syscall.Close(origStdoutFD)
		return fmt.Errorf("failed to duplicate stderr fd: %w", err)
	}

	// Store original file structures
	origStdout = os.NewFile(uintptr(origStdoutFD), "stdout")
	origStderr = os.NewFile(uintptr(origStderrFD), "stderr")

	// Initialize a local extCaptureProc struct
	localProc := &extCaptureProc{
		externalCaptureExitChan: make(chan struct{}),
	}
	localProc.externalCaptureContext, localProc.externalCaptureCancel = context.WithCancel(context.Background())

	// Create pipes for stdout and stderr
	var stdoutPipeR *os.File
	stdoutPipeR, localProc.stdoutPipeW, err = os.Pipe()
	if err != nil {
		syscall.Close(origStdoutFD)
		syscall.Close(origStderrFD)
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderrPipeR *os.File
	stderrPipeR, localProc.stderrPipeW, err = os.Pipe()
	if err != nil {
		syscall.Close(origStdoutFD)
		syscall.Close(origStderrFD)
		stdoutPipeR.Close()
		localProc.stdoutPipeW.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Launch the external process BEFORE redirecting stdout/stderr
	cmd := exec.CommandContext(localProc.externalCaptureContext, outrigPath)
	localProc.cmd = cmd // Store command in the struct

	// Set the AppRunId environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", base.AppRunIdEnvName, appRunId))

	// Add any additional arguments before "capturelogs"
	if len(config.AdditionalArgs) > 0 {
		cmd.Args = append(cmd.Args, config.AdditionalArgs...)
	}

	// Add the "capturelogs" command and any flags
	cmd.Args = append(cmd.Args, "capturelogs")
	if isDev {
		cmd.Args = append(cmd.Args, "--dev")
	}

	// Set up file descriptors for the external process
	cmd.Stdin = stdoutPipeR
	cmd.Stdout = os.NewFile(uintptr(origStdoutFD), "stdout")
	cmd.Stderr = os.NewFile(uintptr(origStderrFD), "stderr")
	cmd.ExtraFiles = []*os.File{stderrPipeR} // This will be fd 3 in the child process

	err = cmd.Start()
	if err != nil {
		cleanupPipesAndFDs(localProc)
		localProc.externalCaptureCancel()
		return fmt.Errorf("failed to start external log capture process: %w", err)
	}

	// Close read ends of pipes as the child process now owns them
	stdoutPipeR.Close()
	stderrPipeR.Close()

	// Now that the process is started, redirect stdout and stderr to pipe write ends if enabled
	if wrapStdout {
		err = dup2Wrap(int(localProc.stdoutPipeW.Fd()), int(os.Stdout.Fd()))
		if err != nil {
			// Kill the process we just started
			cmd.Process.Kill()
			cleanupPipesAndFDs(localProc)
			localProc.externalCaptureCancel()
			return fmt.Errorf("failed to redirect stdout: %w", err)
		}
	}

	if wrapStderr {
		err = dup2Wrap(int(localProc.stderrPipeW.Fd()), int(os.Stderr.Fd()))
		if err != nil {
			// Restore stdout before returning if it was wrapped
			if wrapStdout {
				dup2Wrap(origStdoutFD, int(os.Stdout.Fd()))
			}
			// Kill the process we just started
			cmd.Process.Kill()
			cleanupPipesAndFDs(localProc)
			localProc.externalCaptureCancel()
			return fmt.Errorf("failed to redirect stderr: %w", err)
		}
	}

	// Set the global variables
	externalCaptureActive = true
	activeExtProc = localProc

	// Start monitoring goroutine and pass the local struct to avoid race conditions
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig ExternalLogCapture:monitor")
		monitorExternalProcess(localProc)
	}()

	return nil
}

func disableExternalLogWrapImpl() {
	externalCaptureLock.Lock()
	defer externalCaptureLock.Unlock()

	if !externalCaptureActive || activeExtProc == nil {
		return
	}

	// Set the closing flag to prevent redundant calls from the monitor goroutine
	activeExtProc.closing.Store(true)

	// First, restore original file descriptors (atomic)
	restoreOriginalFDs()

	// Give the process a moment to settle (even though Dup2 should be atomic)
	time.Sleep(10 * time.Millisecond)

	// Close write ends of pipes to signal EOF to the process
	// This should cause it to exit gracefully
	if activeExtProc.stdoutPipeW != nil {
		activeExtProc.stdoutPipeW.Close()
		activeExtProc.stdoutPipeW = nil
	}
	if activeExtProc.stderrPipeW != nil {
		activeExtProc.stderrPipeW.Close()
		activeExtProc.stderrPipeW = nil
	}

	// Try to wait for the process to exit naturally after receiving EOF
	select {
	case <-activeExtProc.externalCaptureExitChan:
		// Process has already exited
	case <-time.After(100 * time.Millisecond):
		// Process didn't exit after receiving EOF, use context cancellation as fallback
		activeExtProc.externalCaptureCancel()
	}

	// Clean up remaining resources
	cleanupPipesAndFDs(activeExtProc)

	externalCaptureActive = false
	activeExtProc = nil
}

// monitorExternalProcess monitors the external process and calls DisableExternalLogWrap if it exits unexpectedly
// we pass the proc struct as a parameter as this is not synchronized with the lock
func monitorExternalProcess(proc *extCaptureProc) {
	if proc == nil || proc.cmd == nil {
		return
	}

	// Wait for the process to exit - this should be the ONLY place that calls Wait()
	err := proc.cmd.Wait()

	// Signal that the process has exited
	close(proc.externalCaptureExitChan)

	// Check if this was an expected termination (context cancelled)
	select {
	case <-proc.externalCaptureContext.Done():
		// This was an expected termination, no need to do anything
		return
	default:
		// Check if disableExternalLogWrapImpl is already running
		if !proc.closing.Load() {
			// This was a truly unexpected termination
			fmt.Fprintf(os.NewFile(uintptr(origStderrFD), "stderr"),
				"[outrig] External log capture process exited unexpectedly: %v\n", err)

			// Call DisableExternalLogWrap to restore original file descriptors
			DisableExternalLogWrap()
		}
	}
}

// restoreOriginalFDs restores the original stdout and stderr file descriptors
// must be called while holding the lock
func restoreOriginalFDs() {
	// Restore stdout if it was wrapped
	if wrapStdout && origStdoutFD != 0 {
		dup2Wrap(origStdoutFD, int(os.Stdout.Fd()))
	}

	// Restore stderr if it was wrapped
	if wrapStderr && origStderrFD != 0 {
		dup2Wrap(origStderrFD, int(os.Stderr.Fd()))
	}

}

// isExternalLogWrapActiveImpl returns whether external log wrapping is currently active
func isExternalLogWrapActiveImpl() bool {
	externalCaptureLock.Lock()
	defer externalCaptureLock.Unlock()
	return externalCaptureActive
}

// cleanupPipesAndFDs closes all pipes and duplicated file descriptors
// must be called while holding the lock
func cleanupPipesAndFDs(proc *extCaptureProc) {
	// Close pipes
	if proc != nil {
		if proc.stdoutPipeW != nil {
			proc.stdoutPipeW.Close()
			proc.stdoutPipeW = nil
		}
		if proc.stderrPipeW != nil {
			proc.stderrPipeW.Close()
			proc.stderrPipeW = nil
		}
	}

	// Close duplicated file descriptors
	if origStdoutFD != 0 {
		syscall.Close(origStdoutFD)
		origStdoutFD = 0
	}
	if origStderrFD != 0 {
		syscall.Close(origStderrFD)
		origStderrFD = 0
	}

	// Clear file references
	origStdout = nil
	origStderr = nil
}

// OrigStdout returns the original stdout as an os.File
func OrigStdout() *os.File {
	externalCaptureLock.Lock()
	defer externalCaptureLock.Unlock()
	if origStdout != nil {
		return origStdout
	}
	return os.Stdout
}

// OrigStderr returns the original stderr as an os.File
func OrigStderr() *os.File {
	externalCaptureLock.Lock()
	defer externalCaptureLock.Unlock()
	if origStderr != nil {
		return origStderr
	}
	return os.Stderr
}

// resolveOutrigPath determines the path to the outrig executable
func resolveOutrigPath(config config.LogProcessorConfig) (string, error) {
	// If a custom path is provided, use it
	if config.OutrigPath != "" {
		return config.OutrigPath, nil
	}

	// Check if outrig is in the PATH
	if _, lookPathErr := exec.LookPath("outrig"); lookPathErr == nil {
		return "outrig", nil
	}

	// Try backup directories that might not be in PATH
	backupPaths := []string{
		"/opt/homebrew/bin/outrig",
		"/usr/local/bin/outrig",
		utilfn.ExpandHomeDir("~/.local/bin/outrig"),
	}

	for _, backupPath := range backupPaths {
		if _, err := os.Stat(backupPath); err == nil {
			return backupPath, nil
		}
	}

	return "", fmt.Errorf("outrig command not found in PATH or backup directories (/opt/homebrew/bin, /usr/local/bin, ~/.local/bin)")
}
