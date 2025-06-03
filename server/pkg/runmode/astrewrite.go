// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runmode

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const outrigImportPath = "github.com/outrigdev/outrig"

// FindMainFile searches through the provided Go files and returns the one containing the main() function
// Returns an error if no main() function is found or if multiple main() functions are found
func FindMainFile(goFiles []string) (string, error) {
	var mainFiles []string

	for _, file := range goFiles {
		hasMain, err := fileHasMainFunction(file)
		if err != nil {
			continue // Skip files that can't be parsed
		}
		if hasMain {
			mainFiles = append(mainFiles, file)
		}
	}

	if len(mainFiles) == 0 {
		return "", fmt.Errorf("no main() function found in any of the provided Go files")
	}

	if len(mainFiles) > 1 {
		return "", fmt.Errorf("multiple main() functions found in files: %v", mainFiles)
	}

	return mainFiles[0], nil
}

// fileHasMainFunction checks if a Go file contains a main() function
func fileHasMainFunction(filename string) (bool, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return false, err
	}

	hasMain := false
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			hasMain = true
			return false
		}
		return true
	})

	return hasMain, nil
}

// addOutrigImport checks if the outrig import exists in the AST node and adds it if not present.
// Returns true if the import was added, false if it already existed.
func addOutrigImport(node *ast.File) bool {
	// Check if outrig import already exists
	for _, imp := range node.Imports {
		if imp.Path.Value == `"`+outrigImportPath+`"` {
			return false
		}
	}

	// Add outrig import since it's not present
	outrigImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + outrigImportPath + `"`,
		},
	}

	// Always create a new import declaration for outrig to avoid formatting issues
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

// modifyMainFunction finds the main function in the AST and injects outrig.Init() and defer outrig.AppDone() calls.
// Returns true if the main function was found and modified, false otherwise.
func modifyMainFunction(node *ast.File) bool {
	mainFound := false
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" {
			return true
		}

		if fn.Body == nil {
			return false
		}

		// Create the outrig.Init("", nil) call statement
		initCall := &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "outrig"},
					Sel: &ast.Ident{Name: "Init"},
				},
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: `""`,
					},
					&ast.Ident{Name: "nil"},
				},
			},
		}

		// Create the defer outrig.AppDone() call statement
		deferCall := &ast.DeferStmt{
			Call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "outrig"},
					Sel: &ast.Ident{Name: "AppDone"},
				},
			},
		}

		// Insert both calls at the beginning of the function body
		fn.Body.List = append([]ast.Stmt{initCall, deferCall}, fn.Body.List...)
		mainFound = true
		return false
	})

	return mainFound
}

// RewriteAndCreateTempFile parses the given Go file, injects outrig.Init() into main(),
// and creates a temporary file with the modified code in the provided temp directory.
// Returns the path to the temporary file.
func RewriteAndCreateTempFile(sourceFile string, tempDir string) (string, error) {
	// Parse the source file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, sourceFile, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", sourceFile, err)
	}

	// Add outrig import if not already present
	addOutrigImport(node)

	// Find and modify the main function
	if !modifyMainFunction(node) {
		return "", fmt.Errorf("unable to find main entry point. Ensure your application has a valid main()")
	}

	// Generate the modified source code with line directives
	var buf strings.Builder
	config := &printer.Config{
		Mode: printer.SourcePos, // Generate line directives to preserve original line numbers
	}
	err = config.Fprint(&buf, fset, node)
	if err != nil {
		return "", fmt.Errorf("failed to print modified code: %w", err)
	}

	modifiedCode := buf.String()

	// Create file with original name in temp directory
	originalName := filepath.Base(sourceFile)
	tempFilePath := filepath.Join(tempDir, originalName)

	err = os.WriteFile(tempFilePath, []byte(modifiedCode), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	return tempFilePath, nil
}

