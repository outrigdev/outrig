// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loglineparser

import (
	"testing"
)

func TestFindFirstJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		allowArrays bool
		expected    *Position
	}{
		{
			name:        "simple object",
			input:       `{"key": "value"}`,
			allowArrays: false,
			expected:    &Position{Start: 0, End: 16},
		},
		{
			name:        "simple array - not detected when arrays disabled",
			input:       `[1, 2, 3]`,
			allowArrays: false,
			expected:    nil,
		},
		{
			name:        "simple array - detected when arrays enabled",
			input:       `[1, 2, 3]`,
			allowArrays: true,
			expected:    &Position{Start: 0, End: 9},
		},
		{
			name:        "object in text",
			input:       `some text {"key": "value"} more text`,
			allowArrays: false,
			expected:    &Position{Start: 10, End: 26},
		},
		{
			name:        "array in text - not detected when arrays disabled",
			input:       `prefix [1, 2, 3] suffix`,
			allowArrays: false,
			expected:    nil,
		},
		{
			name:        "array in text - detected when arrays enabled",
			input:       `prefix [1, 2, 3] suffix`,
			allowArrays: true,
			expected:    &Position{Start: 7, End: 16},
		},
		{
			name:        "nested object",
			input:       `{"outer": {"inner": "value"}}`,
			allowArrays: false,
			expected:    &Position{Start: 0, End: 29},
		},
		{
			name:        "nested array - not detected when arrays disabled",
			input:       `[[1, 2], [3, 4]]`,
			allowArrays: false,
			expected:    nil,
		},
		{
			name:        "nested array - detected when arrays enabled",
			input:       `[[1, 2], [3, 4]]`,
			allowArrays: true,
			expected:    &Position{Start: 0, End: 16},
		},
		{
			name:        "mixed nesting",
			input:       `{"array": [1, 2, {"nested": true}]}`,
			allowArrays: false,
			expected:    &Position{Start: 0, End: 35},
		},
		{
			name:        "string with escaped quotes",
			input:       `{"key": "value with \"quotes\""}`,
			allowArrays: false,
			expected:    &Position{Start: 0, End: 32},
		},
		{
			name:        "string with braces inside",
			input:       `{"key": "value with {braces}"}`,
			allowArrays: false,
			expected:    &Position{Start: 0, End: 30},
		},
		{
			name:     "invalid json - malformed",
			input:    `[5 5 5]`,
			expected: nil,
		},
		{
			name:     "invalid json - unmatched brace",
			input:    `{"key": "value"`,
			expected: nil,
		},
		{
			name:     "invalid json - unmatched bracket",
			input:    `[1, 2, 3`,
			expected: nil,
		},
		{
			name:     "ambiguous json - 1",
			input:    `[1, 2, 3 []]`,
			expected: nil,
		},
		{
			name:     "ambiguous json - 2",
			input:    `there are [4] lights`,
			expected: nil,
		},
		{
			name:     "no json",
			input:    `just some regular text`,
			expected: nil,
		},
		{
			name:        "empty string",
			input:       ``,
			allowArrays: false,
			expected:    nil,
		},
		{
			name:        "first json wins",
			input:       `s:{"first": true} {"second": false}`,
			allowArrays: false,
			expected:    &Position{Start: 2, End: 17},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindFirstJSON(tt.input, tt.allowArrays)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %+v, got nil", tt.expected)
				return
			}

			if result.Start != tt.expected.Start || result.End != tt.expected.End {
				t.Errorf("expected %+v, got %+v", tt.expected, result)

				// Show the actual substring for debugging
				if result.Start >= 0 && result.End <= len(tt.input) {
					t.Errorf("actual substring: %q", tt.input[result.Start:result.End])
				}
			}
		})
	}
}

func TestIsJsonAt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		index    int
		expected *Position
	}{
		{
			name:     "object at start",
			input:    `{"key": "value"}`,
			index:    0,
			expected: &Position{Start: 0, End: 16},
		},
		{
			name:     "array at start",
			input:    `[1, 2, 3]`,
			index:    0,
			expected: &Position{Start: 0, End: 9},
		},
		{
			name:     "object in middle",
			input:    `text {"key": "value"} more`,
			index:    5,
			expected: &Position{Start: 5, End: 21},
		},
		{
			name:     "not json at index",
			input:    `{"key": "value"}`,
			index:    1,
			expected: nil,
		},
		{
			name:     "invalid json at index",
			input:    `{invalid}`,
			index:    0,
			expected: nil,
		},
		{
			name:     "index out of bounds",
			input:    `{"key": "value"}`,
			index:    20,
			expected: nil,
		},
		{
			name:     "empty string",
			input:    ``,
			index:    0,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJsonAt(tt.input, tt.index)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %+v, got nil", tt.expected)
				return
			}

			if result.Start != tt.expected.Start || result.End != tt.expected.End {
				t.Errorf("expected %+v, got %+v", tt.expected, result)

				// Show the actual substring for debugging
				if result.Start >= 0 && result.End <= len(tt.input) {
					t.Errorf("actual substring: %q", tt.input[result.Start:result.End])
				}
			}
		})
	}
}
