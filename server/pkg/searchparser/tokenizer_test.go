// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package searchparser

import (
	"testing"
)

func TestTokenizer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "empty string",
			input: "",
			expected: []Token{
				{Type: TokenEOF, Value: "", Position: Position{Start: 0, End: 0}},
			},
		},
		{
			name:  "single word",
			input: "hello",
			expected: []Token{
				{Type: TokenWord, Value: "hello", Position: Position{Start: 0, End: 5}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 5, End: 5}},
			},
		},
		{
			name:  "two words with whitespace",
			input: "hello world",
			expected: []Token{
				{Type: TokenWord, Value: "hello", Position: Position{Start: 0, End: 5}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 5, End: 6}},
				{Type: TokenWord, Value: "world", Position: Position{Start: 6, End: 11}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 11, End: 11}},
			},
		},
		{
			name:  "special characters",
			input: "( ) | - $ : ~ #",
			expected: []Token{
				{Type: "(", Value: "(", Position: Position{Start: 0, End: 1}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 1, End: 2}},
				{Type: ")", Value: ")", Position: Position{Start: 2, End: 3}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 3, End: 4}},
				{Type: "|", Value: "|", Position: Position{Start: 4, End: 5}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 5, End: 6}},
				{Type: "-", Value: "-", Position: Position{Start: 6, End: 7}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 7, End: 8}},
				{Type: "$", Value: "$", Position: Position{Start: 8, End: 9}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 9, End: 10}},
				{Type: ":", Value: ":", Position: Position{Start: 10, End: 11}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 11, End: 12}},
				{Type: "~", Value: "~", Position: Position{Start: 12, End: 13}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 13, End: 14}},
				{Type: "#", Value: "#", Position: Position{Start: 14, End: 15}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 15, End: 15}},
			},
		},
		{
			name:  "double quoted string",
			input: `"hello world"`,
			expected: []Token{
				{Type: TokenDQuote, Value: "hello world", Position: Position{Start: 0, End: 13}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 13, End: 13}},
			},
		},
		{
			name:  "single quoted string",
			input: `'hello world'`,
			expected: []Token{
				{Type: TokenSQuote, Value: "hello world", Position: Position{Start: 0, End: 13}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 13, End: 13}},
			},
		},
		{
			name:  "regular expression",
			input: `/hello\s+world/`,
			expected: []Token{
				{Type: TokenRegexp, Value: "hello\\s+world", Position: Position{Start: 0, End: 15}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 15, End: 15}},
			},
		},
		{
			name:  "case-sensitive regular expression",
			input: `c/Hello\s+World/`,
			expected: []Token{
				{Type: TokenCRegexp, Value: "Hello\\s+World", Position: Position{Start: 0, End: 16}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 16, End: 16}},
			},
		},
		{
			name:  "incomplete double quoted string",
			input: `"hello world`,
			expected: []Token{
				{Type: TokenDQuote, Value: "hello world", Position: Position{Start: 0, End: 12}, Incomplete: true},
				{Type: TokenEOF, Value: "", Position: Position{Start: 12, End: 12}},
			},
		},
		{
			name:  "incomplete single quoted string",
			input: `'hello world`,
			expected: []Token{
				{Type: TokenSQuote, Value: "hello world", Position: Position{Start: 0, End: 12}, Incomplete: true},
				{Type: TokenEOF, Value: "", Position: Position{Start: 12, End: 12}},
			},
		},
		{
			name:  "incomplete regular expression",
			input: `/hello\s+world`,
			expected: []Token{
				{Type: TokenRegexp, Value: "hello\\s+world", Position: Position{Start: 0, End: 14}, Incomplete: true},
				{Type: TokenEOF, Value: "", Position: Position{Start: 14, End: 14}},
			},
		},
		{
			name:  "field prefix with colon",
			input: "$field:value",
			expected: []Token{
				{Type: "$", Value: "$", Position: Position{Start: 0, End: 1}},
				{Type: TokenWord, Value: "field", Position: Position{Start: 1, End: 6}},
				{Type: ":", Value: ":", Position: Position{Start: 6, End: 7}},
				{Type: TokenWord, Value: "value", Position: Position{Start: 7, End: 12}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 12, End: 12}},
			},
		},
		{
			name:  "field prefix with whitespace",
			input: "$ field : value",
			expected: []Token{
				{Type: "$", Value: "$", Position: Position{Start: 0, End: 1}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 1, End: 2}},
				{Type: TokenWord, Value: "field", Position: Position{Start: 2, End: 7}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 7, End: 8}},
				{Type: ":", Value: ":", Position: Position{Start: 8, End: 9}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 9, End: 10}},
				{Type: TokenWord, Value: "value", Position: Position{Start: 10, End: 15}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 15, End: 15}},
			},
		},
		{
			name:  "not token",
			input: "-hello",
			expected: []Token{
				{Type: "-", Value: "-", Position: Position{Start: 0, End: 1}},
				{Type: TokenWord, Value: "hello", Position: Position{Start: 1, End: 6}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 6, End: 6}},
			},
		},
		{
			name:  "not token with whitespace",
			input: "- hello",
			expected: []Token{
				{Type: "-", Value: "-", Position: Position{Start: 0, End: 1}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 1, End: 2}},
				{Type: TokenWord, Value: "hello", Position: Position{Start: 2, End: 7}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 7, End: 7}},
			},
		},
		{
			name:  "fuzzy token",
			input: "~hello",
			expected: []Token{
				{Type: "~", Value: "~", Position: Position{Start: 0, End: 1}},
				{Type: TokenWord, Value: "hello", Position: Position{Start: 1, End: 6}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 6, End: 6}},
			},
		},
		{
			name:  "tag token",
			input: "#hello",
			expected: []Token{
				{Type: "#", Value: "#", Position: Position{Start: 0, End: 1}},
				{Type: TokenWord, Value: "hello", Position: Position{Start: 1, End: 6}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 6, End: 6}},
			},
		},
		{
			name:  "parentheses grouping",
			input: "hello(mike)foo",
			expected: []Token{
				{Type: TokenWord, Value: "hello", Position: Position{Start: 0, End: 5}},
				{Type: "(", Value: "(", Position: Position{Start: 5, End: 6}},
				{Type: TokenWord, Value: "mike", Position: Position{Start: 6, End: 10}},
				{Type: ")", Value: ")", Position: Position{Start: 10, End: 11}},
				{Type: TokenWord, Value: "foo", Position: Position{Start: 11, End: 14}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 14, End: 14}},
			},
		},
		{
			name:  "complex expression with parentheses",
			input: "hello (mike | john) -foo",
			expected: []Token{
				{Type: TokenWord, Value: "hello", Position: Position{Start: 0, End: 5}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 5, End: 6}},
				{Type: "(", Value: "(", Position: Position{Start: 6, End: 7}},
				{Type: TokenWord, Value: "mike", Position: Position{Start: 7, End: 11}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 11, End: 12}},
				{Type: "|", Value: "|", Position: Position{Start: 12, End: 13}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 13, End: 14}},
				{Type: TokenWord, Value: "john", Position: Position{Start: 14, End: 18}},
				{Type: ")", Value: ")", Position: Position{Start: 18, End: 19}},
				{Type: TokenWhitespace, Value: " ", Position: Position{Start: 19, End: 20}},
				{Type: "-", Value: "-", Position: Position{Start: 20, End: 21}},
				{Type: TokenWord, Value: "foo", Position: Position{Start: 21, End: 24}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 24, End: 24}},
			},
		},
		{
			name:  "regexp with whitespace",
			input: "/ hello /",
			expected: []Token{
				{Type: TokenRegexp, Value: " hello ", Position: Position{Start: 0, End: 9}},
				{Type: TokenEOF, Value: "", Position: Position{Start: 9, End: 9}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens := tokenizer.GetAllTokens()

			if len(tokens) != len(tt.expected) {
				t.Errorf("Token count mismatch: got %d, want %d", len(tokens), len(tt.expected))
				for i, tok := range tokens {
					t.Logf("Token %d: %+v", i, tok)
				}
				return
			}

			for i, expectedToken := range tt.expected {
				actualToken := tokens[i]

				if actualToken.Type != expectedToken.Type {
					t.Errorf("Token[%d] type mismatch: got %s, want %s", i, actualToken.Type, expectedToken.Type)
				}

				if actualToken.Value != expectedToken.Value {
					t.Errorf("Token[%d] value mismatch: got %q, want %q", i, actualToken.Value, expectedToken.Value)
				}

				if actualToken.Position.Start != expectedToken.Position.Start {
					t.Errorf("Token[%d] position start mismatch: got %d, want %d", i, actualToken.Position.Start, expectedToken.Position.Start)
				}

				if actualToken.Position.End != expectedToken.Position.End {
					t.Errorf("Token[%d] position end mismatch: got %d, want %d", i, actualToken.Position.End, expectedToken.Position.End)
				}

				if actualToken.Incomplete != expectedToken.Incomplete {
					t.Errorf("Token[%d] incomplete flag mismatch: got %t, want %t", i, actualToken.Incomplete, expectedToken.Incomplete)
				}
			}
		})
	}
}

