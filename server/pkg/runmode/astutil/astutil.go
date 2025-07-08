// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const OutrigImportPath = "github.com/outrigdev/outrig"

// HasImport checks if the given import path exists in the AST node
func HasImport(node *ast.File, importPath string) bool {
	// Check node.Imports (populated during parsing)
	for _, imp := range node.Imports {
		if imp.Path.Value == `"`+importPath+`"` {
			return true
		}
	}
	
	// Also check import declarations in node.Decls (for programmatically added imports)
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					if importSpec.Path.Value == `"`+importPath+`"` {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// AddOutrigImport checks if the outrig import exists in the AST node and adds it if not present.
// Returns true if the import was added, false if it already existed.
func AddOutrigImport(node *ast.File) bool {
	// Check if outrig import already exists
	if HasImport(node, OutrigImportPath) {
		return false
	}

	// Add outrig import since it's not present
	outrigImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + OutrigImportPath + `"`,
		},
	}

	// Create a new import declaration for outrig
	importDecl := &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: []ast.Spec{outrigImport},
	}

	// Find the position to insert the new import (after existing imports if any)
	insertPos := 0
	for i, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			insertPos = i + 1
		} else {
			break
		}
	}

	// Insert the new import declaration
	node.Decls = append(node.Decls[:insertPos], append([]ast.Decl{importDecl}, node.Decls[insertPos:]...)...)
	return true
}

// AstFileWrap represents a Go file that has been processed with AST transformations
type AstFileWrap struct {
	OriginalPath string
	ModifiedAST  *ast.File
	FileSet      *token.FileSet
	WasModified  bool
}

// WriteToTempFile writes the AST to a temporary file in the specified directory
func (a *AstFileWrap) WriteToTempFile(tempDir string) (string, error) {
	tempFileName := GenerateTempFileName(a.OriginalPath)
	tempFilePath := filepath.Join(tempDir, tempFileName)
	
	return tempFilePath, WriteASTToFile(a.FileSet, a.ModifiedAST, tempFilePath)
}

// FindMainFunction returns the main function declaration if it exists with proper signature in package main, nil otherwise
func FindMainFunction(node *ast.File) *ast.FuncDecl {
	// Check that this is package main
	if node.Name.Name != "main" {
		return nil
	}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" {
			continue
		}

		// Check that it's not a method (no receiver)
		if fn.Recv != nil {
			continue
		}

		// Check that it has no parameters
		if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
			continue
		}

		// Check that it has no return values
		if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
			continue
		}

		return fn
	}
	return nil
}

// WriteASTToFile writes an AST node to a file using the provided file set
func WriteASTToFile(fset *token.FileSet, node *ast.File, fileName string) error {
	var buf strings.Builder
	config := &printer.Config{
		Mode: printer.SourcePos, // Generate line directives to preserve original line numbers
	}
	err := config.Fprint(&buf, fset, node)
	if err != nil {
		return fmt.Errorf("failed to print modified code: %w", err)
	}

	err = os.WriteFile(fileName, []byte(buf.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", fileName, err)
	}

	return nil
}

// GenerateTempFileName creates a unique filename for the temp directory by hashing the original path
func GenerateTempFileName(originalPath string) string {
	// Get the base filename
	baseName := filepath.Base(originalPath)
	
	// Remove .go extension
	nameWithoutExt := strings.TrimSuffix(baseName, ".go")
	
	// Create hash of the full original path to avoid conflicts
	hash := sha256.Sum256([]byte(originalPath))
	hashStr := fmt.Sprintf("%x", hash)[:8] // Use first 8 chars of hash
	
	// Return formatted filename
	return fmt.Sprintf("%s_%s.go", nameWithoutExt, hashStr)
}