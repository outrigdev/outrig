// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gr

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/outrigdev/outrig/server/pkg/runmode/astutil"
)

func TestTransformGoStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple outrig directive",
			input: `package main

func main() {
	//outrig name="hello"
	go func() {
		println("test")
	}()
}`,
			expected: `outrig.Go("hello").Run(func() {
		func() {
			println("test")
		}()
	})`,
		},
		{
			name: "no outrig directive",
			input: `package main

func main() {
	go func() {
		println("test")
	}()
}`,
			expected: `outrig.Go("").Run(func() {
		func() {
			println("test")
		}()
	})`,
		},
		{
			name: "outrig directive with function call",
			input: `package main

func worker() {
	println("working")
}

func main() {
	//outrig name="worker-task"
	go worker()
}`,
			expected: `outrig.Go("worker-task").Run(func() {
		worker()
	})`,
		},
		{
			name: "outrig directive with parameterized anonymous function",
			input: `package main

func main() {
	//outrig name="param-task"
	go func(x int) {
		println("value:", x)
	}(42)
}`,
			expected: `outrig.Go("param-task").Run(func() {
		func(x int) {
			println("value:", x)
		}(42)
	})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", tt.input, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Create a minimal TransformState for testing
			transformState := &astutil.TransformState{
				FileSet:       fset,
				Verbose:       false,
				ModifiedFiles: make(map[string]*astutil.ModifiedFile),
			}

			// Create a ModifiedFile manually for testing
			modifiedFile := &astutil.ModifiedFile{
				FileAST:      node,
				Replacements: []astutil.Replacement{},
				RawBytes:     []byte(tt.input),
				Modified:     false,
			}
			transformState.ModifiedFiles["test.go"] = modifiedFile

			// Apply transformations using the replacement system
			transformCount := TransformGoStatementsWithReplacement(transformState, modifiedFile)
			transformed := transformCount > 0

			// Apply replacements to get the final result
			resultBytes := astutil.ApplyReplacements(modifiedFile.RawBytes, modifiedFile.Replacements)
			result := string(resultBytes)

			// Check if transformation occurred as expected
			if strings.Contains(tt.expected, "outrig.Go") {
				if !transformed {
					t.Errorf("Expected transformation to occur, but it didn't")
				}
				// Check for key elements of the transformation
				if !strings.Contains(result, `outrig.Go("`) {
					t.Errorf("Expected output to contain outrig.Go call, but got:\n%s", result)
				}
				if !strings.Contains(result, `.Run(func() {`) {
					t.Errorf("Expected output to contain .Run(func() call, but got:\n%s", result)
				}
				if !strings.Contains(result, `import "github.com/outrigdev/outrig"`) {
					t.Errorf("Expected output to contain outrig import, but got:\n%s", result)
				}
			} else {
				if transformed {
					t.Errorf("Expected no transformation, but transformation occurred")
				}
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", tt.expected, result)
				}
			}
		})
	}
}

