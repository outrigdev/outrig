// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gr

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
)

const outrigCommentPrefix = "//outrig "

// outrigDirectiveRegex matches lines that start with //outrig followed by key="value" pairs
var outrigDirectiveRegex = regexp.MustCompile(`^//outrig\s+(\w+="[^"]*"\s*)+$`)

// keyValueRegex extracts key="value" pairs from the directive
var keyValueRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)

// OutrigDirective represents a parsed //outrig comment directive
type OutrigDirective struct {
	Name string
	Tags string // for future use
}

// parseOutrigDirective parses an //outrig comment and extracts the directive information
func parseOutrigDirective(comment string) (*OutrigDirective, error) {
	// Remove leading // and whitespace
	content := strings.TrimSpace(strings.TrimPrefix(comment, "//"))

	// Extract key-value pairs directly without strict regex validation
	matches := keyValueRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no key-value pairs found in directive")
	}
	directive := &OutrigDirective{}

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		key := match[1]
		value := match[2]

		switch key {
		case "name":
			directive.Name = value
		case "tags":
			directive.Tags = value
		}
	}

	if directive.Name == "" {
		return nil, fmt.Errorf("name attribute is required in outrig directive")
	}

	return directive, nil
}

// createOutrigGoCall creates an outrig.Go(name).Run(func() { originalCall }) AST node
func createOutrigGoCall(directive *OutrigDirective, originalCall ast.Expr) *ast.ExprStmt {
	// Create the wrapper function: func() { originalCall }
	wrapperFunc := &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{}, // no parameters
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{X: originalCall},
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
				Value: fmt.Sprintf(`"%s"`, directive.Name),
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

	return &ast.ExprStmt{X: runCall}
}

// TransformGoStatements finds all go statements preceded by //outrig directives and transforms them
func TransformGoStatements(fset *token.FileSet, node *ast.File) bool {
	var transformed bool
	var replacements []struct {
		parent  *ast.BlockStmt
		index   int
		newStmt ast.Stmt
	}
	var commentsToRemove []*ast.Comment

	// First pass: find all go statements that need transformation
	ast.Inspect(node, func(n ast.Node) bool {
		switch parent := n.(type) {
		case *ast.BlockStmt:
			for i, stmt := range parent.List {
				if goStmt, ok := stmt.(*ast.GoStmt); ok {
					// Look for an outrig directive in the comments before this go statement
					directive, comment := findOutrigDirectiveWithComment(fset, node.Comments, goStmt.Pos())
					if directive != nil {
						// Transform the go statement
						newCall := createOutrigGoCall(directive, goStmt.Call)
						replacements = append(replacements, struct {
							parent  *ast.BlockStmt
							index   int
							newStmt ast.Stmt
						}{parent, i, newCall})
						if comment != nil {
							commentsToRemove = append(commentsToRemove, comment)
						}
						transformed = true
					}
				}
			}
		}
		return true
	})

	// Second pass: apply all replacements
	for _, replacement := range replacements {
		replacement.parent.List[replacement.index] = replacement.newStmt
	}

	// Third pass: remove outrig comments that were processed
	if len(commentsToRemove) > 0 {
		removeComments(node, commentsToRemove)
	}

	// Add outrig import if we made any transformations and it's not already present
	if transformed {
		astutil.AddOutrigImport(node)
	}

	return transformed
}

// findOutrigDirectiveWithComment looks for an //outrig comment and returns both the directive and the comment
func findOutrigDirectiveWithComment(fset *token.FileSet, comments []*ast.CommentGroup, pos token.Pos) (*OutrigDirective, *ast.Comment) {
	if len(comments) == 0 {
		return nil, nil
	}

	targetLine := fset.Position(pos).Line

	// Look for comments on the line immediately before the go statement
	for _, commentGroup := range comments {
		for _, comment := range commentGroup.List {
			commentLine := fset.Position(comment.Pos()).Line

			// Check if this comment is on the line before our target and starts with outrig prefix
			if commentLine == targetLine-1 && strings.HasPrefix(comment.Text, outrigCommentPrefix) {
				directive, err := parseOutrigDirective(comment.Text)
				if err != nil {
					// Skip invalid directives but don't fail the build
					// TODO: emit warning for malformed directives
					continue
				}
				return directive, comment
			}
		}
	}

	return nil, nil
}

// removeComments removes specific comments from the AST
func removeComments(node *ast.File, commentsToRemove []*ast.Comment) {
	var newCommentGroups []*ast.CommentGroup

	for _, group := range node.Comments {
		var newComments []*ast.Comment
		for _, comment := range group.List {
			shouldRemove := false
			for _, toRemove := range commentsToRemove {
				if comment == toRemove {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				newComments = append(newComments, comment)
			}
		}
		if len(newComments) > 0 {
			newGroup := &ast.CommentGroup{List: newComments}
			newCommentGroups = append(newCommentGroups, newGroup)
		}
	}

	node.Comments = newCommentGroups
}
