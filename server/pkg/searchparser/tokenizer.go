// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package searchparser

import (
	"unicode"
)

// TokenType represents the type of token
type TokenType string

const (
	// Token types for complex tokens
	TokenWord       TokenType = "WORD"    // Plain word token
	TokenDQuote     TokenType = "DQUOTE"  // Double quoted string
	TokenSQuote     TokenType = "SQUOTE"  // Single quoted string
	TokenRegexp     TokenType = "REGEXP"  // Regular expression
	TokenCRegexp    TokenType = "CREGEXP" // Case-sensitive regexp
	TokenWhitespace TokenType = "WS"      // Whitespace
	TokenEOF        TokenType = "EOF"     // End of input

	// Token types for simple characters (using the actual character)
	TokenLParen TokenType = "(" // Left parenthesis
	TokenRParen TokenType = ")" // Right parenthesis
	TokenPipe   TokenType = "|" // Pipe character
	TokenMinus  TokenType = "-" // Minus sign
	TokenDollar TokenType = "$" // Dollar sign
	TokenTilde  TokenType = "~" // Tilde
	TokenHash   TokenType = "#" // Hash
)

// Token represents a token in the search expression
type Token struct {
	Type       TokenType // Type of the token
	Value      string    // Value of the token
	Position   Position  // Position in the source
	Incomplete bool      // True if the token is incomplete (e.g., unterminated string)
}

// Tokenizer represents a lexical analyzer for search expressions
type Tokenizer struct {
	input        string // Input string
	position     int    // Current position in input (points to current char)
	readPosition int    // Current reading position in input (after current char)
	ch           rune   // Current character
}

// NewTokenizer creates a new tokenizer for the given input
func NewTokenizer(input string) *Tokenizer {
	t := &Tokenizer{input: input}
	t.initialize()
	return t
}

// initialize sets up the tokenizer
func (t *Tokenizer) initialize() {
	t.position = 0
	t.readPosition = 0
	t.readChar()
}

// readChar reads the next character from the input
func (t *Tokenizer) readChar() {
	if t.readPosition >= len(t.input) {
		t.ch = 0 // EOF
	} else {
		t.ch = rune(t.input[t.readPosition])
	}
	t.position = t.readPosition
	t.readPosition++
}

// peek performs lookahead and returns the next character
func (t *Tokenizer) peek() rune {
	if t.readPosition >= len(t.input) {
		return 0 // EOF
	}
	return rune(t.input[t.readPosition])
}

// readWhitespace reads a continuous sequence of whitespace
func (t *Tokenizer) readWhitespace() string {
	startPos := t.position

	for unicode.IsSpace(t.ch) {
		t.readChar()
	}

	return t.input[startPos:t.position]
}

// NextToken returns the next token from the input
func (t *Tokenizer) NextToken() Token {
	var tok Token

	startPos := t.position

	switch {
	case unicode.IsSpace(t.ch):
		value := t.readWhitespace()
		tok = Token{Type: TokenWhitespace, Value: value, Position: Position{Start: startPos, End: t.position}}
	case t.ch == '(':
		tok = Token{Type: "(", Value: "(", Position: Position{Start: startPos, End: t.position + 1}}
		t.readChar()
	case t.ch == ')':
		tok = Token{Type: ")", Value: ")", Position: Position{Start: startPos, End: t.position + 1}}
		t.readChar()
	case t.ch == '|':
		tok = Token{Type: "|", Value: "|", Position: Position{Start: startPos, End: t.position + 1}}
		t.readChar()
	case t.ch == '-':
		tok = Token{Type: "-", Value: "-", Position: Position{Start: startPos, End: t.position + 1}}
		t.readChar()
	case t.ch == '$':
		tok = Token{Type: "$", Value: "$", Position: Position{Start: startPos, End: t.position + 1}}
		t.readChar()
	case t.ch == '~':
		tok = Token{Type: "~", Value: "~", Position: Position{Start: startPos, End: t.position + 1}}
		t.readChar()
	case t.ch == '#':
		tok = Token{Type: "#", Value: "#", Position: Position{Start: startPos, End: t.position + 1}}
		t.readChar()
	case t.ch == '"':
		value, incomplete := t.readDoubleQuotedString()
		tok = Token{
			Type:       TokenDQuote,
			Value:      value,
			Position:   Position{Start: startPos, End: t.position},
			Incomplete: incomplete,
		}
	case t.ch == '\'':
		value, incomplete := t.readSingleQuotedString()
		tok = Token{
			Type:       TokenSQuote,
			Value:      value,
			Position:   Position{Start: startPos, End: t.position},
			Incomplete: incomplete,
		}
	case t.ch == '/':
		value, incomplete := t.readRegexpString()
		tok = Token{
			Type:       TokenRegexp,
			Value:      value,
			Position:   Position{Start: startPos, End: t.position},
			Incomplete: incomplete,
		}
	case t.ch == 0:
		tok = Token{Type: TokenEOF, Value: "", Position: Position{Start: startPos, End: startPos}}
	default:
		if t.ch == 'c' && t.peek() == '/' {
			// Handle case-sensitive regexp
			t.readChar() // Skip 'c'
			if t.ch == '/' {
				value, incomplete := t.readRegexpString()
				tok = Token{
					Type:       TokenCRegexp,
					Value:      value,
					Position:   Position{Start: startPos, End: t.position},
					Incomplete: incomplete,
				}
			} else {
				// Just a 'c' followed by something else
				value := t.readWord()
				tok = Token{Type: TokenWord, Value: "c" + value, Position: Position{Start: startPos, End: t.position}}
			}
		} else {
			// Read a word token
			value := t.readWord()
			tok = Token{Type: TokenWord, Value: value, Position: Position{Start: startPos, End: t.position}}
		}
	}

	return tok
}

