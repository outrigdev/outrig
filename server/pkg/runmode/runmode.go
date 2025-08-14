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

	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/execlogwrap"
	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
	"github.com/outrigdev/outrig/server/pkg/runmode/gr"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// RawCmdDef holds configuration for raw command execution
type RawCmdDef struct {
	Cmd []string
	Env map[string]string
	Cwd string
	Cfg config.Config
}

// RunModeConfig holds configuration for ExecRunMode
type RunModeConfig struct {
	Args       []string
	IsVerbose  bool
	NoRun      bool
	ConfigFile string
	RawCmd     *RawCmdDef
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
	// Iterate over all packages
	totalTransformCount := 0
	for _, pkg := range transformState.Packages {
		// Apply go statement transformations to the entire package using replacements
		transformCount, filesTransformed := gr.TransformGoStatementsInPackageWithReplacement(transformState, pkg)
		if transformState.Verbose && transformCount > 0 {
			log.Printf("go-transform pkg:%q files:%d go-statements:%d\n", pkg.PkgPath, filesTransformed, transformCount)
		}
		totalTransformCount += transformCount
	}

	if transformState.Verbose {
		log.Printf("Completed go-transform across all files using replacement system (%d total packages, %d transforms)", len(transformState.Packages), totalTransformCount)
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

	// Copy go.sum if it exists
	goSumPath := filepath.Join(filepath.Dir(goModPath), "go.sum")
	if _, err := os.Stat(goSumPath); err == nil {
		tempGoSumPath := filepath.Join(tempDir, "go.sum")
		err = utilfn.CopyFile(goSumPath, tempGoSumPath)
		if err != nil {
			return fmt.Errorf("failed to copy go.sum: %w", err)
		}
	}

	return nil
}

// addGoWorkReplaceDirectives modifies the copied go.mod file to include replace directives
// that mimic the go.work file's use directives
func addGoWorkReplaceDirectives(transformState *astutil.TransformState) error {
	// Check if go.work exists
	if transformState.GoWorkPath == "" {
		return nil // No go.work file, nothing to do
	}

	if transformState.Verbose {
		log.Printf("Found go.work (%q) file, parsing for use directives", transformState.GoWorkPath)
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

	// Create temp go.mod path from transform state
	tempGoModPath := filepath.Join(transformState.TempDir, "go.mod")

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
			log.Printf("Adding replace directive (from go.work): %s => %s", targetModuleName, modulePath)
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
// Returns the extracted flag value (last one found if multiple) and the filtered arguments
func stripGoFlag(flag string, args []string) (string, []string) {
	var result []string
	var extractedValue string
	flagArg := "-" + flag
	flagPrefix := "-" + flag + "="

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == flagArg {
			// Extract the flag value from the next argument
			if i+1 < len(args) {
				extractedValue = args[i+1]
				i++ // Skip the next argument too
			}
		} else if strings.HasPrefix(arg, flagPrefix) {
			// Extract the flag value from the embedded format
			extractedValue = strings.TrimPrefix(arg, flagPrefix)
			continue
		} else {
			result = append(result, arg)
		}
	}
	return extractedValue, result
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

	// Determine working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return astutil.BuildArgs{}, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Extract -C flag value and strip it from buildFlags
	extractedWorkingDir, buildFlags := stripGoFlag("C", buildFlags)
	if extractedWorkingDir != "" {
		workingDir = extractedWorkingDir
	}

	// Convert working directory to absolute path
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return astutil.BuildArgs{}, fmt.Errorf("failed to get absolute path for working directory %s: %w", workingDir, err)
	}

	// Determine MainDir and file patterns
	mainDir, filePatterns, err := astutil.DetermineMainDirAndPatterns(absWorkingDir, goFiles)
	if err != nil {
		return astutil.BuildArgs{}, err
	}
	if cfg.IsVerbose {
		log.Printf("main-directory: %q\n", mainDir)
	}

	// Load config after we have MainDir
	loadedCfg, configSource, err := config.LoadConfig(cfg.ConfigFile, mainDir)
	if err != nil {
		return astutil.BuildArgs{}, fmt.Errorf("failed to load config (rootdir: %q): %w", mainDir, err)
	}
	var configObj config.Config
	if loadedCfg == nil {
		configObj = *config.DefaultConfig()
		configSource = "default-config"
	} else {
		configObj = *loadedCfg
	}

	if cfg.IsVerbose {
		log.Printf("config loaded from: %s\n", configSource)
	}

	// Load the specified Go files using the new astutil.LoadGoFiles function
	buildArgs := astutil.BuildArgs{
		GoFiles:      goFiles,
		BuildFlags:   buildFlags,
		ProgramArgs:  programArgs,
		WorkingDir:   absWorkingDir,
		MainDir:      mainDir,
		FilePatterns: filePatterns,
		Config:       configObj,
		Verbose:      cfg.IsVerbose,
		ConfigFile:   cfg.ConfigFile,
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

// downloadOutrigSDK runs go mod download to populate go.sum in the temp directory
func downloadOutrigSDK(transformState *astutil.TransformState, verbose bool) error {
	tempGoModPath := filepath.Join(transformState.TempDir, "go.mod")
	args := []string{"get", "-modfile", tempGoModPath, astutil.OutrigImportPath + "@" + config.OutrigSDKVersion}
	cmd := exec.Command("go", args...)

	// Set GOWORK=off to disable workspace mode and GOTOOLCHAIN for version consistency
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN="+transformState.ToolchainVersion)

	if verbose {
		log.Printf("Executing: go %v", strings.Join(args, " "))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("go get for outrig SDK failed: %w", err)
	}
	return nil
}

// determineWorkingDir determines the working directory based on config file location and ExecConfig.Cwd
func determineWorkingDir(jsonFilePath string, execConfigCwd string) (string, error) {
	// Determine working directory - use config file directory as base
	configDir := filepath.Dir(jsonFilePath)
	workingDir := configDir

	// If ExecConfig has a Cwd, use that instead (relative to config file)
	if execConfigCwd != "" {
		if filepath.IsAbs(execConfigCwd) {
			workingDir = execConfigCwd
		} else {
			workingDir = filepath.Join(configDir, execConfigCwd)
		}
	}

	// Convert working directory to absolute path
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for working directory %s: %w", workingDir, err)
	}

	return absWorkingDir, nil
}

// handleJSONConfig processes a JSON configuration file for run mode
func handleJSONConfig(jsonFilePath string, buildFlags []string, verbose bool) (astutil.BuildArgs, error) {
	// JSON mode doesn't allow build flags
	if len(buildFlags) > 0 {
		return astutil.BuildArgs{}, fmt.Errorf("build flags are not allowed when using JSON configuration file")
	}

	// Read and parse the JSON configuration file
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return astutil.BuildArgs{}, fmt.Errorf("failed to read JSON config file %s: %w", jsonFilePath, err)
	}

	var config config.Config
	err = json.Unmarshal(jsonData, &config)
	if err != nil {
		return astutil.BuildArgs{}, fmt.Errorf("failed to parse JSON config file %s: %w", jsonFilePath, err)
	}

	// Validate ExecConfig
	execConfig := config.Exec
	err = execConfig.ValidateExecConfig()
	if err != nil {
		return astutil.BuildArgs{}, err
	}

	// If it's a rawcmd, we can't handle it in normal run mode
	if execConfig.RawCmd != "" {
		return astutil.BuildArgs{}, fmt.Errorf("rawcmd execution not yet implemented in JSON config mode")
	}

	// Determine working directory
	absWorkingDir, err := determineWorkingDir(jsonFilePath, execConfig.Cwd)
	if err != nil {
		return astutil.BuildArgs{}, err
	}

	// Build the BuildArgs from ExecConfig
	buildArgs := astutil.BuildArgs{
		GoFiles:     []string{execConfig.Entry}, // Entry becomes the go file/package
		BuildFlags:  execConfig.BuildFlags,      // Use build flags from ExecConfig
		ProgramArgs: execConfig.Args,            // Args become program arguments
		WorkingDir:  absWorkingDir,
		Verbose:     verbose,
		ConfigFile:  jsonFilePath,
	}

	return buildArgs, nil
}

// setupBuildConfiguration prepares build configuration and handles JSON config files
func setupBuildConfiguration(cfg RunModeConfig) (RunModeConfig, astutil.BuildArgs, error) {
	buildArgs, err := setupBuildArgs(cfg)
	if err != nil {
		return cfg, astutil.BuildArgs{}, err
	}

	// Early return if not a JSON file
	if len(buildArgs.GoFiles) == 0 || !strings.HasSuffix(buildArgs.GoFiles[0], ".json") {
		return cfg, buildArgs, nil
	}

	jsonFilePath := buildArgs.GoFiles[0]

	// Read and parse the JSON configuration file
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return cfg, astutil.BuildArgs{}, fmt.Errorf("failed to read JSON config file %s: %w", jsonFilePath, err)
	}

	var parsedConfig config.Config
	err = json.Unmarshal(jsonData, &parsedConfig)
	if err != nil {
		return cfg, astutil.BuildArgs{}, fmt.Errorf("failed to parse JSON config file %s: %w", jsonFilePath, err)
	}

	// Validate ExecConfig
	execConfig := parsedConfig.Exec
	err = execConfig.ValidateExecConfig()
	if err != nil {
		return cfg, astutil.BuildArgs{}, err
	}

	// Update the config file path to point to the JSON file
	cfg.ConfigFile = jsonFilePath

	// If it's a rawcmd, set up RawCmdDef
	if execConfig.RawCmd != "" {
		// Determine shell to use
		shell := execConfig.RawCmdShell
		if shell == "" {
			shell = os.Getenv("SHELL")
			if shell == "" {
				shell = "/bin/sh" // fallback
			}
		}

		// Determine working directory
		absWorkingDir, err := determineWorkingDir(jsonFilePath, execConfig.Cwd)
		if err != nil {
			return cfg, astutil.BuildArgs{}, err
		}

		if cfg.IsVerbose {
			if execConfig.RawCmdShell != "" {
				log.Printf("Using custom shell: %s", shell)
			}
			log.Printf("Executing raw command: %s", execConfig.RawCmd)
			log.Printf("Working directory: %s", absWorkingDir)
		}

		// Set up RawCmdDef
		cfg.RawCmd = &RawCmdDef{
			Cmd: []string{shell, "-c", execConfig.RawCmd},
			Env: execConfig.Env,
			Cwd: absWorkingDir,
			Cfg: parsedConfig,
		}

		// Return empty BuildArgs since we're using RawCmd
		return cfg, astutil.BuildArgs{}, nil
	}

	if cfg.IsVerbose {
		log.Printf("Executing from JSON config: %s", jsonFilePath)
		log.Printf("Entry point: %s", execConfig.Entry)
	}

	// Handle normal (non-RawCmd) JSON config
	buildArgs, err = handleJSONConfig(jsonFilePath, buildArgs.BuildFlags, cfg.IsVerbose)
	if err != nil {
		return cfg, astutil.BuildArgs{}, err
	}

	return cfg, buildArgs, nil
}

