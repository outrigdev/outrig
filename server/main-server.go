package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

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
func shouldRunGoCommand() bool {
	// Look for the first argument that doesn't start with a "-"
	for i := 1; i < len(os.Args); i++ {
		if !strings.HasPrefix(os.Args[i], "-") {
			// If it's "run", we should run the go command
			return os.Args[i] == "run"
		}
	}
	return false
}

// runGoCommand executes a go command with the provided arguments
func runGoCommand() error {
	// Replace "outrig" with "go" and execute
	execArgs := make([]string, len(os.Args))
	execArgs[0] = "go"
	copy(execArgs[1:], os.Args[1:])

	execCmd := exec.Command(execArgs[0], execArgs[1:]...)
	
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
		err := utilfn.TeeCopy(stdoutPipe, os.Stdout, nil)
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Error copying stdout: %v\n", err)
		}
	}()
	
	// Process stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := utilfn.TeeCopy(stderrPipe, os.Stderr, nil)
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Error copying stderr: %v\n", err)
		}
	}()
	
	// Wait for both stdout and stderr processing to complete
	wg.Wait()
	
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
	if shouldRunGoCommand() {
		err := runGoCommand()
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
