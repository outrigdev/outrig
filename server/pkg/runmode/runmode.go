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

	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/execlogwrap"
	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
	"github.com/outrigdev/outrig/server/pkg/runmode/gr"
	"golang.org/x/mod/modfile"
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

	// Mark the main file as modified since we added import and modified main function
	modifiedFile.Modified = true

	// Store the modified file in the transform state
	transformState.ModifiedFiles[mainFilePath] = modifiedFile

	return nil
}

// writeModifiedFilesWithReplacements writes all modified files using the new replacement system
func writeModifiedFilesWithReplacements(transformState *astutil.TransformState) error {
	// Write only actually modified files to temp directory using the replacement system
	for originalPath, modifiedFile := range transformState.ModifiedFiles {
		// Skip files that weren't actually modified
		if !modifiedFile.Modified {
			continue
		}

		tempFilePath, err := astutil.WriteModifiedFile(transformState, modifiedFile)
		if err != nil {
			return fmt.Errorf("failed to write modified file %s: %w", originalPath, err)
		}

		transformState.OverlayMap[originalPath] = tempFilePath
	}

	return nil
}

// transformGoStatementsInAllFilesWithReplacement iterates over all packages in the transform state and applies go statement transformations using the replacement system
func transformGoStatementsInAllFilesWithReplacement(transformState *astutil.TransformState) error {
	var hasTransformations bool

	// Iterate over all packages
	for _, pkg := range transformState.Packages {
		// Apply go statement transformations to the entire package using replacements
		if gr.TransformGoStatementsInPackageWithReplacement(transformState, pkg) {
			hasTransformations = true
		}
	}

	if transformState.Verbose && hasTransformations {
		log.Printf("Completed go statement transformations across all files using replacement system")
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

// copyGoModFiles copies go.mod and go.sum to the temp directory
func copyGoModFiles(goModPath, tempDir string, verbose bool) error {
	// Copy go.mod
	tempGoModPath := filepath.Join(tempDir, "go.mod")
	err := utilfn.CopyFile(goModPath, tempGoModPath)
	if err != nil {
		return fmt.Errorf("failed to copy go.mod: %w", err)
	}

	if verbose {
		log.Printf("Copied go.mod from %s to %s", goModPath, tempGoModPath)
	}

	// Copy go.sum if it exists
	goSumPath := filepath.Join(filepath.Dir(goModPath), "go.sum")
	if _, err := os.Stat(goSumPath); err == nil {
		tempGoSumPath := filepath.Join(tempDir, "go.sum")
		err = utilfn.CopyFile(goSumPath, tempGoSumPath)
		if err != nil {
			return fmt.Errorf("failed to copy go.sum: %w", err)
		}

		if verbose {
			log.Printf("Copied go.sum from %s to %s", goSumPath, tempGoSumPath)
		}
	}

	return nil
}

// addGoWorkReplaceDirectives modifies the copied go.mod file to include replace directives
// that mimic the go.work file's use directives
func addGoWorkReplaceDirectives(transformState *astutil.TransformState, tempGoModPath string) error {
	// Check if go.work exists
	if transformState.GoWorkPath == "" {
		return nil // No go.work file, nothing to do
	}

	if transformState.Verbose {
		log.Printf("Found go.work file, parsing for use directives")
	}

	// Parse go.work file
	modules, err := astutil.ParseGoWorkFile(transformState.GoWorkPath)
	if err != nil {
		return fmt.Errorf("failed to parse go.work file: %w", err)
	}

	if len(modules) == 0 {
		return nil // No modules to process
	}

	// Check if our current module is listed in go.work
	isCurrentModuleInWorkspace, err := astutil.IsModuleInWorkspace(transformState)
	if err != nil {
		return fmt.Errorf("failed to check if module is in workspace: %w", err)
	}

	if !isCurrentModuleInWorkspace {
		if transformState.Verbose {
			log.Printf("Current module not found in go.work, skipping replace directives")
		}
		return nil // Our module is not in the workspace, don't add replace directives
	}

	if transformState.Verbose {
		log.Printf("Current module found in go.work, adding replace directives")
	}

	// Read and parse the current go.mod file
	goModData, err := os.ReadFile(tempGoModPath)
	if err != nil {
		return fmt.Errorf("failed to read temp go.mod: %w", err)
	}

	goModFile, err := modfile.Parse(transformState.GoModPath, goModData, nil)
	if err != nil {
		return fmt.Errorf("failed to parse temp go.mod: %w", err)
	}
	currentModuleRoot := filepath.Dir(transformState.GoModPath)

	// Generate replace directives for each module in go.work
	var addedReplaces int
	for _, modulePath := range modules {
		// Skip the current module (self-reference)

		if modulePath == currentModuleRoot {
			continue
		}

		// Get the target module's name from its go.mod file
		targetGoModPath := filepath.Join(modulePath, "go.mod")
		targetModuleName, err := astutil.GetModuleName(targetGoModPath)
		if err != nil {
			if transformState.Verbose {
				log.Printf("Skipping module %s: %v", modulePath, err)
			}
			continue
		}

		// Add replace directive using modfile API
		err = goModFile.AddReplace(targetModuleName, "", modulePath, "")
		if err != nil {
			return fmt.Errorf("failed to add replace directive for %s: %w", targetModuleName, err)
		}

		addedReplaces++

		if transformState.Verbose {
			log.Printf("Adding replace directive: %s => %s", targetModuleName, modulePath)
		}
	}

	if addedReplaces == 0 {
		return nil // No replace directives to add
	}

	// Format and write the modified go.mod file
	formattedData, err := goModFile.Format()
	if err != nil {
		return fmt.Errorf("failed to format modified go.mod: %w", err)
	}

	err = os.WriteFile(tempGoModPath, formattedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write modified go.mod: %w", err)
	}

	if transformState.Verbose {
		log.Printf("Added %d replace directives to temp go.mod from go.work", addedReplaces)
	}

	return nil
}

// hasModfileFlag checks if the build flags already contain a -modfile flag
func hasModfileFlag(buildFlags []string) bool {
	for i, flag := range buildFlags {
		if flag == "-modfile" {
			return true
		}
		if strings.HasPrefix(flag, "-modfile=") {
			return true
		}
		// Check if this is a flag that takes an argument and the next arg is the modfile
		if flag == "-modfile" && i+1 < len(buildFlags) {
			return true
		}
	}
	return false
}

// stripGoFlag removes any -[flag] or -[flag]= flags from the given arguments
func stripGoFlag(flag string, args []string) []string {
	var result []string
	flagArg := "-" + flag
	flagPrefix := "-" + flag + "="

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == flagArg {
			// Skip this flag and its argument
			if i+1 < len(args) {
				i++ // Skip the next argument too
			}
		} else if strings.HasPrefix(arg, flagPrefix) {
			// Skip this flag (argument is embedded)
			continue
		} else {
			result = append(result, arg)
		}
	}
	return result
}

// getRelativeMainPkgDir calculates the relative path from the module directory to the main package directory
func getRelativeMainPkgDir(transformState *astutil.TransformState) (string, error) {
	// Get the main module directory
	mainModuleDir := filepath.Dir(transformState.GoModPath)

	// Calculate relative path from module directory to package directory
	relPath, err := filepath.Rel(mainModuleDir, transformState.MainPkg.Dir)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}
	if relPath == "." {
		return ".", nil
	}
	return "./" + relPath, nil
}

