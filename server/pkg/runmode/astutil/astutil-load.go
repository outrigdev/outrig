// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// BuildArgs contains the build configuration for loading Go files
type BuildArgs struct {
	GoFiles     []string
	BuildFlags  []string
	ProgramArgs []string
	WorkingDir  string
	Verbose     bool
}

// TransformState contains the state for AST transformations including FileSet and packages
type TransformState struct {
	FileSet       *token.FileSet
	PackageMap    map[string]*packages.Package
	Packages      []*packages.Package
	OverlayMap    map[string]string
	ModifiedFiles map[string]*ast.File
	GoModPath     string
	TempDir       string
	Verbose       bool
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
		Mode:       packages.LoadSyntax,
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

	// Create package map
	packageMap := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		key := pkg.PkgPath
		if pkg.Name == "main" {
			key = "main"
		}
		packageMap[key] = pkg
	}

	return &TransformState{
		FileSet:    fileSet,
		PackageMap: packageMap,
		Packages:   pkgs,
	}, nil
}

// GetFilePath returns the file path for the given AST file using the FileSet
func (ts *TransformState) GetFilePath(astFile *ast.File) string {
	return ts.FileSet.Position(astFile.Pos()).Filename
}

// MarkFileModified adds the AST file to the ModifiedFiles map using its file path
func (ts *TransformState) MarkFileModified(astFile *ast.File) {
	filePath := ts.GetFilePath(astFile)
	ts.ModifiedFiles[filePath] = astFile
}
