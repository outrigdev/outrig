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
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/server/demo"
	"github.com/outrigdev/outrig/server/pkg/boot"
	"github.com/outrigdev/outrig/server/pkg/execlogwrap"
	"github.com/outrigdev/outrig/server/pkg/runmode"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig/server/pkg/tevent"
	"github.com/outrigdev/outrig/server/pkg/updatecheck"
	"github.com/spf13/cobra"
)

type specialArgs struct {
	IsDev     bool
	IsVerbose bool
	Args      []string
}

func parseSpecialArgs(keyArg string) (specialArgs, error) {
	result := specialArgs{}

	// Find the position of keyArg in os.Args
	keyArgIndex := -1
	for i, arg := range os.Args {
		if arg == keyArg {
			keyArgIndex = i
			break
		}
	}

	if keyArgIndex == -1 {
		return result, fmt.Errorf("key argument '%s' not found in command line", keyArg)
	}

	// Look for --dev and -v flags before keyArg
	for i := 1; i < keyArgIndex; i++ {
		arg := os.Args[i]
		if arg == "--dev" {
			result.IsDev = true
		} else if arg == "-v" {
			result.IsVerbose = true
		}
	}

	// Everything after keyArg goes into Args
	if keyArgIndex+1 < len(os.Args) {
		result.Args = os.Args[keyArgIndex+1:]
	}

	return result, nil
}

var (
	// these get set via -X during build
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
		{Input: stderrIn, Output: os.Stderr, Source: "/dev/stderr"},
	}
	return execlogwrap.ProcessExistingStreams(streams, isDev)
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
	fmt.Printf("To start the Outrig Monitor, run:\n%soutrig monitor%s\n\n", brightCyan, reset)

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
	// Set serverbase consts from main (which gets overridden by build tags)
	serverbase.OutrigBuildTime = OutrigBuildTime
	serverbase.OutrigCommit = OutrigCommit

	rootCmd := &cobra.Command{
		Use:   "outrig",
		Short: "Outrig provides real-time debugging for Go programs",
		Long:  `Outrig provides real-time debugging for Go programs, similar to Chrome DevTools.`,
		// No Run function for root command - it will just display help and exit
	}

	monitorCmd := &cobra.Command{
		Use:     "monitor",
		Aliases: []string{"server"},
		Short:   "Run the Outrig Monitor",
		Long:    `Run the Outrig Monitor which provides real-time debugging capabilities.`,
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
			trayPid, _ := cmd.Flags().GetInt("tray-pid")

			// Create CLI config
			config := boot.CLIConfig{
				Port:         port,
				CloseOnStdin: closeOnStdin,
				TrayAppPid:   trayPid,
			}

			return boot.RunServer(config)
		},
	}
	// Add flags to monitor command
	monitorCmd.Flags().Bool("no-telemetry", false, "Disable telemetry collection")
	monitorCmd.Flags().Bool("no-updatecheck", false, "Disable checking for updates")
	monitorCmd.Flags().Int("port", 0, "Override the default web server port (default: 5005 for production, 6005 for development)")
	monitorCmd.Flags().Bool("close-on-stdin", false, "Shut down the server when stdin is closed")
	monitorCmd.Flags().Int("tray-pid", 0, "PID of the tray application that started the server")
	// Hide this flag since it's only used internally by the tray application
	monitorCmd.Flags().MarkHidden("tray-pid")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Outrig",
		Long:  `Print the version number of Outrig and exit.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n", getVersion())
		},
	}

	captureLogsCmd := &cobra.Command{
		Use:    "capturelogs",
		Short:  "Capture logs from stdin and fd 3",
		Long:   `Capture logs from stdin (stdout of the process) and fd 3 (stderr of the process) and write them to stdout and stderr respectively.`,
		RunE:   runCaptureLogs,
		Hidden: true,
	}
	captureLogsCmd.Flags().String("source", "", "Override the source name for stdout logs (default: /dev/stdout)")
	captureLogsCmd.Flags().Bool("dev", false, "Run in development mode")

	runCmd := &cobra.Command{
		Use:   "run [go-args]",
		Short: "Run a Go program with Outrig enabled",
		Long: `Run a Go program with Outrig logging. Automatically injects outrig.Init() into the main function via AST rewriting.
The original source files are never modified - a temporary file is used for execution.
Example: outrig --dev --verbose run main.go`,
		RunE: func(cmd *cobra.Command, args []string) error {
			specialArgs, err := parseSpecialArgs("run")
			if err != nil {
				return err
			}

			if len(specialArgs.Args) == 0 {
				return fmt.Errorf("run command requires at least one argument")
			}

			config := runmode.Config{
				Args:      specialArgs.Args,
				IsDev:     specialArgs.IsDev,
				IsVerbose: specialArgs.IsVerbose,
			}
			return runmode.ExecRunMode(config)
		},
		// Disable flag parsing for this command so all flags are passed to the go command
		DisableFlagParsing: true,
	}

	execCmd := &cobra.Command{
		Use:   "exec [command]",
		Short: "Execute a command with Outrig logging",
		Long: `Execute a command with Outrig logging. All arguments after exec are passed to the command.
Example: outrig --dev exec ls -latrh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			specialArgs, err := parseSpecialArgs("exec")
			if err != nil {
				return err
			}

			if len(specialArgs.Args) == 0 {
				return fmt.Errorf("exec command requires at least one argument")
			}

			if os.Getenv(config.AppRunIdEnvName) == "" {
				appRunId := uuid.New().String()
				os.Setenv(config.AppRunIdEnvName, appRunId)
			}
	
			os.Setenv(config.ExternalLogCaptureEnvName, "1")

			return execlogwrap.ExecCommand(specialArgs.Args, specialArgs.IsDev)
		},
		// Disable flag parsing for this command so all flags are passed to the executed command
		DisableFlagParsing: true,
		Hidden:             true,
	}

	postinstallCmd := &cobra.Command{
		Use:   "postinstall",
		Short: "Display post-installation information",
		Long:  `Display welcome message and helpful information after installation.`,
		Run:   runPostinstall,
	}

	demoCmd := &cobra.Command{
		Use:   "demo",
		Short: "Run the OutrigAcres demo game",
		Long:  `Run the OutrigAcres demo game to showcase Outrig's debugging capabilities.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			devMode, _ := cmd.Flags().GetBool("dev")
			noBrowserLaunch, _ := cmd.Flags().GetBool("no-browser-launch")
			port, _ := cmd.Flags().GetInt("port")
			closeOnStdin, _ := cmd.Flags().GetBool("close-on-stdin")

			config := demo.Config{
				DevMode:         devMode,
				NoBrowserLaunch: noBrowserLaunch,
				Port:            port,
				CloseOnStdin:    closeOnStdin,
			}

			demo.RunOutrigAcres(config)
			return nil
		},
	}
	demoCmd.Flags().Bool("dev", false, "Run in development mode (serve files from disk)")
	demoCmd.Flags().Bool("no-browser-launch", false, "Don't automatically open the browser")
	demoCmd.Flags().Int("port", 0, "Override the default demo server port (default: 22005)")
	demoCmd.Flags().Bool("close-on-stdin", false, "Shut down the demo when stdin is closed")

	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(captureLogsCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(postinstallCmd)
	rootCmd.AddCommand(demoCmd)
	rootCmd.PersistentFlags().Bool("dev", false, "Run in dev mode")
	rootCmd.PersistentFlags().MarkHidden("dev")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().MarkHidden("verbose")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
