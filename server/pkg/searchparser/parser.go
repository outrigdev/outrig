// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

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
		var tokenType string = searchType
		
		// Check if this is a double quoted token
		if p.ch == '"' {
			token = p.readQuotedToken()
		} else if p.ch == '\'' {
			// Single quoted tokens are exactcase
			token = p.readSingleQuotedToken()
			tokenType = "exactcase"
		} else {
			token = p.readToken()
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
