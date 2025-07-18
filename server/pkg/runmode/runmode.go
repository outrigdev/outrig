// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runmode

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/server/pkg/execlogwrap"
	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
	"github.com/outrigdev/outrig/server/pkg/runmode/gr"
	"golang.org/x/tools/go/packages"
)

// RunModeConfig holds configuration for ExecRunMode
type RunModeConfig struct {
	Args      []string
	IsVerbose bool
	Config    *config.Config
}

// findAndTransformMainFile finds the main file AST and transforms it by adding outrig import and modifying main function
func findAndTransformMainFile(transformState *astutil.TransformState) error {
	// Find the main file AST
	mainFileAST, err := astutil.FindMainFileAST(transformState)
	if err != nil {
		return err
	}

	// Transform the main file: add outrig import and modify main function
	astutil.AddOutrigImport(mainFileAST)
	if !modifyMainFunction(mainFileAST) {
		mainFilePath := transformState.GetFilePath(mainFileAST)
		return fmt.Errorf("unable to find main entry point in %s. Ensure your application has a valid main()", mainFilePath)
	}

	// Mark the file as modified
	transformState.MarkFileModified(mainFileAST)

	return nil
}

// transformGoStatementsInAllFiles iterates over all files in the transform state and applies go statement transformations
func transformGoStatementsInAllFiles(transformState *astutil.TransformState) error {
	var hasTransformations bool

	// Iterate over all packages
	for _, pkg := range transformState.Packages {
		// Iterate over all AST files in each package
		for _, astFile := range pkg.Syntax {
			if astFile == nil {
				continue
			}

			// Apply go statement transformations
			if gr.TransformGoStatements(transformState, astFile) {
				// Mark the file as modified if transformations were applied
				transformState.MarkFileModified(astFile)
				hasTransformations = true

				if transformState.Verbose {
					filePath := transformState.GetFilePath(astFile)
					log.Printf("Applied go statement transformations to: %s", filePath)
				}
			}
		}
	}

	if transformState.Verbose && hasTransformations {
		log.Printf("Completed go statement transformations across all files")
	}

	return nil
}

// setupBuildArgs prepares build arguments from the config
func setupBuildArgs(cfg RunModeConfig) (astutil.BuildArgs, error) {
	// Check if user already provided -overlay flag
	for _, arg := range cfg.Args {
		if arg == "-overlay" || strings.HasPrefix(arg, "-overlay=") {
			return astutil.BuildArgs{}, fmt.Errorf("cannot use -overlay flag with 'outrig run' as it conflicts with AST rewriting")
		}
	}

	// Separate Go files from other arguments
	var goFiles []string
	var otherArgs []string
	for _, arg := range cfg.Args {
		if strings.HasSuffix(arg, ".go") && !strings.HasPrefix(arg, "-") {
			goFiles = append(goFiles, arg)
		} else {
			otherArgs = append(otherArgs, arg)
		}
	}

	// Load the specified Go files using the new astutil.LoadGoFiles function
	buildArgs := astutil.BuildArgs{
		GoFiles:    goFiles,
		BuildFlags: otherArgs,
		Verbose:    cfg.IsVerbose,
	}

	return buildArgs, nil
}

// loadFilesAndSetupTransformState loads Go files and sets up transform state
// Note: This function may call os.Exit() on errors
func loadFilesAndSetupTransformState(buildArgs astutil.BuildArgs, cfg RunModeConfig) *astutil.TransformState {
	transformState, err := astutil.LoadGoFiles(buildArgs)
	if err != nil {
		log.Printf("#outrig failed to load Go files for AST rewriting: %v", err)
		os.Exit(1)
	}

	// Check for compilation errors in the loaded packages
	if packages.PrintErrors(transformState.Packages) > 0 {
		log.Printf("#outrig cannot proceed with AST rewriting due to compilation errors")
		os.Exit(1)
	}

	if cfg.IsVerbose {
		log.Printf("Successfully loaded %d packages with FileSet", len(transformState.Packages))
	}

	// Create temporary directory for all temp files
	tempDir, err := os.MkdirTemp("", "outrig_tmp_*")
	if err != nil {
		log.Printf("#outrig failed to create temporary directory: %v", err)
		os.Exit(1)
	}
	if cfg.IsVerbose {
		log.Printf("Using temp directory: %s", tempDir)
	}

	// Initialize overlay map, modified files map, temp dir, and verbose flag in transform state
	transformState.OverlayMap = make(map[string]string)
	transformState.ModifiedFiles = make(map[string]*ast.File)
	transformState.TempDir = tempDir
	transformState.Verbose = cfg.IsVerbose

	return transformState
}

// ExecRunMode handles the "outrig run" command with AST rewriting
func ExecRunMode(cfg RunModeConfig) error {
	buildArgs, err := setupBuildArgs(cfg)
	if err != nil {
		return err
	}
	transformState := loadFilesAndSetupTransformState(buildArgs, cfg)

	// Find and transform the main file
	err = findAndTransformMainFile(transformState)
	if err != nil {
		return fmt.Errorf("main file transformation failed: %w", err)
	}

	// Second pass: transform go statements in all files
	err = transformGoStatementsInAllFiles(transformState)
	if err != nil {
		return fmt.Errorf("go statement transformation failed: %w", err)
	}

	// Write all modified files to temp directory
	err = astutil.WriteModifiedFiles(transformState)
	if err != nil {
		return fmt.Errorf("failed to write modified files: %w", err)
	}

	if cfg.IsVerbose {
		log.Printf("Created %d temporary files for overlay", len(transformState.OverlayMap))
		for originalFile, tempFile := range transformState.OverlayMap {
			log.Printf("  %s -> %s", originalFile, tempFile)
		}
	}

	// Create overlay file mapping and run
	return runWithOverlay(transformState.OverlayMap, transformState.TempDir, buildArgs.GoFiles, buildArgs.BuildFlags, cfg)
}

// runWithOverlay creates the overlay file and runs the go command
func runWithOverlay(overlayMap map[string]string, tempDir string, goFiles []string, otherArgs []string, cfg RunModeConfig) error {
	// Create overlay file mapping
	overlayData := map[string]interface{}{
		"Replace": overlayMap,
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

	if cfg.IsVerbose {
		log.Printf("Using overlay file: %s", overlayFilePath)
		log.Printf("Overlay content: %s", string(overlayBytes))
	}

	// Build the go run command with overlay
	goArgs := []string{"run", "-overlay", overlayFilePath}
	goArgs = append(goArgs, otherArgs...)
	goArgs = append(goArgs, goFiles...)

	if cfg.IsVerbose {
		log.Printf("Executing go command with args: %v", goArgs)
	}

	return runGoCommand(goArgs, cfg)
}

// runGoCommand executes a go command with the given arguments using execlogwrap
// for log capture and exits with the same exit code as the go command
func runGoCommand(args []string, cfg RunModeConfig) error {
	// Prepare the full command arguments
	goArgs := append([]string{"go"}, args...)

	// Use execlogwrap to execute the command with log capture
	return execlogwrap.ExecCommand(goArgs, config.GetAppRunId(), cfg.Config)
}
