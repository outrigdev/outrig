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
	"syscall"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/ioutrig"
)

var (
	origStdoutFD, origStderrFD int
	stdoutPipeW, stderrPipeW   *os.File
	origStdout, origStderr     *os.File      // Store original file structures
	externalCaptureLock        sync.Mutex
	externalCaptureActive      bool
	externalCaptureContext     context.Context
	externalCaptureCancel      context.CancelFunc
	externalCaptureExitChan    chan struct{} // Reference to the exit channel
	wrapStdout, wrapStderr     bool          // Track which streams are being wrapped
)

func enableExternalLogWrapImpl(appRunId string, config ds.LogProcessorConfig, isDev bool) error {
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
	outrigPath := "outrig" // Default to looking up in PATH
	if config.OutrigPath != "" {
		outrigPath = config.OutrigPath
	} else {
		// Check if outrig is in the PATH when no custom path is provided
		if _, lookPathErr := exec.LookPath("outrig"); lookPathErr != nil {
			return fmt.Errorf("outrig command not found in PATH: %w", lookPathErr)
		}
	}

	// Duplicate original file descriptors to save them
	var err error
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

	var stdoutPipeR *os.File
	stdoutPipeR, stdoutPipeW, err = os.Pipe()
	if err != nil {
		syscall.Close(origStdoutFD)
		syscall.Close(origStderrFD)
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderrPipeR *os.File
	stderrPipeR, stderrPipeW, err = os.Pipe()
	if err != nil {
		syscall.Close(origStdoutFD)
		syscall.Close(origStderrFD)
		stdoutPipeR.Close()
		stdoutPipeW.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Create context with cancellation for the external process and store in local variable
	ctx, cancelFn := context.WithCancel(context.Background())
	externalCaptureContext, externalCaptureCancel = ctx, cancelFn

	// Launch the external process BEFORE redirecting stdout/stderr
	cmd := exec.CommandContext(externalCaptureContext, outrigPath)
	
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
		cleanupPipesAndFDs()
		externalCaptureCancel()
		return fmt.Errorf("failed to start external log capture process: %w", err)
	}

	// Close read ends of pipes as the child process now owns them
	stdoutPipeR.Close()
	stderrPipeR.Close()

	// Now that the process is started, redirect stdout and stderr to pipe write ends if enabled
	if wrapStdout {
		err = syscall.Dup2(int(stdoutPipeW.Fd()), int(os.Stdout.Fd()))
		if err != nil {
			// Kill the process we just started
			cmd.Process.Kill()
			cleanupPipesAndFDs()
			externalCaptureCancel()
			return fmt.Errorf("failed to redirect stdout: %w", err)
		}
	}

	if wrapStderr {
		err = syscall.Dup2(int(stderrPipeW.Fd()), int(os.Stderr.Fd()))
		if err != nil {
			// Restore stdout before returning if it was wrapped
			if wrapStdout {
				syscall.Dup2(origStdoutFD, int(os.Stdout.Fd()))
			}
			// Kill the process we just started
			cmd.Process.Kill()
			cleanupPipesAndFDs()
			externalCaptureCancel()
			return fmt.Errorf("failed to redirect stderr: %w", err)
		}
	}

	externalCaptureActive = true

	// Create channel for process exit notification
	exitChan := make(chan struct{})
	externalCaptureExitChan = exitChan

	// Start monitoring goroutine and pass all necessary parameters to avoid race conditions
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig ExternalLogCapture:monitor")
		monitorExternalProcess(cmd, ctx, exitChan)
	}()

	// Print a message to the redirected stdout (will be captured by the external process)
	fmt.Println("[outrig] External log capture process started")

	return nil
}

func disableExternalLogWrapImpl() {
	externalCaptureLock.Lock()
	defer externalCaptureLock.Unlock()

	if !externalCaptureActive {
		return
	}

	// First, restore original file descriptors (atomic)
	restoreOriginalFDs()

	// no wait is necessar because Dup2 is atomic

	// Close write ends of pipes to signal EOF to the process
	// This should cause it to exit gracefully
	stdoutPipeW.Close()
	stdoutPipeW = nil
	stderrPipeW.Close()
	stderrPipeW = nil

	// Try to wait for the process to exit naturally after receiving EOF
	select {
	case <-externalCaptureExitChan:
		// Process has already exited
	case <-time.After(100 * time.Millisecond):
		// Process didn't exit after receiving EOF, use context cancellation as fallback
		externalCaptureCancel()
	}

	// Reset the exit channel reference
	externalCaptureExitChan = nil

	// Clean up remaining resources
	cleanupPipesAndFDs()

	externalCaptureActive = false
	externalCaptureCancel = nil
}

// monitorExternalProcess monitors the external process and calls DisableExternalLogWrap if it exits unexpectedly
// we pass these variables as parameters as this is not synchronized with the lock
func monitorExternalProcess(cmd *exec.Cmd, ctx context.Context, exitChan chan struct{}) {
	if cmd == nil {
		return
	}

	// Wait for the process to exit - this should be the ONLY place that calls Wait()
	err := cmd.Wait()

	// Signal that the process has exited
	close(exitChan)

	// Check if this was an expected termination (context cancelled)
	select {
	case <-ctx.Done():
		// This was an expected termination, no need to do anything
		return
	default:
		// This was an unexpected termination
		fmt.Fprintf(os.NewFile(uintptr(origStderrFD), "stderr"),
			"[outrig] External log capture process exited unexpectedly: %v\n", err)

		// Call DisableExternalLogWrap to restore original file descriptors
		DisableExternalLogWrap()
	}
}

// restoreOriginalFDs restores the original stdout and stderr file descriptors
// must be called while holding the lock
func restoreOriginalFDs() {
	// Restore stdout if it was wrapped
	if wrapStdout && origStdoutFD != 0 {
		syscall.Dup2(origStdoutFD, int(os.Stdout.Fd()))
	}

	// Restore stderr if it was wrapped
	if wrapStderr && origStderrFD != 0 {
		syscall.Dup2(origStderrFD, int(os.Stderr.Fd()))
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
func cleanupPipesAndFDs() {
	// Close pipes
	if stdoutPipeW != nil {
		stdoutPipeW.Close()
		stdoutPipeW = nil
	}
	if stderrPipeW != nil {
		stderrPipeW.Close()
		stderrPipeW = nil
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
