// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Search Parser Grammar (EBNF):
//
// search           = { token } ;
// token            = fuzzy_token | regexp_token | case_regexp_token | simple_token ;
// fuzzy_token      = "~" simple_token ;
// regexp_token     = "/" { any_char - "/" | "\/" } "/" ;
// case_regexp_token = "c/" { any_char - "/" | "\/" } "/" ;
// simple_token     = quoted_token | single_quoted_token | plain_token ;
// quoted_token     = '"' { any_char - '"' } '"' ;
// single_quoted_token = "'" { any_char - "'" } "'" ;
// plain_token      = { any_char - whitespace } ;
// any_char         = ? any Unicode character ? ;
// whitespace       = ? Unicode whitespace character ? ;
//
// Notes:
// - Empty quoted strings ("" or '') are ignored (no token)
// - Empty fuzzy prefix (~) followed by whitespace is ignored (no token)
// - Single quoted tokens are treated as case-sensitive (exactcase)
// - Fuzzy tokens with single quotes (~'...') are treated as case-sensitive fuzzy search (fzfcase)
// - Regular expression tokens (/foo/) are case-insensitive by default
// - Case-sensitive regular expression tokens (c/Foo/) are prefixed with 'c'

package searchparser

import (
	"strings"
	"unicode"
)

// SearchToken represents a single token in a search query
type SearchToken struct {
	Type       string // The search type (exact, regexp, fzf, etc.)
	SearchTerm string // The actual search term
}

// Parser represents a recursive descent parser for search expressions
type Parser struct {
	input        string
	position     int
	readPosition int
	ch           rune
}

// NewParser creates a new parser for the given search expression
func NewParser(input string) *Parser {
	p := &Parser{input: input}
	p.initialize()
	return p
}

// initialize sets up the parser
func (p *Parser) initialize() {
	p.position = 0
	p.readPosition = 0

	// Read the first character
	p.readChar()
}

// readChar reads the next character from the input
func (p *Parser) readChar() {
	if p.readPosition >= len(p.input) {
		p.ch = 0 // EOF
	} else {
		p.ch = rune(p.input[p.readPosition])
	}
	p.position = p.readPosition
	p.readPosition++
}

// skipWhitespace skips any whitespace characters
func (p *Parser) skipWhitespace() {
	for unicode.IsSpace(p.ch) {
		p.readChar()
	}
}

// readToken reads a token (any sequence of non-whitespace characters)
func (p *Parser) readToken() string {
	position := p.position

	// Read until whitespace or EOF
	for !unicode.IsSpace(p.ch) && p.ch != 0 {
		p.readChar()
	}

	return p.input[position:p.position]
}

// readQuotedToken reads a token enclosed in double quotes
// If the closing quote is missing, it reads until the end of the input
func (p *Parser) readQuotedToken() string {
	// Skip the opening quote
	p.readChar()

	position := p.position

	// Read until closing quote or EOF
	for p.ch != '"' && p.ch != 0 {
		p.readChar()
	}

	// Store the token content
	token := p.input[position:p.position]

	// Skip the closing quote if present
	if p.ch == '"' {
		p.readChar()
	}

	return token
}

// readSingleQuotedToken reads a token enclosed in single quotes
// If the closing quote is missing, it reads until the end of the input
// Single quoted tokens preserve case (exactcase)
func (p *Parser) readSingleQuotedToken() string {
	// Skip the opening quote
	p.readChar()

	position := p.position

	// Read until closing quote or EOF
	for p.ch != '\'' && p.ch != 0 {
		p.readChar()
	}

	// Store the token content
	token := p.input[position:p.position]

	// Skip the closing quote if present
	if p.ch == '\'' {
		p.readChar()
	}

	return token
}

// readRegexpToken reads a token enclosed in slashes (/)
// Handles escaped slashes (\/) within the regexp
// If the closing slash is missing, it reads until the end of the input
func (p *Parser) readRegexpToken() string {
	// Skip the opening slash
	p.readChar()

	position := p.position
	escaped := false

	// Read until closing slash or EOF, handling escaped slashes
	for {
		if p.ch == 0 {
			// EOF
			break
		}

		if escaped {
			// Previous character was a backslash, so this character is escaped
			escaped = false
			p.readChar()
			continue
		}

		if p.ch == '\\' {
			// Backslash - next character will be escaped
			escaped = true
			p.readChar()
			continue
		}

		if p.ch == '/' {
			// Unescaped closing slash
			break
		}

		// Regular character
		p.readChar()
	}

	// Store the token content
	token := p.input[position:p.position]

	// Skip the closing slash if present
	if p.ch == '/' {
		p.readChar()
	}

	return token
}

// parseSimpleToken parses a simple token (quoted, single-quoted, or plain)
func (p *Parser) parseSimpleToken(defaultType string) (string, string, bool) {
	var token string
	var tokenType string = defaultType

	if p.ch == '"' {
		// Double quoted tokens
		token = p.readQuotedToken()
		// Skip empty quoted strings
		if token == "" {
			return "", "", false
		}
	} else if p.ch == '\'' {
		// Single quoted tokens are exactcase
		token = p.readSingleQuotedToken()
		// Skip empty quoted strings
		if token == "" {
			return "", "", false
		}
		tokenType = "exactcase"
	} else {
		token = p.readToken()
	}

	return token, tokenType, true
}

// Parse parses the input string into a slice of tokens
func (p *Parser) Parse(searchType string) []SearchToken {
	var tokens []SearchToken

	for p.ch != 0 {
		// Skip whitespace
		p.skipWhitespace()

		// If we've reached the end of the input, break
		if p.ch == 0 {
			break
		}

		var token string
		var tokenType string

		// Check for fuzzy search indicator (~)
		if p.ch == '~' {
			// Skip the ~ character
			p.readChar()

			// If we've reached the end of the input or whitespace, skip this token
			if p.ch == 0 || unicode.IsSpace(p.ch) {
				continue
			}

			// Parse the simple token
			simpleToken, simpleType, valid := p.parseSimpleToken(searchType)
			if !valid {
				continue
			}

			// Convert to fuzzy search type
			if simpleType == "exactcase" {
				tokenType = "fzfcase"
			} else {
				tokenType = "fzf"
			}
			token = simpleToken
		} else if p.ch == '/' {
			// Handle regexp token (case-insensitive by default)
			token = p.readRegexpToken()

			// Skip empty regexp
			if token == "" {
				continue
			}

			tokenType = "regexp"
		} else if p.ch == 'c' && p.readPosition < len(p.input) && p.input[p.readPosition] == '/' {
			// Handle case-sensitive regexp token (c/Foo/)
			// Skip the 'c' character
			p.readChar()
			
			token = p.readRegexpToken()

			// Skip empty regexp
			if token == "" {
				continue
			}

			tokenType = "regexpcase"
		} else {
			// Parse a regular simple token
			var valid bool
			token, tokenType, valid = p.parseSimpleToken(searchType)
			if !valid {
				continue
			}
		}

		// Add the token to the result
		tokens = append(tokens, SearchToken{
			Type:       tokenType,
			SearchTerm: token,
		})
	}

	return tokens
}

// TokenizeSearch splits a search string into tokens using the parser
func TokenizeSearch(searchType string, searchString string) []SearchToken {
	searchString = strings.TrimSpace(searchString)
	if searchString == "" {
		return []SearchToken{}
	}
	parser := NewParser(searchString)
	return parser.Parse(searchType)
}