// checkMonitorVersion verifies that the outrig monitor is running and compatible
func checkMonitorVersion(cfg RunModeConfig) error {
	var monitorConfig *config.Config

	if cfg.RawCmd != nil {
		monitorConfig = &cfg.RawCmd.Cfg
	} else {
		// For non-RawCmd cases, use default config for now
		monitorConfig = config.DefaultConfig()
	}

	serverVersion, _, err := comm.GetServerVersion(monitorConfig)
	if err != nil {
		return fmt.Errorf("outrig monitor is not running: %w", err)
	}

	comparison, err := utilfn.CompareSemVerCore(serverVersion, comm.MinServerVersion)
	if err != nil {
		return fmt.Errorf("invalid server version format: %s", serverVersion)
	}

	if comparison < 0 {
		return fmt.Errorf("outrig monitor version %s is incompatible (minimum required: %s)", serverVersion, comm.MinServerVersion)
	}

	return nil
}

// ExecRunMode handles the "outrig run" command with AST rewriting
func ExecRunMode(cfg RunModeConfig) error {
	cfg, buildArgs, err := setupBuildConfiguration(cfg)
	if err != nil {
		return err
	}

	// Check if monitor is running and compatible
	err = checkMonitorVersion(cfg)
	if err != nil {
		return err
	}

	if cfg.RawCmd != nil {
		if cfg.NoRun {
			log.Printf("--norun flag set, not executing command")
			return nil
		}
		return execlogwrap.ExecCommand(cfg.RawCmd.Cmd, config.GetAppRunId(), &cfg.RawCmd.Cfg, cfg.RawCmd.Env)
	} else {
		transformState, err := performASTTransformation(buildArgs, cfg)
		if err != nil {
			return err
		}
		if cfg.NoRun {
			log.Printf("--norun flag set: transforms complete, tempdir %s", transformState.TempDir)
			return nil
		}
		return runWithOverlay(transformState, buildArgs.GoFiles, buildArgs.BuildFlags, buildArgs.ProgramArgs, cfg)
	}
}

