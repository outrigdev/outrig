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
)

type ModifiedFile struct {
	FileAST           *ast.File
	Replacements      []Replacement
	RawBytes          []byte
	Modified          bool
	OutrigImportAdded bool
}

// statementBoundary represents the result of finding a statement boundary
type statementBoundary struct {
	AdvanceBytes    int64 // Number of bytes to advance past the boundary
	NewlineCount    int   // Number of newlines found in the boundary
	EndsWithNewline bool  // Whether the boundary ends with a newline
	BoundaryFound   bool  // Whether any boundary was found
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

// findStatementBoundary looks for statement boundaries in the given bytes starting from offset 0
// Returns information about the boundary found (whitespace + optional block comments + terminators)
// Empty data is treated as end-of-file boundary
func findStatementBoundary(data []byte) statementBoundary {
	// Look for whitespace + optional block comments + terminators (including end-of-input)
	// (?s) makes . match newlines, (?m) enables multiline mode for ^/$
	stmtBoundaryRegex := regexp.MustCompile(`(?ms)^([ \t]*(?:/\*.*?\*/)*[ \t]*(?:\r?\n|;|//.*?\r?\n|$))`)

	if match := stmtBoundaryRegex.Find(data); match != nil {
		matchStr := string(match)
		newlineCount := strings.Count(matchStr, "\n")
		endsWithNewline := strings.HasSuffix(matchStr, "\n")
		return statementBoundary{
			AdvanceBytes:    int64(len(match)),
			NewlineCount:    newlineCount,
			EndsWithNewline: endsWithNewline,
			BoundaryFound:   true,
		}
	}

	return statementBoundary{AdvanceBytes: 0, NewlineCount: 0, EndsWithNewline: false, BoundaryFound: false}
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
