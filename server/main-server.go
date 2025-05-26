// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/server/pkg/boot"
	"github.com/outrigdev/outrig/server/pkg/execlogwrap"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig/server/pkg/tevent"
	"github.com/outrigdev/outrig/server/pkg/updatecheck"
	"github.com/spf13/cobra"
)

var (
	// these get set via -X during build
	OutrigVersion   = ""
	OutrigBuildTime = ""
	OutrigCommit    = ""
)

func runCaptureLogs(cmd *cobra.Command, args []string) error {
	// Ignore SIGINT, SIGTERM, and SIGHUP signals
	// This ensures the sidecar process doesn't terminate prematurely before the main process
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	stderrIn := os.NewFile(3, "stderr-in")
	source, _ := cmd.Flags().GetString("source")
	isDev, _ := cmd.Flags().GetBool("dev")
	if source == "" {
		source = "/dev/stdout"
	}
	streams := []execlogwrap.TeeStreamDecl{
		{Input: os.Stdin, Output: os.Stdout, Source: source},
	}
	if stderrIn != nil {
		streams = append(streams, execlogwrap.TeeStreamDecl{Input: stderrIn, Output: os.Stderr, Source: "/dev/stderr"})
	}
	return execlogwrap.ProcessExistingStreams(streams, isDev)
}

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

func runPostinstall(cmd *cobra.Command, args []string) {
	brightCyan := "\x1b[96m"
	brightBlueUnderline := "\x1b[94;4m"
	reset := "\x1b[0m"

	// Header
	fmt.Printf("%s*** Outrig %s installed successfully! ***%s\n\n", brightCyan, getVersion(), reset)

	// Quickstart link
	fmt.Println("Quick start (and documentation):")
	fmt.Printf("%shttps://outrig.run/docs/quickstart%s\n\n", brightBlueUnderline, reset)

	// Server start instructions
	fmt.Printf("To start the Outrig server, run:\n%soutrig server%s\n\n", brightCyan, reset)

	// Separator
	fmt.Println("---")

	// Open source call to action
	fmt.Println("Outrig is open source and free for individual users.")
	fmt.Printf("If you find it useful, please support us with a star at\n  %shttps://github.com/outrigdev/outrig%s\n", brightBlueUnderline, reset)
}

func getVersion() string {
	if serverbase.OutrigCommit != "" {
		return fmt.Sprintf("%s+%s", serverbase.OutrigServerVersion, serverbase.OutrigCommit)
	} else {
		return fmt.Sprintf("%s+dev", serverbase.OutrigServerVersion)
	}
}

func main() {
	// Set serverbase version from main version (which gets overridden by build tags)
	if OutrigVersion != "" {
		serverbase.OutrigServerVersion = OutrigVersion
	}
	serverbase.OutrigBuildTime = OutrigBuildTime
	serverbase.OutrigCommit = OutrigCommit

	rootCmd := &cobra.Command{
		Use:   "outrig",
		Short: "Outrig provides real-time debugging for Go programs",
		Long:  `Outrig provides real-time debugging for Go programs, similar to Chrome DevTools.`,
		// No Run function for root command - it will just display help and exit
	}

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run the Outrig server",
		Long:  `Run the Outrig server which provides real-time debugging capabilities.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if telemetry should be disabled
			noTelemetry, _ := cmd.Flags().GetBool("no-telemetry")
			noTelemetryEnv := os.Getenv("OUTRIG_NOTELEMETRY")
			if noTelemetry || noTelemetryEnv != "" {
				if noTelemetry {
					log.Printf("Telemetry collection is disabled (via --no-telemetry flag)\n")
				} else {
					log.Printf("Telemetry collection is disabled (via OUTRIG_NOTELEMETRY env var)\n")
				}
				tevent.Disabled.Store(true)
			}

			// Check if update checking should be disabled
			noUpdateCheck, _ := cmd.Flags().GetBool("no-updatecheck")
			noUpdateCheckEnv := os.Getenv("OUTRIG_NOUPDATECHECK")
			if noUpdateCheck || noUpdateCheckEnv != "" {
				if noUpdateCheck {
					log.Printf("Update checking is disabled (via --no-updatecheck flag)\n")
				} else {
					log.Printf("Update checking is disabled (via OUTRIG_NOUPDATECHECK env var)\n")
				}
				updatecheck.Disabled.Store(true)
			}

			// Get the flag values
			port, _ := cmd.Flags().GetInt("port")
			closeOnStdin, _ := cmd.Flags().GetBool("close-on-stdin")
			fromTrayApp, _ := cmd.Flags().GetBool("from-trayapp")

			// Create CLI config
			config := boot.CLIConfig{
				Port:         port,
				CloseOnStdin: closeOnStdin,
				FromTrayApp:  fromTrayApp,
			}

			return boot.RunServer(config)
		},
	}
	// Add flags to server command
	serverCmd.Flags().Bool("no-telemetry", false, "Disable telemetry collection")
	serverCmd.Flags().Bool("no-updatecheck", false, "Disable checking for updates")
	serverCmd.Flags().Int("port", 0, "Override the default web server port (default: 5005 for production, 6005 for development)")
	serverCmd.Flags().Bool("close-on-stdin", false, "Shut down the server when stdin is closed")
	serverCmd.Flags().Bool("from-trayapp", false, "Indicates the server was started from the tray application")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Outrig",
		Long:  `Print the version number of Outrig and exit.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n", getVersion())
		},
	}

	captureLogsCmd := &cobra.Command{
		Use:   "capturelogs",
		Short: "Capture logs from stdin and fd 3",
		Long:  `Capture logs from stdin (stdout of the process) and fd 3 (stderr of the process) and write them to stdout and stderr respectively.`,
		RunE:  runCaptureLogs,
	}
	captureLogsCmd.Flags().String("source", "", "Override the source name for stdout logs (default: /dev/stdout)")

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

			if os.Getenv(base.AppRunIdEnvName) == "" {
				appRunId := uuid.New().String()
				os.Setenv(base.AppRunIdEnvName, appRunId)
			}

			os.Setenv(base.ExternalLogCaptureEnvName, "1")

			return execlogwrap.ExecCommand(goArgs, isDev)
		},
		// Disable flag parsing for this command so all flags are passed to the go command
		DisableFlagParsing: true,
	}

	execCmd := &cobra.Command{
		Use:   "exec [command]",
		Short: "Execute a command with Outrig logging",
		Long: `Execute a command with Outrig logging. All arguments after exec are passed to the command.
Note: The --dev flag must come before the exec command to be recognized as an Outrig flag.
Example: outrig --dev exec ls -latrh`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filteredArgs, isDev := processDevFlag(args)

			if os.Getenv(base.AppRunIdEnvName) == "" {
				appRunId := uuid.New().String()
				os.Setenv(base.AppRunIdEnvName, appRunId)
			}

			os.Setenv(base.ExternalLogCaptureEnvName, "1")

			return execlogwrap.ExecCommand(filteredArgs, isDev)
		},
		// Disable flag parsing for this command so all flags are passed to the executed command
		DisableFlagParsing: true,
	}

	postinstallCmd := &cobra.Command{
		Use:   "postinstall",
		Short: "Display post-installation information",
		Long:  `Display welcome message and helpful information after installation.`,
		Run:   runPostinstall,
	}

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(captureLogsCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(postinstallCmd)

	// Add dev flag to root command (will be inherited by all subcommands)
	rootCmd.PersistentFlags().Bool("dev", false, "Run in development mode")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
