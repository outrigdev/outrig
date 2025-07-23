// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gr

import (
	"fmt"
	"go/ast"
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

// createOutrigGoCallPrelude creates the outrig.Go("name").WithTags("...").Run(func() { part
func createOutrigGoCallPrelude(directive *astutil.OutrigDirective) string {
	code := fmt.Sprintf("outrig.Go(%q)", directive.Go.Name)
	if directive.Go.Tags != "" {
		// Split tags on comma and create separate arguments
		tags := strings.Split(directive.Go.Tags, ",")
		var quotedTags []string
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				quotedTags = append(quotedTags, fmt.Sprintf("%q", tag))
			}
		}
		if len(quotedTags) > 0 {
			code += ".WithTags(" + strings.Join(quotedTags, ", ") + ")"
		}
	}
	code += ".Run(func() {\n"
	return code
}

// TransformGoStatementsInPackageWithReplacement iterates over all files in a package and applies go statement transformations using the replacement system
func TransformGoStatementsInPackageWithReplacement(transformState *astutil.TransformState, pkg *packages.Package) bool {
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

		// Get or create ModifiedFile for this AST file
		filePath := transformState.GetFilePath(astFile)
		modifiedFile, exists := transformState.ModifiedFiles[filePath]
		if !exists {
			var err error
			modifiedFile, err = astutil.MakeModifiedFile(transformState, astFile)
			if err != nil {
				if transformState.Verbose {
					log.Printf("Failed to create ModifiedFile for %s: %v", filePath, err)
				}
				continue
			}
			transformState.ModifiedFiles[filePath] = modifiedFile
		}

		// Apply go statement transformations using replacements
		if transformCount := TransformGoStatementsWithReplacement(transformState, modifiedFile); transformCount > 0 {
			hasTransformations = true
			modifiedFile.Modified = true
		}
	}

	return hasTransformations
}

// transformSingleGoStatement processes a single go statement and applies the outrig transformation if needed
func transformSingleGoStatement(transformState *astutil.TransformState, modifiedFile *astutil.ModifiedFile, goStmt *ast.GoStmt) bool {
	// Look for an outrig directive in the comments before this go statement
	directive := astutil.ParseOutrigDirectiveForStmt(transformState.FileSet, modifiedFile.FileAST, goStmt, astutil.ScopeGo)

	// Get the position of the "go" keyword and the end of the call
	goPos := transformState.FileSet.Position(goStmt.Pos())
	callPos := transformState.FileSet.Position(goStmt.Call.Pos())
	callEndPos := transformState.FileSet.Position(goStmt.Call.End())

	// Create the outrig.Go().Run(func() { prelude
	prelude := createOutrigGoCallPrelude(&directive)

	// Delete the "go " keyword
	deleteReplacement := astutil.Replacement{
		Mode:     astutil.ReplacementModeDelete,
		StartPos: int64(goPos.Offset),
		EndPos:   int64(callPos.Offset),
	}
	modifiedFile.Replacements = append(modifiedFile.Replacements, deleteReplacement)

	// Insert the prelude at the exact position and add line directive separately
	modifiedFile.AddInsert(int64(goPos.Offset), prelude)
	modifiedFile.AddLineDirective(int64(goPos.Offset), goPos.Filename, goPos.Line)

	// Add the closing part after the call
	modifiedFile.AddInsert(int64(callEndPos.Offset), " })")

	return true
}

// TransformGoStatementsWithReplacement finds all go statements preceded by //outrig directives and transforms them using the replacement system
func TransformGoStatementsWithReplacement(transformState *astutil.TransformState, modifiedFile *astutil.ModifiedFile) int {
	var transformCount int

	// Find all go statements that need transformation
	ast.Inspect(modifiedFile.FileAST, func(n ast.Node) bool {
		goStmt, ok := n.(*ast.GoStmt)
		if !ok {
			return true
		}

		if transformSingleGoStatement(transformState, modifiedFile, goStmt) {
			transformCount++
		}

		return true
	})

	// Add outrig import if we made any transformations
	if transformCount > 0 {
		err := astutil.AddOutrigImportReplacement(transformState, modifiedFile)
		if err != nil && transformState.Verbose {
			log.Printf("Failed to add outrig import replacement: %v", err)
		}

		if transformState.Verbose {
			fileName := transformState.GetFilePath(modifiedFile.FileAST)
			log.Printf("Transformed %d go statements in file: %s\n", transformCount, filepath.Base(fileName))
		}
	}

	return transformCount
}
