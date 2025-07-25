// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/outrigdev/outrig/pkg/config"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// shouldTransformPackage checks if a package should be transformed based on the transform patterns
func shouldTransformPackage(pkgPath string, transformPkgs []string) bool {
	if len(transformPkgs) == 0 {
		return false
	}

	// Check excludes first
	for _, pattern := range transformPkgs {
		if !strings.HasPrefix(pattern, "!") {
			continue
		}
		exclude := pattern[1:] // Remove the "!" prefix
		// Handle patterns ending with /** to also match the base path
		if strings.HasSuffix(exclude, "/**") {
			basePath := exclude[:len(exclude)-3] // Remove "/**"
			if pkgPath == basePath {
				return false
			}
		}
		matched, err := doublestar.Match(exclude, pkgPath)
		if err == nil && matched {
			return false
		}
	}

	// Check includes
	for _, pattern := range transformPkgs {
		if strings.HasPrefix(pattern, "!") {
			continue
		}
		// Handle patterns ending with /** to also match the base path
		if strings.HasSuffix(pattern, "/**") {
			basePath := pattern[:len(pattern)-3] // Remove "/**"
			if pkgPath == basePath {
				return true
			}
		}
		matched, err := doublestar.Match(pattern, pkgPath)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// TransformState contains the state for AST transformations including FileSet and packages
type TransformState struct {
	FileSet          *token.FileSet
	PackageMap       map[string]*packages.Package
	Packages         []*packages.Package
	MainPkg          *packages.Package // never nil - always contains the main package
	OverlayMap       map[string]string
	ModifiedFiles    map[string]*ModifiedFile
	GoModPath        string // absolute path to go.mod file
	GoWorkPath       string // absolute path to go.work file (empty if not found)
	ToolchainVersion string // Go toolchain version from "go env GOVERSION"
	MainDir          string // absolute path to main directory
	TempDir          string
	Verbose          bool
}

// BuildArgs contains the build configuration for loading Go files
type BuildArgs struct {
	GoFiles     []string
	BuildFlags  []string
	ProgramArgs []string
	WorkingDir  string // will always be set (will not be empty)
	Verbose     bool
}

// ParseGoWorkFile parses a go.work file and returns the absolute paths of modules listed in the use directive
func ParseGoWorkFile(goWorkPath string) ([]string, error) {
	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return nil, err
	}

	workFile, err := modfile.ParseWork(goWorkPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.work file: %w", err)
	}

	// Get project root from go.work path
	goWorkRoot := filepath.Dir(goWorkPath)

	var modules []string
	for _, use := range workFile.Use {
		module := strings.Trim(use.Path, `"`)

		// Resolve the module path relative to go.work
		var modulePath string
		if !filepath.IsAbs(module) {
			modulePath = filepath.Join(goWorkRoot, module)
		} else {
			modulePath = module
		}

		// Convert to absolute path
		absModulePath, err := filepath.Abs(modulePath)
		if err != nil {
			continue
		}

		modules = append(modules, absModulePath)
	}

	return modules, nil
}

// GetModuleName reads a go.mod file and returns the module name
func GetModuleName(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod file %s: %w", goModPath, err)
	}

	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse go.mod file %s: %w", goModPath, err)
	}

	if modFile.Module == nil {
		return "", fmt.Errorf("no module declaration found in %s", goModPath)
	}

	return modFile.Module.Mod.Path, nil
}

// IsModuleInWorkspace checks if the current module is listed in the go.work file
func IsModuleInWorkspace(transformState *TransformState) (bool, error) {
	if transformState.GoWorkPath == "" {
		return false, nil // No go.work file
	}

	// Parse go.work file to get absolute module paths
	modules, err := ParseGoWorkFile(transformState.GoWorkPath)
	if err != nil {
		return false, err
	}

	// Get current module root (GoModPath is already absolute)
	currentModuleRoot := filepath.Dir(transformState.GoModPath)

	// Check if our current module is listed in go.work
	for _, modulePath := range modules {
		if modulePath == currentModuleRoot {
			return true, nil
		}
	}

	return false, nil
}

