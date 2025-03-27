package execlogwrap

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

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
	appRunId := os.Getenv("OUTRIG_APPRUNID")
	if appRunId == "" {
		return nil
	}

	domainSocketPath := base.GetDomainSocketNameForClient(isDev)
	serverAddr := base.GetTCPAddrForClient(isDev)

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
			// fmt.Printf("[outrig] connected stdout via %s\n", conn.PeerName)
		}
	}

	// Try to connect stderr if not already connected
	if stderrConn.Load() == nil {
		if conn := tryConnect("/dev/stderr", isDev); conn != nil {
			stderrConn.Store(conn)
			// fmt.Printf("[outrig] connected stderr via %s\n", conn.PeerName)
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

// ProcessLogData sends log data to the appropriate Outrig connection if available
func ProcessLogData(source string, data []byte) {
	ptr := getAtomicPointerForSource(source)
	if ptr == nil {
		return
	}
	connPtr := ptr.Load()
	if connPtr == nil {
		return
	}

	_, err := connPtr.Conn.Write(data)
	if err != nil {
		ptr.Store(nil)
	}
}

// closeConnections closes any open connections and resets the connection pointers
func closeConnections() {
	if conn := stdoutConn.Load(); conn != nil {
		conn.Close()
		stdoutConn.Store(nil)
	}

	if conn := stderrConn.Load(); conn != nil {
		conn.Close()
		stderrConn.Store(nil)
	}
}

// TeeStreamDecl defines a stream to be processed with TeeCopy
type TeeStreamDecl struct {
	Input  io.Reader
	Output io.Writer
	Source string
}

// processStream processes a stream using TeeCopy in a goroutine
func processStream(wg *sync.WaitGroup, decl TeeStreamDecl) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := utilfn.TeeCopy(decl.Input, decl.Output, func(data []byte) {
			ProcessLogData(decl.Source, data)
		})
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Error copying %s: %v\n", decl.Source, err)
		}
	}()
}

// ProcessExistingStreams handles capturing logs from provided input/output streams
func ProcessExistingStreams(streams []TeeStreamDecl, isDev bool) error {
	appRunId := os.Getenv("OUTRIG_APPRUNID")

	if appRunId != "" {
		ensureConnections(isDev)
		startConnPoller(isDev)
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
func ExecCommand(args []string, isDev bool) error {
	execCmd := exec.Command(args[0], args[1:]...)

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
	ProcessExistingStreams(streams, isDev)

	err = execCmd.Wait()
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	return err
}
