// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gr

import (
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"
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
			expected: `go func() {
		println("test")
	}()`,
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

			transformed := TransformGoStatements(fset, node)

			var buf strings.Builder
			config := &printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
			err = config.Fprint(&buf, fset, node)
			if err != nil {
				t.Fatalf("Failed to print AST: %v", err)
			}

			result := buf.String()

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

func TestParseOutrigDirective(t *testing.T) {
	tests := []struct {
		name        string
		comment     string
		expectError bool
		expectedName string
	}{
		{
			name:         "valid simple directive",
			comment:      `//outrig name="hello"`,
			expectError:  false,
			expectedName: "hello",
		},
		{
			name:         "valid directive with spaces",
			comment:      `//outrig name="worker-task"`,
			expectError:  false,
			expectedName: "worker-task",
		},
		{
			name:        "invalid directive - no name",
			comment:     `//outrig`,
			expectError: true,
		},
		{
			name:        "invalid directive - malformed",
			comment:     `//outrig invalid`,
			expectError: true,
		},
		{
			name:        "missing name attribute",
			comment:     `//outrig tags="test"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directive, err := parseOutrigDirective(tt.comment)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if directive.Name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, directive.Name)
			}
		})
	}
}