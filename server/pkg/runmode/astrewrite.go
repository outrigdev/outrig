// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runmode

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
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

	// Check if outrig import already exists
	hasOutrigImport := false
	var firstNonImportDeclLine int
	for _, imp := range node.Imports {
		if imp.Path.Value == `"`+outrigImportPath+`"` {
			hasOutrigImport = true
			break
		}
	}

	// Find the line number after imports for line directive restoration
	if len(node.Decls) > 0 {
		for _, decl := range node.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
				continue
			}
			// This is the first non-import declaration
			firstNonImportDeclLine = fset.Position(decl.Pos()).Line
			break
		}
	}

	// Add outrig import if not present
	addedImport := false
	if !hasOutrigImport {
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
		addedImport = true
	}

	// Find and modify the main function
	mainFound := false
	var mainFuncStartLine int
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			if fn.Body == nil {
				return false
			}

			// Get the line number of the first statement in main (or opening brace + 1)
			if len(fn.Body.List) > 0 {
				mainFuncStartLine = fset.Position(fn.Body.List[0].Pos()).Line
			} else {
				mainFuncStartLine = fset.Position(fn.Body.Lbrace).Line + 1
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
		}
		return true
	})

	if !mainFound {
		return "", fmt.Errorf("unable to find main entry point. Ensure your application has a valid main()")
	}

	// Generate the modified source code
	var buf strings.Builder
	err = format.Node(&buf, fset, node)
	if err != nil {
		return "", fmt.Errorf("failed to format modified code: %w", err)
	}

	// Post-process the generated code to add line directives
	modifiedCode := buf.String()
	modifiedCode = addLineDirectives(modifiedCode, sourceFile, addedImport, firstNonImportDeclLine, mainFuncStartLine)

	// Create file with original name in temp directory
	originalName := filepath.Base(sourceFile)
	tempFilePath := filepath.Join(tempDir, originalName)

	err = os.WriteFile(tempFilePath, []byte(modifiedCode), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	return tempFilePath, nil
}

// addLineDirectives adds //line directives to restore correct line numbers
func addLineDirectives(code, sourceFile string, addedImport bool, firstNonImportDeclLine, mainFuncStartLine int) string {
	lines := strings.Split(code, "\n")
	var result []string
	
	for _, line := range lines {
		result = append(result, line)
		
		// If we added an import, add a line directive after the import block
		if addedImport && firstNonImportDeclLine > 0 && strings.Contains(line, outrigImportPath) {
			result = append(result, fmt.Sprintf("//line %s:%d", sourceFile, firstNonImportDeclLine))
		}
		
		// Add line directive after the injected outrig calls in main
		if mainFuncStartLine > 0 && strings.Contains(line, "outrig.AppDone()") {
			result = append(result, fmt.Sprintf("//line %s:%d", sourceFile, mainFuncStartLine))
		}
	}
	
	return strings.Join(result, "\n")
}