// addPackageToMap adds a package to the packageMap if not already present
// and processes its imports to add them as well
func addPackageToMap(pkg *packages.Package, packageMap map[string]*packages.Package, visited map[string]bool, transformPkgs []string) {
	if pkg == nil || pkg.Module == nil {
		return
	}

	key := pkg.ID
	if visited[key] {
		return
	}
	visited[key] = true

	// Check if we should transform this package
	if !shouldTransformPackage(pkg.Module.Path, transformPkgs) {
		return
	}

	// Add to map if not already present
	if _, exists := packageMap[key]; !exists {
		packageMap[key] = pkg
	}

	// Process imports
	for _, importedPkg := range pkg.Imports {
		addPackageToMap(importedPkg, packageMap, visited, transformPkgs)
	}
}

// findGoWorkPath searches for a go.work file starting from the given directory
// and working up through parent directories. It also handles the GOWORK environment variable.
// If GOWORK is set to "off", it returns an empty string.
// Returns an absolute path to the go.work file if found.
func findGoWorkPath(startDir string) (string, error) {
	// Check GOWORK environment variable first
	gowork := os.Getenv("GOWORK")
	if gowork == "off" {
		return "", nil
	}
	if gowork != "" {
		// If GOWORK is set to a specific path, use it
		if filepath.IsAbs(gowork) {
			return gowork, nil
		}
		// If relative path, resolve it relative to current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}
		goWorkPath := filepath.Join(cwd, gowork)
		// Make it absolute
		absPath, err := filepath.Abs(goWorkPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path for %s: %w", goWorkPath, err)
		}
		return absPath, nil
	}

	// Start from the given directory and work up
	currentDir := startDir
	for {
		goWorkPath := filepath.Join(currentDir, "go.work")
		if _, err := os.Stat(goWorkPath); err == nil {
			// Make it absolute
			absPath, err := filepath.Abs(goWorkPath)
			if err != nil {
				return "", fmt.Errorf("failed to get absolute path for %s: %w", goWorkPath, err)
			}
			return absPath, nil
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached the root directory
			break
		}
		currentDir = parentDir
	}

	return "", nil
}

// DetectToolchainVersion runs "go env GOVERSION" to get the Go toolchain version
func DetectToolchainVersion(pkgDir string) (string, error) {
	cmd := exec.Command("go", "env", "GOVERSION")
	cmd.Dir = pkgDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Go version: %w", err)
	}

	version := strings.TrimSpace(string(output))
	return version, nil
}