// performASTTransformation handles all AST transformation steps
func performASTTransformation(buildArgs astutil.BuildArgs, cfg RunModeConfig) (*astutil.TransformState, error) {
	transformState := loadFilesAndSetupTransformState(buildArgs, cfg)

	// Copy go.mod and go.sum to temp directory
	err := copyGoModFiles(transformState.GoModPath, transformState.TempDir, cfg.IsVerbose)
	if err != nil {
		return nil, fmt.Errorf("failed to copy go.mod files: %w", err)
	}

	// If we have a go.work file, modify the copied go.mod with replace directives
	err = addGoWorkReplaceDirectives(transformState)
	if err != nil {
		return nil, fmt.Errorf("failed to add go.work replace directives: %w", err)
	}

	// Add the version locked Outrig SDK dependency to the temp go.mod
	tempGoModPath := filepath.Join(transformState.TempDir, "go.mod")
	err = astutil.AddOutrigSDKDependency(tempGoModPath, cfg.IsVerbose, transformState.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to add outrig SDK dependency: %w", err)
	}

	// Download dependencies to populate go.sum in temp directory
	err = downloadOutrigSDK(transformState, cfg.IsVerbose)
	if err != nil {
		return nil, fmt.Errorf("failed to download dependencies: %w", err)
	}

	// Find and transform the main file using new replacement flow
	err = findAndTransformMainFileWithReplacement(transformState)
	if err != nil {
		return nil, fmt.Errorf("main file transformation failed: %w", err)
	}

	// Second pass: transform go statements in all files using replacement system
	err = transformGoStatementsInAllFilesWithReplacement(transformState)
	if err != nil {
		return nil, fmt.Errorf("go statement transformation failed: %w", err)
	}

	// Write all modified files to temp directory using new replacement system
	err = writeModifiedFilesWithReplacements(transformState)
	if err != nil {
		return nil, fmt.Errorf("failed to write modified files: %w", err)
	}

	if cfg.IsVerbose {
		log.Printf("Instrumented %d files for overlay\n", len(transformState.OverlayMap))
		if cfg.IsVerbose {
			for originalFile, tempFile := range transformState.OverlayMap {
				log.Printf("  %s -> %s", originalFile, tempFile)
			}
		}
	}

	return transformState, nil
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

	// Get the main module directory
	mainModuleDir := filepath.Dir(transformState.GoModPath)

	// Add -modfile flag to use the copied go.mod in temp directory
	tempGoModPath := filepath.Join(transformState.TempDir, "go.mod")

	// Calculate the relative path from module directory to the main package
	packagePath, err := getRelativeMainPkgDir(transformState)
	if err != nil {
		return fmt.Errorf("failed to get relative main package directory: %w", err)
	}

	// Build the go run command with -C to change to main module directory (note that -C was already stripped from otherArgs)
	goArgs := []string{"run", "-C", mainModuleDir, "-overlay", overlayFilePath, "-modfile", tempGoModPath}
	goArgs = append(goArgs, otherArgs...)
	goArgs = append(goArgs, packagePath)
	goArgs = append(goArgs, programArgs...)

	if cfg.IsVerbose {
		log.Printf("Using overlay file: %s", overlayFilePath)
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
		"GOWORK":                  "off",
		"GOTOOLCHAIN":             transformState.ToolchainVersion,
		config.FromRunModeEnvName: "1",
	}

	// Use execlogwrap to execute the command with log capture
	return execlogwrap.ExecCommand(goArgs, config.GetAppRunId(), &transformState.Config, extraEnv)
}
