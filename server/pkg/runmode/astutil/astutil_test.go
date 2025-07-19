// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"go/ast"
	"testing"
)

func TestParseOutrigDirective(t *testing.T) {
	tests := []struct {
		name         string
		comment      string
		expectError  bool
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
			// Create a mock comment group for testing
			comment := &ast.Comment{Text: tt.comment}
			commentGroup := &ast.CommentGroup{List: []*ast.Comment{comment}}
			directive := ParseOutrigDirective([]*ast.CommentGroup{commentGroup}, ScopeGo)

			if tt.expectError {
				if directive.Go.Name != "" {
					t.Errorf("Expected error (empty directive), but got directive: %+v", directive)
				}
				return
			}

			if directive.Go.Name == "" {
				t.Errorf("Expected directive, but got empty directive")
				return
			}

			if directive.Go.Name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, directive.Go.Name)
			}
		})
	}
}