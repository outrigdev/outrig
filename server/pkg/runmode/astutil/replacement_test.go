// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"testing"
)

func TestApplyReplacements(t *testing.T) {
	tests := []struct {
		name         string
		fileBytes    []byte
		replacements []Replacement
		expected     []byte
	}{
		{
			name:      "double insertion at same position",
			fileBytes: []byte("ppy"),
			replacements: []Replacement{
				{Mode: ReplacementModeInsert, StartPos: 0, NewText: []byte("h")},
				{Mode: ReplacementModeInsert, StartPos: 0, NewText: []byte("a")},
			},
			expected: []byte("happy"),
		},
		{
			name:      "insertion in deleted chunk should be ignored",
			fileBytes: []byte("hello world"),
			replacements: []Replacement{
				{Mode: ReplacementModeDelete, StartPos: 0, EndPos: 5},
				{Mode: ReplacementModeInsert, StartPos: 2, NewText: []byte("X")},
			},
			expected: []byte(" world"),
		},
		{
			name:      "insertion beyond end of file should be ignored",
			fileBytes: []byte("hello"),
			replacements: []Replacement{
				{Mode: ReplacementModeInsert, StartPos: 10, NewText: []byte("world")},
			},
			expected: []byte("hello"),
		},
		{
			name:      "deletion beyond end of file should truncate",
			fileBytes: []byte("hello"),
			replacements: []Replacement{
				{Mode: ReplacementModeDelete, StartPos: 3, EndPos: 20},
			},
			expected: []byte("hel"),
		},
		{
			name:      "normal insertion",
			fileBytes: []byte("hello world"),
			replacements: []Replacement{
				{Mode: ReplacementModeInsert, StartPos: 5, NewText: []byte(" beautiful")},
			},
			expected: []byte("hello beautiful world"),
		},
		{
			name:      "normal deletion",
			fileBytes: []byte("hello world"),
			replacements: []Replacement{
				{Mode: ReplacementModeDelete, StartPos: 5, EndPos: 6},
			},
			expected: []byte("helloworld"),
		},
		{
			name:      "mixed operations",
			fileBytes: []byte("hello world"),
			replacements: []Replacement{
				{Mode: ReplacementModeInsert, StartPos: 0, NewText: []byte("Hi ")},
				{Mode: ReplacementModeDelete, StartPos: 5, EndPos: 6},
				{Mode: ReplacementModeInsert, StartPos: 11, NewText: []byte("!")},
			},
			expected: []byte("Hi helloworld!"),
		},
		{
			name:      "empty file with insertion",
			fileBytes: []byte(""),
			replacements: []Replacement{
				{Mode: ReplacementModeInsert, StartPos: 0, NewText: []byte("hello")},
			},
			expected: []byte("hello"),
		},
		{
			name:         "no replacements",
			fileBytes:    []byte("hello world"),
			replacements: []Replacement{},
			expected:     []byte("hello world"),
		},
		{
			name:      "deletion at start of file",
			fileBytes: []byte("hello world"),
			replacements: []Replacement{
				{Mode: ReplacementModeDelete, StartPos: 0, EndPos: 6},
			},
			expected: []byte("world"),
		},
		{
			name:      "deletion at end of file",
			fileBytes: []byte("hello world"),
			replacements: []Replacement{
				{Mode: ReplacementModeDelete, StartPos: 6, EndPos: 11},
			},
			expected: []byte("hello "),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyReplacements(tt.fileBytes, tt.replacements)
			if string(result) != string(tt.expected) {
				t.Errorf("Expected %q, got %q", string(tt.expected), string(result))
			}
		})
	}
}
