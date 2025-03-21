package execlogwrap

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// Connection management
var stdoutConn atomic.Pointer[comm.ConnWrap]
var stderrConn atomic.Pointer[comm.ConnWrap]
var connLock sync.Mutex
var connPollerStarted bool

const ConnPollTime = 1 * time.Second

func getAtomicPointerForSource(source string) *atomic.Pointer[comm.ConnWrap] {
	if source == "/dev/stdout" {
		return &stdoutConn
	} else if source == "/dev/stderr" {
		return &stderrConn
	}
	return nil
}

// tryConnect attempts to connect to the Outrig server for the specified source
// It returns the connection if successful, or nil if it fails
func tryConnect(source string, isDev bool) *comm.ConnWrap {
	// Get the app run ID from the environment
	appRunId := os.Getenv("OUTRIG_APPRUNID")
	if appRunId == "" {
		return nil
	}

	// Get the domain socket path and TCP address using the base package functions
	domainSocketPath := base.GetDomainSocketNameForClient(isDev)
	serverAddr := base.GetTCPAddrForClient(isDev)

	// Try to connect
	connWrap, err := comm.Connect(base.ConnectionModeLog, source, appRunId, domainSocketPath, serverAddr)
	if err != nil {
		return nil
	}

	return connWrap
}

// ensureConnections ensures that we have connections to the Outrig server
// for both stdout and stderr
func ensureConnections(isDev bool) {
	connLock.Lock()
	defer connLock.Unlock()

	// Try to connect stdout if not already connected
	if stdoutConn.Load() == nil {
		if conn := tryConnect("/dev/stdout", isDev); conn != nil {
			stdoutConn.Store(conn)
			fmt.Printf("[outrig] connected stdout via %s\n", conn.PeerName)
		}
	}

	// Try to connect stderr if not already connected
	if stderrConn.Load() == nil {
		if conn := tryConnect("/dev/stderr", isDev); conn != nil {
			stderrConn.Store(conn)
			fmt.Printf("[outrig] connected stderr via %s\n", conn.PeerName)
		}
	}
}

// startConnPoller starts a goroutine that periodically tries to establish
// connections to the Outrig server if they don't already exist
func startConnPoller(isDev bool) {
	connLock.Lock()
	defer connLock.Unlock()

	if connPollerStarted {
		return
	}

	connPollerStarted = true

	go func() {
		for {
			ensureConnections(isDev)
			time.Sleep(ConnPollTime)
		}
	}()
}

// ProcessLogData processes log data from a specific source
// It sends the data to the appropriate Outrig connection if available
func ProcessLogData(source string, data []byte) {
	ptr := getAtomicPointerForSource(source)
	if ptr == nil {
		return
	}
	connPtr := ptr.Load()
	if connPtr == nil {
		return
	}
	// Send the data to the server
	_, err := connPtr.Conn.Write(data)
	if err != nil {
		// Connection error, clear the connection so we'll try to reconnect
		ptr.Store(nil)
	}
}

// runGoCommand executes a go command with the provided arguments
func ExecCommand(args []string, isDev bool) error {
	// Generate a UUID for the app run ID
	appRunId := uuid.New().String()

	// Set the environment variable directly
	os.Setenv("OUTRIG_APPRUNID", appRunId)

	// Create the command
	execCmd := exec.Command(args[0], args[1:]...)

	// Create pipes for stdout and stderr
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Connect stdin directly
	execCmd.Stdin = os.Stdin

	// Try to connect to the Outrig server initially
	ensureConnections(isDev)

	// Start the connection poller to maintain connections
	startConnPoller(isDev)

	// Start the command
	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	// Use WaitGroup to wait for both stdout and stderr processing to complete
	var wg sync.WaitGroup

	// Process stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := utilfn.TeeCopy(stdoutPipe, os.Stdout, func(data []byte) {
			ProcessLogData("/dev/stdout", data)
		})
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Error copying stdout: %v\n", err)
		}
	}()

	// Process stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := utilfn.TeeCopy(stderrPipe, os.Stderr, func(data []byte) {
			ProcessLogData("/dev/stderr", data)
		})
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Error copying stderr: %v\n", err)
		}
	}()

	// Wait for both stdout and stderr processing to complete
	wg.Wait()

	// Close any open connections
	if conn := stdoutConn.Load(); conn != nil {
		conn.Close()
		stdoutConn.Store(nil)
	}

	if conn := stderrConn.Load(); conn != nil {
		conn.Close()
		stderrConn.Store(nil)
	}

	// Wait for the command to complete
	err = execCmd.Wait()
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	return err
}
