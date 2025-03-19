package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/boot"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/spf13/cobra"
)

// OutrigVersion is the current version of Outrig
var OutrigVersion = "v0.0.0"

// OutrigBuildTime is the build timestamp of Outrig
var OutrigBuildTime = ""

// shouldRunGoCommand checks if the command line arguments indicate we should run the go command
// It returns the processed arguments, a flag indicating if dev mode is enabled, and a flag indicating if we should run the go command
func shouldRunGoCommand() ([]string, bool, bool) {
	newArgs := make([]string, 0, len(os.Args))
	newArgs = append(newArgs, "go") // Replace "outrig" with "go"
	
	isDev := false
	foundRun := false
	
	// Process all arguments
	for i := 1; i < len(os.Args); i++ {
		// Check for --dev flag before the "run" command
		if os.Args[i] == "--dev" && !foundRun {
			isDev = true
			continue // Skip this argument
		}
		
		// If we find a non-flag argument that isn't "run", we're not running a go command
		if !strings.HasPrefix(os.Args[i], "-") && !foundRun {
			if os.Args[i] != "run" {
				// Not a "run" command, so we're not running a go command
				return nil, false, false
			}
			foundRun = true
		}
		
		// Add the argument to the new args list
		newArgs = append(newArgs, os.Args[i])
	}
	
	return newArgs, isDev, foundRun
}

// Connection management
var stdoutConn atomic.Pointer[comm.ConnWrap]
var stderrConn atomic.Pointer[comm.ConnWrap]
var connLock sync.Mutex
var connPollerStarted bool

const ConnPollTime = 1 * time.Second

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
	var connPtr *comm.ConnWrap

	// Get the appropriate connection based on the source
	if source == "/dev/stdout" {
		connPtr = stdoutConn.Load()
	} else if source == "/dev/stderr" {
		connPtr = stderrConn.Load()
	}

	// If we have a connection, send the data
	if connPtr != nil {
		// Send the data to the server
		_, err := connPtr.Conn.Write(append(data, '\n'))
		if err != nil {
			// Connection error, clear the connection so we'll try to reconnect
			if source == "/dev/stdout" {
				stdoutConn.Store(nil)
			} else if source == "/dev/stderr" {
				stderrConn.Store(nil)
			}
		}
	}
}

// runGoCommand executes a go command with the provided arguments
func runGoCommand(args []string, isDev bool) error {
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

// runCaptureLogs captures logs from stdin and fd 3 and writes them to stdout and stderr respectively
func runCaptureLogs(cmd *cobra.Command, args []string) error {
	// Create a file for fd 3 (stderr input)
	stderrIn := os.NewFile(3, "stderr-in")

	var wg sync.WaitGroup

	// Start goroutine to copy from stdin to stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := utilfn.TeeCopy(os.Stdin, os.Stdout, nil)
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Error copying from stdin to stdout: %v\n", err)
		}
	}()

	// If we have a valid stderr input file, copy from it to stderr
	if stderrIn != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer stderrIn.Close()
			err := utilfn.TeeCopy(stderrIn, os.Stderr, nil)
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error copying from stderr-in to stderr: %v\n", err)
			}
		}()
	}

	// Wait for both copy operations to complete
	wg.Wait()
	return nil
}

func main() {
	// Set serverbase version from main version (which gets overridden by build tags)
	serverbase.OutrigVersion = OutrigVersion
	serverbase.OutrigBuildTime = OutrigBuildTime

	// Check if we should run the go command
	args, isDev, shouldRun := shouldRunGoCommand()
	if shouldRun {
		err := runGoCommand(args, isDev)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create the root command
	rootCmd := &cobra.Command{
		Use:   "outrig",
		Short: "Outrig provides real-time debugging for Go programs",
		Long:  `Outrig provides real-time debugging for Go programs, similar to Chrome DevTools.`,
		// No Run function for root command - it will just display help and exit
	}

	// Create the server command
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run the Outrig server",
		Long:  `Run the Outrig server which provides real-time debugging capabilities.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return boot.RunServer()
		},
	}

	// Create the version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Outrig",
		Long:  `Print the version number of Outrig and exit.`,
		Run: func(cmd *cobra.Command, args []string) {
			if OutrigBuildTime != "" {
				fmt.Printf("%s+%s\n", OutrigVersion, OutrigBuildTime)
			} else {
				fmt.Printf("%s+dev\n", OutrigVersion)
			}
		},
	}

	// Create the capturelogs command
	captureLogsCmd := &cobra.Command{
		Use:   "capturelogs",
		Short: "Capture logs from stdin and fd 3",
		Long:  `Capture logs from stdin (stdout of the process) and fd 3 (stderr of the process) and write them to stdout and stderr respectively.`,
		RunE:  runCaptureLogs,
	}

	// Add commands to the root command
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(captureLogsCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
