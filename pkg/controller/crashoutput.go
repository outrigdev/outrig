//go:build go1.23

package controller

import (
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// CrashOutputConnPollTime is the interval at which we poll for crash output connection
const CrashOutputConnPollTime = 5 * time.Second

// crashOutputConnected is an atomic flag indicating if we have a crash output connection
var crashOutputConnected atomic.Bool

// crashOutputPollerOnce ensures the poller is started only once
var crashOutputPollerOnce sync.Once

// setupCrashOutput attempts to establish a connection for crash output
// and returns a file descriptor that can be used with SetCrashOutput
func (c *ControllerImpl) setupCrashOutput() (*os.File, error) {
	// Use the same domain socket path as the main connection
	dsPath := utilfn.ExpandHomeDir(c.config.DomainSocketPath)
	if c.config.DomainSocketPath == "-" || dsPath == "" {
		return nil, fmt.Errorf("domain socket is disabled")
	}

	// Check if the domain socket exists
	if _, errStat := os.Stat(dsPath); errStat != nil {
		return nil, fmt.Errorf("domain socket not found: %v", errStat)
	}

	// Connect to the domain socket
	conn, err := net.DialTimeout("unix", dsPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to domain socket: %v", err)
	}

	// Send the sentinel message to identify this as a crash output connection
	sentinelMsg := fmt.Sprintf("CRASHOUTPUT %s\n", c.AppInfo.AppRunId)
	_, err = conn.Write([]byte(sentinelMsg))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send sentinel message: %v", err)
	}

	// Get the underlying file descriptor from the connection
	// This is the key part - we need to get a file descriptor that can be used with SetCrashOutput
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("connection is not a unix domain socket connection")
	}

	// Get a copy of the file descriptor
	// Note: File() returns a dup of the fd, so we don't need to worry about the original being closed
	file, err := unixConn.File()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get file descriptor from connection: %v", err)
	}

	// We can close the original connection now since we have a dup of the fd
	conn.Close()

	crashOutputConnected.Store(true)
	return file, nil
}

// runCrashOutputConnPoller continuously tries to establish and maintain
// a crash output connection
func (c *ControllerImpl) runCrashOutputConnPoller() {
	crashOutputPollerOnce.Do(func() {
		for {
			config := c.GetConfig()
			if config.LogProcessorConfig != nil &&
				config.LogProcessorConfig.CaptureUnhandledCrashes &&
				global.OutrigEnabled.Load() &&
				!crashOutputConnected.Load() {

				crashFile, err := c.setupCrashOutput()
				if err == nil {
					// Set the new crash output file
					err = debug.SetCrashOutput(crashFile, debug.CrashOptions{})
					if err != nil {
						fmt.Printf("Failed to set crash output: %v\n", err)
					} else {
						fmt.Printf("Crash output configured for app run %s (%v)\n", c.AppInfo.AppRunId, crashFile)
					}
					crashFile.Close() // ok, since SetCrashOuput dups the file
				}
			}

			time.Sleep(CrashOutputConnPollTime)
		}
	})
}

// initCrashOutput initializes crash output handling if enabled in the config
func (c *ControllerImpl) initCrashOutput() {
	if c.config.LogProcessorConfig != nil && c.config.LogProcessorConfig.CaptureUnhandledCrashes {
		go c.runCrashOutputConnPoller()
	}
}