// setupBuildArgs prepares build arguments from the config
func setupBuildArgs(cfg RunModeConfig) (astutil.BuildArgs, error) {
	// Check if user already provided -overlay flag
	for _, arg := range cfg.Args {
		if arg == "-overlay" || strings.HasPrefix(arg, "-overlay=") {
			return astutil.BuildArgs{}, fmt.Errorf("cannot use -overlay flag with 'outrig run' as it conflicts with AST rewriting")
		}
	}

	// Check if user already provided -modfile flag
	for _, arg := range cfg.Args {
		if arg == "-modfile" || strings.HasPrefix(arg, "-modfile=") {
			return astutil.BuildArgs{}, fmt.Errorf("cannot use -modfile flag with 'outrig run' as it conflicts with go.mod handling")
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
	transformState.TempDir = tempDir
	transformState.Verbose = cfg.IsVerbose

	return transformState
}

// downloadDependencies runs go mod download to populate go.sum in the temp directory
func downloadDependencies(tempGoModPath string, verbose bool) error {
	if verbose {
		log.Printf("Running go mod download to populate go.sum")
	}

	// Run go mod download with -modfile flag pointing to temp go.mod
	args := []string{"get", "-modfile", tempGoModPath, "github.com/outrigdev/outrig@" + config.OutrigSDKVersion}

	cmd := exec.Command("go", args...)

	// Set GOWORK=off to disable workspace mode
	cmd.Env = append(os.Environ(), "GOWORK=off")

	if verbose {
		log.Printf("Executing: go %v", strings.Join(args, " "))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("go mod download failed: %w", err)
	}

	if verbose {
		log.Printf("Successfully downloaded dependencies and updated go.sum")
	}

	return nil
}

// ExecRunMode handles the "outrig run" command with AST rewriting
func ExecRunMode(cfg RunModeConfig) error {
	buildArgs, err := setupBuildArgs(cfg)
	if err != nil {
		return err
	}
	transformState := loadFilesAndSetupTransformState(buildArgs, cfg)

	// Copy go.mod and go.sum to temp directory
	err = copyGoModFiles(transformState.GoModPath, transformState.TempDir, cfg.IsVerbose)
	if err != nil {
		return fmt.Errorf("failed to copy go.mod files: %w", err)
	}

	// If we have a go.work file, modify the copied go.mod with replace directives
	tempGoModPath := filepath.Join(transformState.TempDir, "go.mod")
	err = addGoWorkReplaceDirectives(transformState, tempGoModPath)
	if err != nil {
		return fmt.Errorf("failed to add go.work replace directives: %w", err)
	}

	// Add the version locked Outrig SDK dependency to the temp go.mod
	err = astutil.AddOutrigSDKDependency(tempGoModPath, cfg.IsVerbose)
	if err != nil {
		return fmt.Errorf("failed to add outrig SDK dependency: %w", err)
	}

	// Download dependencies to populate go.sum in temp directory
	err = downloadDependencies(tempGoModPath, cfg.IsVerbose)
	if err != nil {
		return fmt.Errorf("failed to download dependencies: %w", err)
	}

	// Find and transform the main file using new replacement flow
	err = findAndTransformMainFileWithReplacement(transformState)
	if err != nil {
		return fmt.Errorf("main file transformation failed: %w", err)
	}

	// Second pass: transform go statements in all files using replacement system
	err = transformGoStatementsInAllFilesWithReplacement(transformState)
	if err != nil {
		return fmt.Errorf("go statement transformation failed: %w", err)
	}

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
	return runWithOverlay(transformState, buildArgs.GoFiles, buildArgs.BuildFlags, buildArgs.ProgramArgs, cfg)
}

// runWithOverlay creates the overlay file and runs the go command
func runWithOverlay(transformState *astutil.TransformState, goFiles []string, otherArgs []string, programArgs []string, cfg RunModeConfig) error {
	// Create overlay file mapping
	overlayData := map[string]interface{}{
		"Replace": transformState.OverlayMap,
	}

	overlayBytes, err := json.Marshal(overlayData)
	if err != nil {
		return fmt.Errorf("failed to create overlay JSON: %w", err)
	}

	// Create overlay file in the same temp directory
	overlayFilePath := filepath.Join(transformState.TempDir, "overlay.json")
	err = os.WriteFile(overlayFilePath, overlayBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write overlay file: %w", err)
	}

	if cfg.IsVerbose {
		log.Printf("Using overlay file: %s", overlayFilePath)
		log.Printf("Overlay content: %s", string(overlayBytes))
	}

	// Get the main module directory
	mainModuleDir := filepath.Dir(transformState.GoModPath)

	// Add -modfile flag to use the copied go.mod in temp directory
	tempGoModPath := filepath.Join(transformState.TempDir, "go.mod")

	// Strip any existing -C flags from otherArgs since we're controlling it
	otherArgs = stripGoFlag("C", otherArgs)

	// Calculate the relative path from module directory to the main package
	packagePath, err := getRelativeMainPkgDir(transformState)
	if err != nil {
		return fmt.Errorf("failed to get relative main package directory: %w", err)
	}

	// Build the go run command with -C to change to main module directory
	goArgs := []string{"run", "-C", mainModuleDir, "-overlay", overlayFilePath, "-modfile", tempGoModPath}
	goArgs = append(goArgs, otherArgs...)
	goArgs = append(goArgs, packagePath)
	goArgs = append(goArgs, programArgs...)

	if cfg.IsVerbose {
		log.Printf("Using -C flag to change to main module directory: %s", mainModuleDir)
		log.Printf("Using -modfile flag: %s", tempGoModPath)
		log.Printf("Executing go command with args: %v", append([]string{"go"}, goArgs...))
	}

	return runGoCommand(goArgs, transformState, cfg)
}

// runGoCommand executes a go command with the given arguments using execlogwrap
// for log capture and exits with the same exit code as the go command
func runGoCommand(args []string, transformState *astutil.TransformState, cfg RunModeConfig) error {
	// Prepare the full command arguments
	goArgs := append([]string{"go"}, args...)

	// Set GOWORK=off to disable workspace mode when using replace directives
	extraEnv := map[string]string{
		"GOWORK":      "off",
		"GOTOOLCHAIN": transformState.ToolchainVersion,
	}

	// Use execlogwrap to execute the command with log capture
	return execlogwrap.ExecCommand(goArgs, config.GetAppRunId(), cfg.Config, extraEnv)
}
