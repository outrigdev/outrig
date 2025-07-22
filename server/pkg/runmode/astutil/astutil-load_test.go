// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"testing"
)

func TestFindStatementBoundary(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedAdvance int64
		expectedNewlines int
		expectedEndsWithNewline bool
		expectedFound   bool
	}{
		{
			name:            "empty input",
			input:           "",
			expectedAdvance: 0,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   true, // end-of-input matches $
		},
		{
			name:            "simple newline",
			input:           "\n",
			expectedAdvance: 1,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "windows newline",
			input:           "\r\n",
			expectedAdvance: 2,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "semicolon",
			input:           ";",
			expectedAdvance: 1,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   true,
		},
		{
			name:            "whitespace then newline",
			input:           "  \t\n",
			expectedAdvance: 4,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "whitespace then semicolon",
			input:           "  \t;",
			expectedAdvance: 4,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   true,
		},
		{
			name:            "line comment",
			input:           "// this is a comment\n",
			expectedAdvance: 21,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "whitespace then line comment",
			input:           "  // comment\n",
			expectedAdvance: 13,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "block comment",
			input:           "/* comment */\n",
			expectedAdvance: 14,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "block comment with newlines inside",
			input:           "/* line1\nline2\nline3 */\n",
			expectedAdvance: 24,
			expectedNewlines: 3,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "multiple block comments",
			input:           "/* first */ /* second */\n",
			expectedAdvance: 25,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
		{
			name:            "whitespace, block comment, semicolon",
			input:           "  /* comment */;",
			expectedAdvance: 16,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   true,
		},
		{
			name:            "whitespace only at end of input",
			input:           "   ",
			expectedAdvance: 3,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   true,
		},
		{
			name:            "block comment at end of input",
			input:           "/* comment */",
			expectedAdvance: 13,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   true,
		},
		{
			name:            "no boundary found",
			input:           "x := 5",
			expectedAdvance: 0,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   false,
		},
		{
			name:            "boundary after some code",
			input:           "x := 5; y := 6",
			expectedAdvance: 0,
			expectedNewlines: 0,
			expectedEndsWithNewline: false,
			expectedFound:   false,
		},
		{
			name:            "complex case: whitespace, block comment, line comment",
			input:           "  /* block */ // line comment\n",
			expectedAdvance: 30,
			expectedNewlines: 1,
			expectedEndsWithNewline: true,
			expectedFound:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findStatementBoundary([]byte(tt.input))
			
			if result.AdvanceBytes != tt.expectedAdvance {
				t.Errorf("AdvanceBytes = %d, want %d", result.AdvanceBytes, tt.expectedAdvance)
			}
			
			if result.NewlineCount != tt.expectedNewlines {
				t.Errorf("NewlineCount = %d, want %d", result.NewlineCount, tt.expectedNewlines)
			}
			
			if result.EndsWithNewline != tt.expectedEndsWithNewline {
				t.Errorf("EndsWithNewline = %v, want %v", result.EndsWithNewline, tt.expectedEndsWithNewline)
			}
			
			if result.BoundaryFound != tt.expectedFound {
				t.Errorf("BoundaryFound = %v, want %v", result.BoundaryFound, tt.expectedFound)
			}
		})
	}
}