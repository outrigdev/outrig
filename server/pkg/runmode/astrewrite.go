// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runmode

import (
	"go/ast"
	"go/token"

	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
)

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
