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
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// TransformState contains the state for AST transformations including FileSet and packages
type TransformState struct {
	FileSet           *token.FileSet
	PackageMap        map[string]*packages.Package
	Packages          []*packages.Package
	MainPkg           *packages.Package // never nil - always contains the main package
	OverlayMap        map[string]string
	ModifiedFiles     map[string]*ModifiedFile
	GoModPath         string // absolute path to go.mod file
	GoWorkPath        string // absolute path to go.work file (empty if not found)
	ToolchainVersion  string // Go toolchain version from "go env GOVERSION"
	TempDir           string
	Verbose           bool
}

// BuildArgs contains the build configuration for loading Go files
type BuildArgs struct {
	GoFiles     []string
	BuildFlags  []string
	ProgramArgs []string
	WorkingDir  string
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

type ModifiedFile struct {
	FileAST           *ast.File
	Replacements      []Replacement
	RawBytes          []byte
	Modified          bool
	OutrigImportAdded bool
}

// AddInsert adds an insert replacement at the specified position
func (mf *ModifiedFile) AddInsert(pos int64, text string) {
	replacement := Replacement{
		Mode:     ReplacementModeInsert,
		StartPos: pos,
		NewText:  []byte(text),
	}
	mf.Replacements = append(mf.Replacements, replacement)
}

// AddLineDirective adds a line directive replacement at the specified position
func (mf *ModifiedFile) AddLineDirective(pos int64, fileName string, lineNum int) {
	lineDirective := MakeLineDirective(fileName, lineNum)
	replacement := Replacement{
		Mode:     ReplacementModeInsert,
		StartPos: pos,
		NewText:  []byte(lineDirective),
	}
	mf.Replacements = append(mf.Replacements, replacement)
}

// BackupToLineStart backs up from the given position to find the start of the line.
// Returns either position 0 (start of file) or the position right after a newline character.
func (mf *ModifiedFile) BackupToLineStart(pos int64) int64 {
	if pos <= 0 {
		return 0
	}

	// Search backwards from pos-1 to find a newline
	for i := pos - 1; i >= 0; i-- {
		if mf.RawBytes[i] == '\n' {
			// Return position right after the newline
			return i + 1
		}
	}

	// If no newline found, return start of file
	return 0
}

// StatementBoundary represents the result of finding a statement boundary
type StatementBoundary struct {
	AdvanceBytes    int64 // Number of bytes to advance past the boundary
	NewlineCount    int   // Number of newlines found in the boundary
	EndsWithNewline bool  // Whether the boundary ends with a newline
	BoundaryFound   bool  // Whether any boundary was found
}

// findStatementBoundary looks for statement boundaries in the given bytes starting from offset 0
// Returns information about the boundary found (whitespace + optional block comments + terminators)
// Empty data is treated as end-of-file boundary
func findStatementBoundary(data []byte) StatementBoundary {
	// Look for whitespace + optional block comments + terminators (including end-of-input)
	// (?s) makes . match newlines, (?m) enables multiline mode for ^/$
	stmtBoundaryRegex := regexp.MustCompile(`(?ms)^([ \t]*(?:/\*.*?\*/)*[ \t]*(?:\r?\n|;|//.*?\r?\n|$))`)

	if match := stmtBoundaryRegex.Find(data); match != nil {
		matchStr := string(match)
		newlineCount := strings.Count(matchStr, "\n")
		endsWithNewline := strings.HasSuffix(matchStr, "\n")
		return StatementBoundary{
			AdvanceBytes:    int64(len(match)),
			NewlineCount:    newlineCount,
			EndsWithNewline: endsWithNewline,
			BoundaryFound:   true,
		}
	}

	return StatementBoundary{AdvanceBytes: 0, NewlineCount: 0, EndsWithNewline: false, BoundaryFound: false}
}

// AddInsertStmt adds an insert replacement at the specified position with automatic line directive handling.
// It checks the raw bytes to see if there's a newline after the position and adjusts accordingly.
// The token.Position provides enough information to add the "//line" directive and handle line numbering.
func (mf *ModifiedFile) AddInsertStmt(pos token.Position, text string) {
	offset := int64(pos.Offset)
	insertOffset := offset
	var insertText string

	// Ensure text ends with newline for line directive to work properly
	if !strings.HasSuffix(text, "\n") && !strings.HasSuffix(text, "\r\n") {
		text = text + "\n"
	}

	// Find statement boundary
	remainingBytes := mf.RawBytes[offset:]
	boundary := findStatementBoundary(remainingBytes)

	// Always advance past the boundary (or stay at current position if no boundary)
	insertOffset = offset + boundary.AdvanceBytes

	// Calculate line number based on newlines in boundary
	nextLineNum := pos.Line + boundary.NewlineCount

	// Prepend newline if no boundary found or boundary doesn't end with newline
	if !boundary.BoundaryFound || !boundary.EndsWithNewline {
		insertText = "\n" + text
	} else {
		insertText = text
	}

	// Add the insert replacement
	replacement := Replacement{
		Mode:     ReplacementModeInsert,
		StartPos: insertOffset,
		NewText:  []byte(insertText),
	}
	mf.Replacements = append(mf.Replacements, replacement)

	// Add line directive to preserve line numbering
	mf.AddLineDirective(insertOffset, pos.Filename, nextLineNum)
}

// addPackageToMap adds a package to the packageMap if not already present
// and processes its imports to add them as well
func addPackageToMap(pkg *packages.Package, packageMap map[string]*packages.Package, visited map[string]bool, transformMods map[string]bool) {
	if pkg == nil {
		return
	}

	key := pkg.PkgPath
	if pkg.Module != nil && pkg.Module.Main {
		key = "main"
	}

	// Skip if already visited
	if visited[key] {
		return
	}
	visited[key] = true

	if pkg.Module == nil || !transformMods[pkg.Module.Dir] {
		return
	}

	// Add to map if not already present
	if _, exists := packageMap[key]; !exists {
		packageMap[key] = pkg
	}

	// Process imports
	for _, importedPkg := range pkg.Imports {
		addPackageToMap(importedPkg, packageMap, visited, transformMods)
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
func LoadGoFiles(buildArgs BuildArgs) (*TransformState, error) {
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
	}

	// Set working directory if provided
	if buildArgs.WorkingDir != "" {
		pkgConfig.Dir = buildArgs.WorkingDir
	}

	// Prepare file patterns with "file=" prefix
	var filePatterns []string
	for _, goFile := range buildArgs.GoFiles {
		if strings.HasSuffix(goFile, ".go") {
			filePatterns = append(filePatterns, "file="+goFile)
		} else {
			filePatterns = append(filePatterns, goFile)
		}
	}

	// Load packages using the file patterns
	pkgs, err := packages.Load(pkgConfig, filePatterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	// Create package map and populate with all packages recursively
	packageMap := make(map[string]*packages.Package)
	visited := make(map[string]bool)
	transformMods := make(map[string]bool)

	// First, populate transformMods with the main module and find main package
	var mainPkg *packages.Package
	for _, pkg := range pkgs {
		if pkg.Module != nil && pkg.Module.Main {
			transformMods[pkg.Module.Dir] = true
			mainPkg = pkg
		}
	}

	// Process each package and its imports
	for _, pkg := range pkgs {
		addPackageToMap(pkg, packageMap, visited, transformMods)
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
	}, nil
}

// GetFilePath returns the file path for the given AST file using the FileSet
func (ts *TransformState) GetFilePath(astFile *ast.File) string {
	return ts.FileSet.Position(astFile.Pos()).Filename
}

func MakeModifiedFile(state *TransformState, fileAST *ast.File) (*ModifiedFile, error) {
	// Get the original file path from the first position in the file
	position := state.FileSet.Position(fileAST.Pos())
	originalFilePath := position.Filename

	// Read the original file content
	rawBytes, err := os.ReadFile(originalFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", originalFilePath, err)
	}

	mf := &ModifiedFile{
		FileAST:      fileAST,
		Replacements: []Replacement{},
		RawBytes:     rawBytes,
	}

	// Add line directive before the package declaration
	packagePos := state.FileSet.Position(fileAST.Name.Pos())
	lineStartPos := mf.BackupToLineStart(int64(packagePos.Offset))
	mf.AddLineDirective(lineStartPos, originalFilePath, packagePos.Line)

	return mf, nil
}

// WriteModifiedFile applies the replacements to the original file content,
// generates a temporary filename, and writes the modified content to the temp file.
// Returns the path to the written temporary file.
func WriteModifiedFile(state *TransformState, modifiedFile *ModifiedFile) (string, error) {
	// Get the original file path
	originalFilePath := state.GetFilePath(modifiedFile.FileAST)

	// Read the original file content
	originalContent, err := os.ReadFile(originalFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read original file %s: %w", originalFilePath, err)
	}

	// Apply the replacements to get the modified content
	modifiedContent := ApplyReplacements(originalContent, modifiedFile.Replacements)

	// Generate a temporary filename
	tempFileName := GenerateTempFileName(originalFilePath)
	tempFilePath := filepath.Join(state.TempDir, tempFileName)

	// Write the modified content to the temporary file
	err = os.WriteFile(tempFilePath, modifiedContent, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write modified content to temp file %s: %w", tempFilePath, err)
	}

	return tempFilePath, nil
}
