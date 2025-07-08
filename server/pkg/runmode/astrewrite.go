// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runmode

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
	"github.com/outrigdev/outrig/server/pkg/runmode/gr"
)

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


// fileHasMainFunction checks if a Go file contains a proper main() function in package main
func fileHasMainFunction(filename string) (bool, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		return false, err
	}

	return astutil.FindMainFunction(node) != nil, nil
}

// modifyMainFunction finds the main function in the AST and injects outrig.Init() and defer outrig.AppDone() calls.
// Returns true if the main function was found and modified, false otherwise.
func modifyMainFunction(node *ast.File) bool {
	fn := astutil.FindMainFunction(node)
	if fn == nil || fn.Body == nil {
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
	return true
}


// RewriteGoFiles processes all Go files from the packages, applying appropriate transformations:
// - modifyMainFunction is applied ONLY to the file containing main()
// - TransformGoStatements is applied to ALL files
// Returns a slice of AstFileWrap structs for files that were modified.
func RewriteGoFiles(goFiles []string, mainFile string, fileSet *token.FileSet) ([]*astutil.AstFileWrap, error) {
	// Use provided FileSet if available, otherwise create a new one
	fset := fileSet
	if fset == nil {
		fset = token.NewFileSet()
	}

	var modifiedFiles []*astutil.AstFileWrap

	for _, sourceFile := range goFiles {
		// Parse the source file
		node, err := parser.ParseFile(fset, sourceFile, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", sourceFile, err)
		}

		// Track if any modifications were made
		modified := false

		// If this is the main file, add outrig import and modify main function
		if sourceFile == mainFile {
			astutil.AddOutrigImport(node)
			if !modifyMainFunction(node) {
				return nil, fmt.Errorf("unable to find main entry point in %s. Ensure your application has a valid main()", sourceFile)
			}
			modified = true
		}

		// Apply gr transformations to ALL files
		if gr.TransformGoStatements(fset, node) {
			modified = true
		}

		// Only include files that were modified
		if modified {
			modifiedFiles = append(modifiedFiles, &astutil.AstFileWrap{
				OriginalPath: sourceFile,
				ModifiedAST:  node,
				FileSet:      fset,
				WasModified:  true,
			})
		}
	}

	return modifiedFiles, nil
}

// WriteTempFiles writes all modified files to the temp directory with unique names
// Returns a map of original file paths to temporary file paths.
func WriteTempFiles(modifiedFiles []*astutil.AstFileWrap, tempDir string) (map[string]string, error) {
	overlayMap := make(map[string]string)

	for _, modifiedFile := range modifiedFiles {
		// Write the modified AST to the temp file using the wrapper method
		tempFilePath, err := modifiedFile.WriteToTempFile(tempDir)
		if err != nil {
			return nil, fmt.Errorf("failed to write temp file for %s: %w", modifiedFile.OriginalPath, err)
		}

		overlayMap[modifiedFile.OriginalPath] = tempFilePath
	}

	return overlayMap, nil
}

// RewriteAndCreateTempFiles processes all Go files from the packages, applying appropriate transformations
// and writing them to temp files. This is the main entry point that combines RewriteGoFiles and WriteTempFiles.
func RewriteAndCreateTempFiles(goFiles []string, mainFile string, tempDir string, fileSet *token.FileSet) (map[string]string, error) {
	// First, rewrite the ASTs
	modifiedFiles, err := RewriteGoFiles(goFiles, mainFile, fileSet)
	if err != nil {
		return nil, err
	}

	// Then write them to temp files
	return WriteTempFiles(modifiedFiles, tempDir)
}

