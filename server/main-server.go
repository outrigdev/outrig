package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/boot"
	"github.com/outrigdev/outrig/server/pkg/execlogwrap"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/spf13/cobra"
)

// OutrigVersion is the current version of Outrig
var OutrigVersion = "v0.0.0"

// OutrigBuildTime is the build timestamp of Outrig
var OutrigBuildTime = ""
var appRunIdRe = regexp.MustCompile(`apprunid:([a-zA-Z0-9-]+)`)

func sendData(source string, appRunId string, data []byte) {

}

func captureLogsSource(source string, input io.Reader, output io.Writer) error {
	var appRunId string
	lineBuf := utilfn.MakeLineBuf()
	err := utilfn.TeeCopy(input, output, func(data []byte) {
		if appRunId == "" {
			lines := lineBuf.ProcessBuf(data)
			for _, line := range lines {
				if strings.Contains(line, "[outrig]") && strings.Contains(line, "apprunid:") {
					matches := appRunIdRe.FindStringSubmatch(line)
					if matches == nil {
						continue
					}
					if _, err := uuid.Parse(matches[1]); err != nil {
						continue
					}
					appRunId = matches[1]
				}
				if appRunId != "" {
					sendData(source, appRunId, []byte(line))
				}
			}
			if appRunId != "" {
				partialBuf := lineBuf.GetPartialAndReset()
				if len(partialBuf) > 0 {
					sendData(source, appRunId, []byte(partialBuf))
				}
			}
			return
		}
		sendData(source, appRunId, data)
	})
	return err
}

// runCaptureLogs captures logs from stdin and fd 3 and writes them to stdout and stderr respectively
func runCaptureLogs(cmd *cobra.Command, args []string) error {
	// Create a file for fd 3 (stderr input)
	stderrIn := os.NewFile(3, "stderr-in")

	// Get the source flag value
	source, _ := cmd.Flags().GetString("source")
	if source == "" {
		source = "/dev/stdout"
	}

	var wg sync.WaitGroup

	// Start goroutine to copy from stdin to stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := captureLogsSource(source, os.Stdin, os.Stdout)
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
			err := captureLogsSource("/dev/stderr", stderrIn, os.Stderr)
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error copying from stderr-in to stderr: %v\n", err)
			}
		}()
	}

	// Wait for both copy operations to complete
	wg.Wait()
	return nil
}

// processDevFlag checks if the --dev flag is present in the arguments and returns
// the filtered arguments (without --dev) and a boolean indicating if --dev was found
func processDevFlag(args []string) ([]string, bool) {
	isDev := false
	filteredArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--dev" {
			isDev = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	return filteredArgs, isDev
}

func main() {
	// Set serverbase version from main version (which gets overridden by build tags)
	serverbase.OutrigVersion = OutrigVersion
	serverbase.OutrigBuildTime = OutrigBuildTime

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

	// Add source flag to capturelogs command
	captureLogsCmd.Flags().String("source", "", "Override the source name for stdout logs (default: /dev/stdout)")

	// Create the run command
	runCmd := &cobra.Command{
		Use:   "run [go-args]",
		Short: "Run a Go program with Outrig logging",
		Long: `Run a Go program with Outrig logging. All arguments after run are passed to the go command.
Note: The --dev flag must come before the run command to be recognized as an Outrig flag.
Example: outrig --dev run main.go`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filteredArgs, isDev := processDevFlag(args)
			goArgs := append([]string{"go", "run"}, filteredArgs...)
			return execlogwrap.ExecCommand(goArgs, isDev)
		},
		// Disable flag parsing for this command so all flags are passed to the go command
		DisableFlagParsing: true,
	}

	// Create the exec command
	execCmd := &cobra.Command{
		Use:   "exec [command]",
		Short: "Execute a command with Outrig logging",
		Long: `Execute a command with Outrig logging. All arguments after exec are passed to the command.
Note: The --dev flag must come before the exec command to be recognized as an Outrig flag.
Example: outrig --dev exec ls -latrh`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filteredArgs, isDev := processDevFlag(args)
			return execlogwrap.ExecCommand(filteredArgs, isDev)
		},
		// Disable flag parsing for this command so all flags are passed to the executed command
		DisableFlagParsing: true,
	}

	// Add commands to the root command
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(captureLogsCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(runCmd)

	// Add dev flag to root command (will be inherited by all subcommands)
	rootCmd.PersistentFlags().Bool("dev", false, "Run in development mode")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
