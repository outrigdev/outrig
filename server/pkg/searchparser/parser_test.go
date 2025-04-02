// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package searchparser

import (
	"testing"
)

func TestParseAST(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Node
	}{
		{
			name:  "empty string",
			input: "",
			expected: &Node{
				Type:     "and",
				Position: Position{Start: 0, End: 0},
				Children: []Node{},
			},
		},
		{
			name:  "single token",
			input: "hello",
			expected: &Node{
				Type:       "search",
				Position:   Position{Start: 0, End: 5},
				SearchType: "exact",
				SearchTerm: "hello",
			},
		},
		{
			name:  "two tokens (implicit AND)",
			input: "hello world",
			expected: &Node{
				Type:     "and",
				Position: Position{Start: 0, End: 11},
				Children: []Node{
					{
						Type:       "search",
						Position:   Position{Start: 0, End: 5},
						SearchType: "exact",
						SearchTerm: "hello",
					},
					{
						Type:       "search",
						Position:   Position{Start: 6, End: 11},
						SearchType: "exact",
						SearchTerm: "world",
					},
				},
			},
		},
		{
			name:  "OR expression",
			input: "hello | world",
			expected: &Node{
				Type:     "or",
				Position: Position{Start: 0, End: 13},
				Children: []Node{
					{
						Type:       "search",
						Position:   Position{Start: 0, End: 5},
						SearchType: "exact",
						SearchTerm: "hello",
					},
					{
						Type:       "search",
						Position:   Position{Start: 8, End: 13},
						SearchType: "exact",
						SearchTerm: "world",
					},
				},
			},
		},
		{
			name:  "NOT token",
			input: "-hello",
			expected: &Node{
				Type:       "search",
				Position:   Position{Start: 0, End: 6},
				SearchType: "exact",
				SearchTerm: "hello",
				IsNot:      true,
			},
		},
		{
			name:  "field prefix",
			input: "$field:hello",
			expected: &Node{
				Type:       "search",
				Position:   Position{Start: 0, End: 12},
				SearchType: "exact",
				SearchTerm: "hello",
				Field:      "field",
			},
		},
		{
			name:  "complex expression",
			input: "hello world | -$field:test",
			expected: &Node{
				Type:     "or",
				Position: Position{Start: 0, End: 26},
				Children: []Node{
					{
						Type:     "and",
						Position: Position{Start: 0, End: 11},
						Children: []Node{
							{
								Type:       "search",
								Position:   Position{Start: 0, End: 5},
								SearchType: "exact",
								SearchTerm: "hello",
							},
							{
								Type:       "search",
								Position:   Position{Start: 6, End: 11},
								SearchType: "exact",
								SearchTerm: "world",
							},
						},
					},
					{
						Type:       "search",
						Position:   Position{Start: 14, End: 26},
						SearchType: "exact",
						SearchTerm: "test",
						Field:      "field",
						IsNot:      true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSearch(tt.input)

			// Compare the result with the expected value
			compareNodes(t, result, tt.expected)
		})
	}
}

// compareNodes recursively compares two nodes for equality
func compareNodes(t *testing.T, actual, expected *Node) {
	if actual == nil && expected == nil {
		return
	}
	if actual == nil {
		t.Errorf("actual is nil, expected %+v", expected)
		return
	}
	if expected == nil {
		t.Errorf("expected is nil, actual %+v", actual)
		return
	}

	// Compare node types
	if actual.Type != expected.Type {
		t.Errorf("node type mismatch: got %s, want %s", actual.Type, expected.Type)
	}

	// Compare positions
	if actual.Position.Start != expected.Position.Start {
		t.Errorf("position start mismatch: got %d, want %d", actual.Position.Start, expected.Position.Start)
	}
	if actual.Position.End != expected.Position.End {
		t.Errorf("position end mismatch: got %d, want %d", actual.Position.End, expected.Position.End)
	}

	// Compare search-specific fields for search nodes
	if actual.Type == "search" {
		if actual.SearchType != expected.SearchType {
			t.Errorf("search type mismatch: got %s, want %s", actual.SearchType, expected.SearchType)
		}
		if actual.SearchTerm != expected.SearchTerm {
			t.Errorf("search term mismatch: got %s, want %s", actual.SearchTerm, expected.SearchTerm)
		}
		if actual.Field != expected.Field {
			t.Errorf("field mismatch: got %s, want %s", actual.Field, expected.Field)
		}
		if actual.IsNot != expected.IsNot {
			t.Errorf("isNot mismatch: got %t, want %t", actual.IsNot, expected.IsNot)
		}
	}

	// Compare children for non-leaf nodes
	if len(actual.Children) != len(expected.Children) {
		t.Errorf("children count mismatch: got %d, want %d", len(actual.Children), len(expected.Children))
		return
	}

	for i := range actual.Children {
		// Get pointers to the children
		actualChild := &actual.Children[i]
		expectedChild := &expected.Children[i]

		// Recursively compare children
		compareNodes(t, actualChild, expectedChild)
	}
}
