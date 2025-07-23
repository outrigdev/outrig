// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"fmt"
	"go/ast"
)

// FindMainFileAST finds the main file AST from the parsed packages
func FindMainFileAST(transformState *TransformState) (*ast.File, error) {
	mainPkg := transformState.MainPkg

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
