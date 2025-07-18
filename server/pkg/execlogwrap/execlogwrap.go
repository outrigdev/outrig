// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package execlogwrap

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const ConnPollTime = 1 * time.Second

// LogDataWrap encapsulates a connection with its lock and related functionality
type LogDataWrap struct {
	conn   *comm.ConnWrap
	lock   sync.Mutex
	source string
}

// TeeStreamDecl defines a stream to be processed with TeeCopy
type TeeStreamDecl struct {
	Input  io.Reader
	Output io.Writer
	Source string
}

var stdoutWrap = LogDataWrap{source: "/dev/stdout"}
var stderrWrap = LogDataWrap{source: "/dev/stderr"}

// getLogDataWrap returns the appropriate LogDataWrap for the given source
func getLogDataWrap(source string) *LogDataWrap {
	if source == "/dev/stdout" {
		return &stdoutWrap
	} else if source == "/dev/stderr" {
		return &stderrWrap
	}
	return nil
}

// processLogData sends log data to the connection if available
func (ldw *LogDataWrap) processLogData(data []byte) {
	ldw.lock.Lock()
	defer ldw.lock.Unlock()

	if ldw.conn == nil {
		return
	}

	_, err := ldw.conn.Conn.Write(data)
	if err != nil {
		ldw.conn = nil
	}
}

// ensureConnection ensures that we have a connection to the Outrig server
func (ldw *LogDataWrap) ensureConnection(cfg *config.Config) {
	ldw.lock.Lock()
	defer ldw.lock.Unlock()

	if ldw.conn == nil {
		if conn := tryConnect(ldw.source, cfg); conn != nil {
			ldw.conn = conn
			// fmt.Printf("#outrig connected %s via %s\n", ldw.source, conn.PeerName)
		}
	}
}

// closeConnection closes the connection and resets the connection pointer
func (ldw *LogDataWrap) closeConnection() {
	ldw.lock.Lock()
	defer ldw.lock.Unlock()

	if ldw.conn != nil {
		ldw.conn.Close()
		ldw.conn = nil
	}
}

// tryConnect attempts to connect to the Outrig server for the specified source
// It returns the connection if successful, or nil if it fails
func tryConnect(source string, cfg *config.Config) *comm.ConnWrap {
	appRunId := config.GetExternalAppRunId()
	if appRunId == "" {
		// it is an error if we don't have an _external_ apprunid
		return nil
	}

	connWrap, _, transErr := comm.Connect(comm.ConnectionModeLog, source, appRunId, cfg)
	if transErr != nil {
		return nil
	}
	return connWrap
}

// ensureConnections ensures that we have connections to the Outrig server
// for both stdout and stderr
func ensureConnections(cfg *config.Config) {
	stdoutWrap.ensureConnection(cfg)
	stderrWrap.ensureConnection(cfg)
}

// startConnPoller starts a goroutine that periodically tries to establish
// connections to the Outrig server if they don't already exist
func startConnPoller(cfg *config.Config) {
	go func() {
		for {
			ensureConnections(cfg)
			time.Sleep(ConnPollTime)
		}
	}()
}

// closeConnections closes any open connections and resets the connection pointers
func closeConnections() {
	stdoutWrap.closeConnection()
	stderrWrap.closeConnection()
}

// processStream processes a stream using TeeCopy in a goroutine
func processStream(wg *sync.WaitGroup, decl TeeStreamDecl) {
	ldw := getLogDataWrap(decl.Source)

	wg.Add(1)
	go func() {
		defer wg.Done()
		utilfn.TeeCopy(decl.Input, decl.Output, func(data []byte) {
			if ldw != nil {
				ldw.processLogData(data)
			}
		})
		// do not log errors, just ignore
	}()
}

// ProcessExistingStreams handles capturing logs from provided input/output streams
func ProcessExistingStreams(streams []TeeStreamDecl, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	appRunId := config.GetExternalAppRunId()
	if appRunId != "" {
		ensureConnections(cfg)
		startConnPoller(cfg)
	}

	var wg sync.WaitGroup
	for _, stream := range streams {
		processStream(&wg, stream)
	}
	wg.Wait()
	closeConnections()

	return nil
}

// ExecCommand executes a command with the provided arguments
func ExecCommand(args []string) error {
	execCmd := exec.Command(args[0], args[1:]...)

	// Set up environment variables for external log capture on the command
	execCmd.Env = os.Environ()
	appRunId := config.GetAppRunId()
	if appRunId != "" {
		execCmd.Env = append(execCmd.Env, config.AppRunIdEnvName+"="+appRunId)
	}
	execCmd.Env = append(execCmd.Env, config.ExternalLogCaptureEnvName+"=1")

	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	execCmd.Stdin = os.Stdin

	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	streams := []TeeStreamDecl{
		{Input: stdoutPipe, Output: os.Stdout, Source: "/dev/stdout"},
		{Input: stderrPipe, Output: os.Stderr, Source: "/dev/stderr"},
	}
	ProcessExistingStreams(streams, nil)

	err = execCmd.Wait()
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	return err
}