// readUntilDelimiter reads characters until it finds an unescaped delimiter
// Returns the content and a boolean indicating if the reading is incomplete
func (t *Tokenizer) readUntilDelimiter(delimiter rune) (string, bool) {
	startPos := t.position
	escaped := false

	for {
		if t.ch == 0 {
			// EOF
			break
		}

		if escaped {
			// Previous character was a backslash, so this character is escaped
			escaped = false
			t.readChar()
			continue
		}

		if t.ch == '\\' {
			// Backslash - next character will be escaped
			escaped = true
			t.readChar()
			continue
		}

		if t.ch == delimiter {
			// Unescaped delimiter found
			break
		}

		// Regular character
		t.readChar()
	}

	value := t.input[startPos:t.position]
	incomplete := t.ch == 0 // True if we reached EOF without closing delimiter

	if t.ch == delimiter {
		t.readChar() // Skip closing delimiter
	}

	return value, incomplete
}

// readDoubleQuotedString reads a string enclosed in double quotes
// Returns the content and a boolean indicating if the string is incomplete
func (t *Tokenizer) readDoubleQuotedString() (string, bool) {
	t.readChar() // Skip opening quote
	return t.readUntilDelimiter('"')
}

// readSingleQuotedString reads a string enclosed in single quotes
// Returns the content and a boolean indicating if the string is incomplete
func (t *Tokenizer) readSingleQuotedString() (string, bool) {
	t.readChar() // Skip opening quote
	return t.readUntilDelimiter('\'')
}

// readRegexpString reads a regexp enclosed in slashes
// Returns the content and a boolean indicating if the regexp is incomplete
func (t *Tokenizer) readRegexpString() (string, bool) {
	t.readChar() // Skip opening slash
	return t.readUntilDelimiter('/')
}

// readWord reads a word token (any sequence of non-word-break characters)
func (t *Tokenizer) readWord() string {
	startPos := t.position

	for {
		// Stop at EOF or word break characters
		if t.ch == 0 || isWordBreakChar(t.ch) {
			break
		}
		t.readChar()
	}

	return t.input[startPos:t.position]
}

// isWordBreakChar returns true if the character breaks a word
func isWordBreakChar(ch rune) bool {
	return unicode.IsSpace(ch) ||
		ch == '(' || ch == ')' || ch == '|' ||
		ch == '"' || ch == '\''
}

// GetAllTokens tokenizes the entire input and returns all tokens
func (t *Tokenizer) GetAllTokens() []Token {
	var tokens []Token

	for {
		tok := t.NextToken()
		tokens = append(tokens, tok)

		if tok.Type == TokenEOF {
			break
		}
	}

	return tokens
}

// TokensToString converts a slice of tokens to a string representation
func TokensToString(tokens []Token) string {
	var result string
	for _, tok := range tokens {
		result += tok.Value
	}
	return result
}
