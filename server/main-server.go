// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/server/pkg/boot"
	"github.com/outrigdev/outrig/server/pkg/execlogwrap"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/spf13/cobra"
)

var OutrigVersion = "v0.0.0"

var OutrigBuildTime = ""

func runCaptureLogs(cmd *cobra.Command, args []string) error {
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

func main() {
	// Set serverbase version from main version (which gets overridden by build tags)
	serverbase.OutrigVersion = OutrigVersion
	serverbase.OutrigBuildTime = OutrigBuildTime

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
			return boot.RunServer()
		},
	}

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

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(captureLogsCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(runCmd)

	// Add dev flag to root command (will be inherited by all subcommands)
	rootCmd.PersistentFlags().Bool("dev", false, "Run in development mode")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
