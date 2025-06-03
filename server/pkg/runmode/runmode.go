// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runmode

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config holds configuration for ExecRunMode
type Config struct {
	Args      []string
	IsDev     bool
	IsVerbose bool
}

// ExecRunMode handles the "outrig run" command with AST rewriting
func ExecRunMode(config Config) error {
	// Set dev environment variable if needed
	if config.IsDev {
		if config.IsVerbose {
			log.Println("Running in development mode, setting OUTRIG_DEVCONFIG")
		}
		os.Setenv("OUTRIG_DEVCONFIG", "1")
	}

	// Check if user already provided -overlay flag
	for _, arg := range config.Args {
		if arg == "-overlay" || strings.HasPrefix(arg, "-overlay=") {
			return fmt.Errorf("cannot use -overlay flag with 'outrig run' as it conflicts with AST rewriting")
		}
	}

	// Separate Go files from other arguments
	var goFiles []string
	var otherArgs []string

	for _, arg := range config.Args {
		if strings.HasSuffix(arg, ".go") && !strings.HasPrefix(arg, "-") {
			goFiles = append(goFiles, arg)
		} else {
			otherArgs = append(otherArgs, arg)
		}
	}

	if len(goFiles) == 0 {
		if config.IsVerbose {
			log.Printf("No .go files found, falling back to standard go run")
		}
		goArgs := append([]string{"run"}, config.Args...)
		return runGoCommand(goArgs)
	}

	// Find which file contains the main() function
	mainGoFile, err := FindMainFile(goFiles)
	if err != nil {
		if config.IsVerbose {
			log.Printf("Could not find main function, falling back to standard go run: %v", err)
		}
		goArgs := append([]string{"run"}, config.Args...)
		return runGoCommand(goArgs)
	}

	// Convert to absolute path for safety
	absPath, err := filepath.Abs(mainGoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", mainGoFile, err)
	}

	// Create temporary directory for all temp files
	tempDir, err := os.MkdirTemp("", "outrig_tmp_*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	if config.IsVerbose {
		log.Printf("Using temp directory: %s", tempDir)
	}

	// Perform AST rewriting
	tempFile, err := RewriteAndCreateTempFile(absPath, tempDir)
	if err != nil {
		return fmt.Errorf("AST rewrite failed: %w", err)
	}

	// Create overlay file mapping
	overlayData := map[string]interface{}{
		"Replace": map[string]string{
			mainGoFile: tempFile,
		},
	}

	overlayBytes, err := json.Marshal(overlayData)
	if err != nil {
		return fmt.Errorf("failed to create overlay JSON: %w", err)
	}

	// Create overlay file in the same temp directory
	overlayFilePath := filepath.Join(tempDir, "overlay.json")
	err = os.WriteFile(overlayFilePath, overlayBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write overlay file: %w", err)
	}

	if config.IsVerbose {
		log.Printf("Using overlay file: %s", overlayFilePath)
		log.Printf("Overlay content: %s", string(overlayBytes))
	}

	// Build the go run command with overlay
	goArgs := []string{"run", "-overlay", overlayFilePath}
	goArgs = append(goArgs, goFiles...)
	goArgs = append(goArgs, otherArgs...)

	if config.IsVerbose {
		log.Printf("Executing go command with args: %v", goArgs)
	}

	return runGoCommand(goArgs)
}

// runGoCommand executes a go command with the given arguments
// and exits with the same exit code as the go command
func runGoCommand(args []string) error {
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return err
	}
	return nil
}