// LoadGoFiles loads the specified Go files using packages.Load with "file=" prefix
// and returns a TransformState containing the FileSet and package information
func LoadGoFiles(buildArgs BuildArgs, cfg config.Config) (*TransformState, error) {
	if len(buildArgs.GoFiles) == 0 {
		return nil, fmt.Errorf("no Go files provided")
	}

	// Create our own FileSet
	fileSet := token.NewFileSet()

	// Configure packages.Config
	pkgConfig := &packages.Config{
		Mode:       packages.LoadSyntax | packages.NeedImports | packages.NeedDeps | packages.NeedModule,
		Fset:       fileSet,
		BuildFlags: buildArgs.BuildFlags,
		Env:        os.Environ(),
		Dir:        buildArgs.WorkingDir,
	}

	// Set working directory if provided
	if buildArgs.WorkingDir != "" {
		pkgConfig.Dir = buildArgs.WorkingDir
	}

	// Prepare file patterns with "file=" prefix and determine MainDir
	var filePatterns []string
	var mainDir string
	for _, goFile := range buildArgs.GoFiles {
		if strings.HasSuffix(goFile, ".go") {
			// Validate .go file exists
			filePath := filepath.Join(buildArgs.WorkingDir, goFile)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return nil, fmt.Errorf("Go file does not exist: %s", filePath)
			}

			filePatterns = append(filePatterns, "file="+goFile)
			if mainDir == "" {
				mainDir = filepath.Dir(goFile)
			}
		} else {
			// Validate directory exists
			dirPath := filepath.Join(buildArgs.WorkingDir, goFile)
			if stat, err := os.Stat(dirPath); os.IsNotExist(err) {
				return nil, fmt.Errorf("directory does not exist: %s", dirPath)
			} else if err != nil {
				return nil, fmt.Errorf("error checking directory %s: %w", dirPath, err)
			} else if !stat.IsDir() {
				return nil, fmt.Errorf("path is not a directory: %s (full module paths are not supported)", dirPath)
			}

			filePatterns = append(filePatterns, goFile)
			if mainDir == "" {
				mainDir = goFile
			}
		}
	}

	// Make MainDir absolute
	mainDir, err := filepath.Abs(mainDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for main directory: %w", err)
	}

	// Load packages using the file patterns
	pkgs, err := packages.Load(pkgConfig, filePatterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	// Create package map and populate with all packages recursively
	packageMap := make(map[string]*packages.Package)
	visited := make(map[string]bool)

	// Create combined transform patterns including main module and configured patterns
	transformPkgs := make([]string, len(cfg.RunMode.TransformPkgs))
	copy(transformPkgs, cfg.RunMode.TransformPkgs)
	if len(transformPkgs) >= 0 && buildArgs.Verbose {
		log.Printf("transformpkgs overrides: %v\n", transformPkgs)
	}

	// Find main package and add its module to transform patterns
	var mainPkg *packages.Package
	for _, pkg := range pkgs {
		if pkg.Name == "main" && pkg.Dir == mainDir {
			if pkg.Module != nil {
				log.Printf("transforming %q\n", pkg.Module.Path)
				transformPkgs = append(transformPkgs, pkg.Module.Path)
				transformPkgs = append(transformPkgs, pkg.Module.Path+"/**")
			}
			mainPkg = pkg
		}
	}

	// Process each package and its imports
	for _, pkg := range pkgs {
		addPackageToMap(pkg, packageMap, visited, transformPkgs)
	}

	// Convert packageMap to slice for Packages field, with main package first
	var packages []*packages.Package
	if mainPkg != nil {
		packages = append(packages, mainPkg)
	}
	for _, pkg := range packageMap {
		if pkg != mainPkg {
			packages = append(packages, pkg)
		}
	}

	// Validate that we have a main package with module information
	if mainPkg == nil {
		return nil, fmt.Errorf("no main package found")
	}
	if mainPkg.Module == nil {
		return nil, fmt.Errorf("main package has no module information")
	}
	if mainPkg.Module.GoMod == "" {
		return nil, fmt.Errorf("main package module has no go.mod file")
	}

	// Set GoModPath from the main package's module (make it absolute)
	goModPath, err := filepath.Abs(mainPkg.Module.GoMod)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for go.mod: %w", err)
	}

	// Find GoWorkPath starting from the main package's directory
	goWorkPath, err := findGoWorkPath(mainPkg.Module.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to find go.work path: %w", err)
	}

	// Detect toolchain version
	toolchainVersion, err := DetectToolchainVersion(mainPkg.Module.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect toolchain version: %w", err)
	}

	if buildArgs.Verbose {
		log.Printf("Detected Go toolchain version: %s", toolchainVersion)
	}

	return &TransformState{
		FileSet:          fileSet,
		PackageMap:       packageMap,
		Packages:         packages,
		MainPkg:          mainPkg,
		GoModPath:        goModPath,
		GoWorkPath:       goWorkPath,
		ToolchainVersion: toolchainVersion,
		MainDir:          mainDir,
	}, nil
}

// GetFilePath returns the file path for the given AST file using the FileSet
func (ts *TransformState) GetFilePath(astFile *ast.File) string {
	return ts.FileSet.Position(astFile.Pos()).Filename
}
