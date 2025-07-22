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
	NoRun     bool
	Config    *config.Config
}

// findAndTransformMainFileWithReplacement finds the main file AST and adds replacements for outrig import and main function modification
func findAndTransformMainFileWithReplacement(transformState *astutil.TransformState) error {
	// Find the main file AST
	mainFileAST, err := astutil.FindMainFileAST(transformState)
	if err != nil {
		return err
	}

	// Create a ModifiedFile for the main file
	mainFilePath := transformState.GetFilePath(mainFileAST)
	modifiedFile, err := astutil.MakeModifiedFile(transformState, mainFileAST)
	if err != nil {
		return fmt.Errorf("failed to create modified file for %s: %w", mainFilePath, err)
	}

	// Add outrig import replacement
	err = astutil.AddOutrigImportReplacement(transformState, modifiedFile)
	if err != nil {
		return fmt.Errorf("failed to add outrig import replacement: %w", err)
	}

	// Add main function modification replacement
	if !modifyMainFunctionWithReplacement(transformState, modifiedFile) {
		return fmt.Errorf("unable to find main entry point in %s. Ensure your application has a valid main()", mainFilePath)
	}

	// Store the modified file in the transform state
	transformState.ModifiedFiles[mainFilePath] = modifiedFile

	return nil
}

// findAndTransformMainFile finds the main file AST and transforms it by adding outrig import and modifying main function
func findAndTransformMainFile(transformState *astutil.TransformState) error {
	// Find the main file AST
	mainFileAST, err := astutil.FindMainFileAST(transformState)
	if err != nil {
		return err
	}

	// Transform the main file: add outrig import and modify main function
	astutil.AddOutrigImport(transformState.FileSet, mainFileAST)
	if !modifyMainFunction(mainFileAST) {
		mainFilePath := transformState.GetFilePath(mainFileAST)
		return fmt.Errorf("unable to find main entry point in %s. Ensure your application has a valid main()", mainFilePath)
	}

	// Mark the file as modified
	transformState.MarkFileModified(mainFileAST)

	return nil
}

// writeModifiedFilesWithReplacements writes all modified files using the new replacement system
func writeModifiedFilesWithReplacements(transformState *astutil.TransformState) error {
	// Write all modified files to temp directory using the replacement system
	for originalPath, modifiedFile := range transformState.ModifiedFiles {
		tempFilePath, err := astutil.WriteModifiedFile(transformState, modifiedFile)
		if err != nil {
			return fmt.Errorf("failed to write modified file %s: %w", originalPath, err)
		}

		transformState.OverlayMap[originalPath] = tempFilePath
	}

	return nil
}

// transformGoStatementsInAllFiles iterates over all packages in the transform state and applies go statement transformations
func transformGoStatementsInAllFiles(transformState *astutil.TransformState) error {
	var hasTransformations bool

	// Iterate over all packages
	for _, pkg := range transformState.Packages {
		// Apply go statement transformations to the entire package
		if gr.TransformGoStatementsInPackage(transformState, pkg) {
			hasTransformations = true
		}
	}

	if transformState.Verbose && hasTransformations {
		log.Printf("Completed go statement transformations across all files")
	}

	return nil
}

var flagsWithArgs = map[string]bool{
	"-C":             true,
	"-p":             true,
	"-covermode":     true,
	"-coverpkg":      true,
	"-asmflags":      true,
	"-buildmode":     true,
	"-buildvcs":      true,
	"-compiler":      true,
	"-gccgoflags":    true,
	"-gcflags":       true,
	"-installsuffix": true,
	"-ldflags":       true,
	"-mod":           true,
	"-modfile":       true,
	"-overlay":       true,
	"-pgo":           true,
	"-pkgdir":        true,
	"-tags":          true,
	"-toolexec":      true,
	"-o":             true,
}

