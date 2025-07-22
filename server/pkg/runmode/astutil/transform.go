// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"fmt"
	"go/ast"
	"path/filepath"
)

// FindMainFileAST finds the main file AST from the parsed packages
func FindMainFileAST(transformState *TransformState) (*ast.File, error) {
	mainPkg := transformState.PackageMap["main"]
	if mainPkg == nil {
		return nil, fmt.Errorf("no main package found")
	}

	for _, astFile := range mainPkg.Syntax {
		if astFile == nil {
			continue
		}

		if FindMainFunction(astFile) != nil {
			return astFile, nil
		}
	}

	return nil, fmt.Errorf("no main() function found in main package files")
}

// WriteModifiedFiles writes all modified files from TransformState to temporary files for overlay
func WriteModifiedFiles(transformState *TransformState) error {
	// Write all modified files to temp directory
	for originalPath, astFile := range transformState.OldModifiedFiles {
		tempFileName := GenerateTempFileName(originalPath)
		tempFilePath := filepath.Join(transformState.TempDir, tempFileName)

		err := WriteASTToFile(transformState.FileSet, astFile, tempFilePath)
		if err != nil {
			return fmt.Errorf("failed to write modified file %s: %w", originalPath, err)
		}

		transformState.OverlayMap[originalPath] = tempFilePath
	}

	return nil
}