func TestTokensToString(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []Token
		expected string
	}{
		{
			name: "empty tokens",
			tokens: []Token{
				{Type: TokenEOF, Value: ""},
			},
			expected: "",
		},
		{
			name: "single word",
			tokens: []Token{
				{Type: TokenWord, Value: "hello"},
				{Type: TokenEOF, Value: ""},
			},
			expected: "hello",
		},
		{
			name: "multiple tokens",
			tokens: []Token{
				{Type: TokenWord, Value: "hello"},
				{Type: TokenWhitespace, Value: " "},
				{Type: TokenWord, Value: "world"},
				{Type: TokenEOF, Value: ""},
			},
			expected: "hello world",
		},
		{
			name: "complex expression",
			tokens: []Token{
				{Type: TokenWord, Value: "hello"},
				{Type: TokenWhitespace, Value: " "},
				{Type: "(", Value: "("},
				{Type: TokenWord, Value: "mike"},
				{Type: TokenWhitespace, Value: " "},
				{Type: "|", Value: "|"},
				{Type: TokenWhitespace, Value: " "},
				{Type: TokenWord, Value: "john"},
				{Type: ")", Value: ")"},
				{Type: TokenEOF, Value: ""},
			},
			expected: "hello (mike | john)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TokensToString(tt.tokens)
			if result != tt.expected {
				t.Errorf("TokensToString() = %q, want %q", result, tt.expected)
			}
		})
	}
}