// setupBuildArgs prepares build arguments from the config
func setupBuildArgs(cfg RunModeConfig) (astutil.BuildArgs, error) {
	// Check if user already provided -overlay flag
	for _, arg := range cfg.Args {
		if arg == "-overlay" || strings.HasPrefix(arg, "-overlay=") {
			return astutil.BuildArgs{}, fmt.Errorf("cannot use -overlay flag with 'outrig run' as it conflicts with AST rewriting")
		}
	}

	// Parse arguments in three phases: flags, go files/package, program args
	var goFiles []string
	var programArgs []string
	var buildFlags []string

	const (
		parseFlags = iota
		parseGoFiles
		parseProgArgs
	)

	state := parseFlags

	for i := 0; i < len(cfg.Args); i++ {
		arg := cfg.Args[i]

		switch state {
		case parseFlags:
			if strings.HasPrefix(arg, "-") {
				// This is a flag
				buildFlags = append(buildFlags, arg)

				// Check if this flag takes an argument
				flagName := arg
				if strings.Contains(arg, "=") {
					// Flag is in -flag=value format, no need to consume next arg
					continue
				}

				if flagsWithArgs[flagName] && i+1 < len(cfg.Args) {
					// This flag takes an argument, consume the next argument too
					i++
					buildFlags = append(buildFlags, cfg.Args[i])
				}
			} else {
				// Hit first non-flag, switch to parsing go files/package
				state = parseGoFiles
				i-- // Re-process this argument in the new state
			}

		case parseGoFiles:
			if strings.HasSuffix(arg, ".go") {
				// This is a .go file, collect it
				goFiles = append(goFiles, arg)
			} else {
				// This is either a package or the first program arg
				if len(goFiles) == 0 {
					// No .go files seen yet, this must be a package
					goFiles = append(goFiles, arg)
				} else {
					// We already have .go files, this is a program arg
					state = parseProgArgs
					i-- // Re-process this argument in the new state
				}
			}

		case parseProgArgs:
			// Everything from here on is a program argument
			programArgs = append(programArgs, arg)
		}
	}

	// Load the specified Go files using the new astutil.LoadGoFiles function
	buildArgs := astutil.BuildArgs{
		GoFiles:     goFiles,
		BuildFlags:  buildFlags,
		ProgramArgs: programArgs,
		Verbose:     cfg.IsVerbose,
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
	transformState.ModifiedFiles = make(map[string]*astutil.ModifiedFile)
	transformState.OldModifiedFiles = make(map[string]*ast.File)
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

	// Find and transform the main file using new replacement flow
	err = findAndTransformMainFileWithReplacement(transformState)
	if err != nil {
		return fmt.Errorf("main file transformation failed: %w", err)
	}

	// Second pass: transform go statements in all files (commented out for now)
	// err = transformGoStatementsInAllFiles(transformState)
	// if err != nil {
	// 	return fmt.Errorf("go statement transformation failed: %w", err)
	// }

	// Write all modified files to temp directory using new replacement system
	err = writeModifiedFilesWithReplacements(transformState)
	if err != nil {
		return fmt.Errorf("failed to write modified files: %w", err)
	}

	if cfg.IsVerbose || cfg.NoRun {
		log.Printf("Created %d temporary files for overlay", len(transformState.OverlayMap))
		for originalFile, tempFile := range transformState.OverlayMap {
			log.Printf("  %s -> %s", originalFile, tempFile)
		}
	}

	// If NoRun is set, exit early after transforms are complete
	if cfg.NoRun {
		log.Printf("--norun flag set: transforms completed, temporary files written to %s", transformState.TempDir)
		return nil
	}

	// Create overlay file mapping and run
	return runWithOverlay(transformState.OverlayMap, transformState.TempDir, buildArgs.GoFiles, buildArgs.BuildFlags, buildArgs.ProgramArgs, cfg)
}

// runWithOverlay creates the overlay file and runs the go command
func runWithOverlay(overlayMap map[string]string, tempDir string, goFiles []string, otherArgs []string, programArgs []string, cfg RunModeConfig) error {
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
	goArgs = append(goArgs, programArgs...)

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
