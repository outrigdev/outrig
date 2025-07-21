// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gr

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"path/filepath"
	"strings"

	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
	"golang.org/x/tools/go/packages"
)

// isPackageBlacklisted checks if a package path should be blacklisted from transformation
func isPackageBlacklisted(pkgPath string) bool {
	// Exact match for github.com/outrigdev/outrig
	if pkgPath == "github.com/outrigdev/outrig" {
		return true
	}

	// Prefix match for github.com/outrigdev/outrig/pkg/**
	if strings.HasPrefix(pkgPath, "github.com/outrigdev/outrig/pkg/") {
		return true
	}

	return false
}

// createOutrigGoCall creates an outrig.Go(name).Run(func() { originalCall }) AST node
func createOutrigGoCall(directive *astutil.OutrigDirective, goStmt *ast.GoStmt) *ast.ExprStmt {
	// Create the wrapper function: func() { originalCall }
	wrapperFunc := &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{}, // no parameters
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{X: goStmt.Call},
			},
		},
	}

	// Create outrig.Go(name)
	outrigGoCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "outrig"},
			Sel: &ast.Ident{Name: "Go"},
		},
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf(`"%s"`, directive.Go.Name),
			},
		},
	}

	// Create .Run(wrapperFunc)
	runCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   outrigGoCall,
			Sel: &ast.Ident{Name: "Run"},
		},
		Args: []ast.Expr{wrapperFunc},
	}

	// Create the expression statement and preserve the position from the original go statement
	result := &ast.ExprStmt{X: runCall}

	// Set positions to match the original go statement
	if goStmt.Pos().IsValid() {
		outrigGoCall.Fun.(*ast.SelectorExpr).X.(*ast.Ident).NamePos = goStmt.Pos()
		runCall.Lparen = goStmt.Pos()
		result.X = runCall
	}

	return result
}

// TransformGoStatementsInPackage iterates over all files in a package and applies go statement transformations
func TransformGoStatementsInPackage(transformState *astutil.TransformState, pkg *packages.Package) bool {
	// Skip blacklisted packages
	if isPackageBlacklisted(pkg.PkgPath) {
		return false
	}

	var hasTransformations bool

	// Iterate over all AST files in the package
	for _, astFile := range pkg.Syntax {
		if astFile == nil {
			continue
		}

		// Apply go statement transformations
		if TransformGoStatements(transformState, astFile) {
			// Mark the file as modified if transformations were applied
			transformState.MarkFileModified(astFile)
			hasTransformations = true

			if transformState.Verbose {
				filePath := transformState.GetFilePath(astFile)
				log.Printf("Applied go statement transformations to: %s", filePath)
			}
		}
	}

	return hasTransformations
}

// TransformGoStatements finds all go statements preceded by //outrig directives and transforms them
func TransformGoStatements(transformState *astutil.TransformState, node *ast.File) bool {
	var transformCount int
	var replacements []struct {
		parent  *ast.BlockStmt
		index   int
		newStmt ast.Stmt
	}

	// Find all go statements that need transformation
	ast.Inspect(node, func(n ast.Node) bool {
		parent, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for i, stmt := range parent.List {
			goStmt, ok := stmt.(*ast.GoStmt)
			if !ok {
				continue
			}
			// Look for an outrig directive in the comments before this go statement
			directive := astutil.ParseOutrigDirectiveForStmt(transformState.FileSet, node, goStmt, astutil.ScopeGo)
			// Transform the go statement
			newCall := createOutrigGoCall(&directive, goStmt)
			replacements = append(replacements, struct {
				parent  *ast.BlockStmt
				index   int
				newStmt ast.Stmt
			}{parent, i, newCall})
			transformCount++
		}
		return true
	})

	// Apply all replacements
	for _, replacement := range replacements {
		replacement.parent.List[replacement.index] = replacement.newStmt
	}

	// Add outrig import if we made any transformations and it's not already present
	if transformCount > 0 {
		astutil.AddOutrigImport(transformState.FileSet, node)
		if transformState.Verbose {
			fileName := transformState.GetFilePath(node)
			log.Printf("Transformed %d go statements in file: %s\n", transformCount, filepath.Base(fileName))
		}
	}

	return transformCount > 0
}
