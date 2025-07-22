// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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

type ModifiedFile struct {
	FileAST      *ast.File
	Replacements []Replacement
	RawBytes     []byte
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

// TransformState contains the state for AST transformations including FileSet and packages
type TransformState struct {
	FileSet          *token.FileSet
	PackageMap       map[string]*packages.Package
	Packages         []*packages.Package
	OverlayMap       map[string]string
	ModifiedFiles    map[string]*ModifiedFile
	OldModifiedFiles map[string]*ast.File
	GoModPath        string
	TempDir          string
	Verbose          bool
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
	ts.OldModifiedFiles[filePath] = astFile
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

	// Create the initial line directive to mark the original file
	lineDirective := "//line " + originalFilePath + ":1\n"

	// Create the initial replacement at position 0
	initialReplacement := Replacement{
		Mode:     ReplacementModeInsert,
		StartPos: 0,
		NewText:  []byte(lineDirective),
	}

	return &ModifiedFile{
		FileAST:      fileAST,
		Replacements: []Replacement{initialReplacement},
		RawBytes:     rawBytes,
	}, nil
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
