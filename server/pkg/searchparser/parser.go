// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Search Parser Grammar (EBNF):
//
// search           = or_expr ;
// or_expr          = and_expr { "|" and_expr } ;
// and_expr         = { token } ;
// token            = not_token | field_token ;
// not_token        = "-" field_token ;
// field_token      = field_prefix? unmodified_token ;
// field_prefix     = "$" { any_char - ":" - whitespace } ":" ;
// unmodified_token = fuzzy_token | regexp_token | case_regexp_token | tag_token | simple_token ;
// fuzzy_token      = "~" simple_token ;
// regexp_token     = "/" { any_char - "/" | "\/" } "/" ;
// case_regexp_token = "c/" { any_char - "/" | "\/" } "/" ;
// tag_token        = "#" { any_char - whitespace } [ "/" ] ;
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
// - Empty hash prefix (#) followed by whitespace is ignored (no token)
// - Single quoted tokens are treated as case-sensitive (exactcase)
// - Fuzzy tokens with single quotes (~'...') are treated as case-sensitive fuzzy search (fzfcase)
// - Regular expression tokens (/foo/) are case-insensitive by default
// - Case-sensitive regular expression tokens (c/Foo/) are prefixed with 'c'
// - Tag tokens (#foo) search for tags that start at word boundaries
// - Tag tokens with trailing slash (#foo/) require exact matches
// - Special case: #marked or #m uses the marked searcher to find marked lines
// - Not token (-) negates the search result of the token that follows it
// - A literal "-" at the start of a token must be quoted: "-hello" searches for "-hello" literally

package searchparser

import (
	"strings"
	"unicode"
)

// SearchToken represents a single token in a search query
type SearchToken struct {
	Type       string // The search type (exact, regexp, fzf, etc.)
	SearchTerm string // The actual search term
	IsNot      bool   // Whether this token is negated (e.g., -hello)
	Field      string // The field to search in (e.g., name, source)
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

// peek performs n-character lookahead and returns the character at position + n
// Returns 0 (EOF) if the position is out of range
func (p *Parser) peek(n int) rune {
	pos := p.readPosition + n - 1
	if pos >= len(p.input) {
		return 0 // EOF
	}
	return rune(p.input[pos])
}

// skipWhitespace skips any whitespace characters
func (p *Parser) skipWhitespace() {
	for unicode.IsSpace(p.ch) {
		p.readChar()
	}
}

// readToken reads a token (any sequence of non-whitespace characters)
// If splitOnPipe is true, it will stop at a pipe character
func (p *Parser) readToken(splitOnPipe bool) string {
	position := p.position

	// Read until whitespace, pipe (if splitOnPipe is true), or EOF
	for !unicode.IsSpace(p.ch) && p.ch != 0 && !(splitOnPipe && p.ch == '|') {
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
func (p *Parser) parseSimpleToken() (string, string, bool) {
	var token string
	var tokenType string = "exact" // Default type is "exact"

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
		// For plain tokens, we want to split on pipe characters
		token = p.readToken(true)
	}

	return token, tokenType, true
}

// parseFieldPrefix parses a field prefix in the form of "$fieldname:"
func (p *Parser) parseFieldPrefix() (string, bool) {
	// Check for field indicator ($)
	if p.ch != '$' {
		return "", false
	}

	// Skip the $ character
	p.readChar()

	// If we've reached the end of the input or whitespace, return empty
	if p.ch == 0 || unicode.IsSpace(p.ch) {
		return "", false
	}

	position := p.position

	// Read until colon or whitespace or EOF
	for p.ch != ':' && !unicode.IsSpace(p.ch) && p.ch != 0 {
		p.readChar()
	}

	// If we didn't find a colon, this isn't a valid field prefix
	if p.ch != ':' {
		return "", false
	}

	fieldName := p.input[position:p.position]

	// Skip the colon
	p.readChar()

	return fieldName, true
}

// parseUnmodifiedToken parses a token that is not negated (not preceded by -)
// This includes fuzzy tokens, regexp tokens, hash tokens, and simple tokens
func (p *Parser) parseUnmodifiedToken() (SearchToken, bool) {
	var token string
	var tokenType string

	// Check for fuzzy search indicator (~)
	if p.ch == '~' {
		// Skip the ~ character
		p.readChar()

		// If we've reached the end of the input or whitespace, skip this token
		if p.ch == 0 || unicode.IsSpace(p.ch) {
			return SearchToken{}, false
		}

		// Parse the simple token
		simpleToken, simpleType, valid := p.parseSimpleToken()
		if !valid {
			return SearchToken{}, false
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
			return SearchToken{}, false
		}

		tokenType = "regexp"
	} else if p.ch == 'c' && p.peek(1) == '/' {
		// Handle case-sensitive regexp token (c/Foo/)
		// Skip the 'c' character
		p.readChar()

		token = p.readRegexpToken()

		// Skip empty regexp
		if token == "" {
			return SearchToken{}, false
		}

		tokenType = "regexpcase"
	} else if p.ch == '#' {
		// Handle # special character
		// Skip the # character
		p.readChar()

		// If we've reached the end of the input or whitespace, create a token for just "#"
		if p.ch == 0 || unicode.IsSpace(p.ch) {
			tokenType = "exact" // Default to exact search
			token = "#"
		} else {
			// Read the token after #
			position := p.position

			for !unicode.IsSpace(p.ch) && p.ch != 0 {
				// Check for trailing slash indicating exact match
				if p.ch == '/' && (p.peek(1) == 0 || unicode.IsSpace(p.peek(1))) {
					p.readChar() // Consume the slash
					break
				}
				p.readChar()
			}

			token = p.input[position:p.position]

			// Special case for #marked or #userquery
			if strings.ToLower(token) == "marked" || strings.ToLower(token) == "m" {
				tokenType = "marked"
				token = "" // The marked searcher doesn't need a search term
			} else if strings.ToLower(token) == "userquery" {
				tokenType = "userquery"
				token = "" // The userquery searcher doesn't need a search term
			} else {
				// For tag tokens, use the tag searcher
				tokenType = "tag" // Use tag search
				// The exactMatch flag will be passed to the tag searcher
				// We don't need to modify the token here
			}
		}
	} else {
		// Parse a regular simple token
		var valid bool
		token, tokenType, valid = p.parseSimpleToken()
		if !valid {
			return SearchToken{}, false
		}
	}

	return SearchToken{
		Type:       tokenType,
		SearchTerm: token,
		IsNot:      false,
	}, true
}

// parseNotToken parses a token that is negated (preceded by -)
func (p *Parser) parseNotToken() (SearchToken, bool) {
	// Skip the - character
	p.readChar()

	// If we've reached the end of the input or whitespace, treat '-' as a literal token
	if p.ch == 0 || unicode.IsSpace(p.ch) {
		return SearchToken{
			Type:       "exact", // Default to exact search
			SearchTerm: "-",
			IsNot:      false,
		}, true
	}

	// Check for field prefix
	field, hasField := p.parseFieldPrefix()

	// Parse the unmodified token
	unmodifiedToken, valid := p.parseUnmodifiedToken()
	if !valid {
		return SearchToken{}, false
	}

	if hasField {
		unmodifiedToken.Field = field
	}

	// Set the IsNot flag to true
	unmodifiedToken.IsNot = true

	return unmodifiedToken, true
}

// parseToken parses a single token, which can be either a not_token or an unmodified_token
func (p *Parser) parseToken() (SearchToken, bool) {
	// Check for not operator (-)
	if p.ch == '-' {
		return p.parseNotToken()
	}

	// Check for field prefix
	field, hasField := p.parseFieldPrefix()

	// Parse an unmodified token
	token, valid := p.parseUnmodifiedToken()
	if !valid {
		return SearchToken{}, false
	}

	if hasField {
		token.Field = field
	}

	return token, true
}

// parseAndExpr parses a sequence of tokens (AND expression)
func (p *Parser) parseAndExpr() []SearchToken {
	var tokens []SearchToken

	for p.ch != 0 && p.ch != '|' {
		// Skip whitespace
		p.skipWhitespace()

		// If we've reached the end of the input or a pipe, break
		if p.ch == 0 || p.ch == '|' {
			break
		}

		// Parse a token
		token, valid := p.parseToken()
		if !valid {
			continue
		}

		// Add the token to the result
		tokens = append(tokens, token)
	}

	return tokens
}

// parseOrExpr parses an OR expression (and_expr { "|" and_expr })
func (p *Parser) parseOrExpr() [][]SearchToken {
	var orGroups [][]SearchToken

	// Handle the case where the expression starts with a pipe
	if p.ch == '|' {
		// Skip the "|" character
		p.readChar()

		// Skip any whitespace after the "|"
		p.skipWhitespace()

		// Add an empty group before the pipe
		orGroups = append(orGroups, []SearchToken{})

		// Parse the expression after the pipe
		andTokens := p.parseAndExpr()
		orGroups = append(orGroups, andTokens)

		// Continue parsing if there are more pipes
		for p.ch == '|' {
			// Skip the "|" character
			p.readChar()

			// Skip any whitespace after the "|"
			p.skipWhitespace()

			// Parse the next AND expression
			andTokens = p.parseAndExpr()
			orGroups = append(orGroups, andTokens)
		}

		return orGroups
	}

	// Parse the first AND expression
	andTokens := p.parseAndExpr()
	orGroups = append(orGroups, andTokens)

	// Parse additional AND expressions separated by "|"
	for p.ch == '|' {
		// Skip the "|" character
		p.readChar()

		// Skip any whitespace after the "|"
		p.skipWhitespace()

		// Parse the next AND expression
		andTokens = p.parseAndExpr()
		orGroups = append(orGroups, andTokens)
	}

	return orGroups
}

// Parse parses the input string into a slice of tokens with OR groups
func (p *Parser) Parse() []SearchToken {
	// Special case for a single pipe character
	if p.ch == '|' && p.peek(1) == 0 {
		return []SearchToken{
			{
				Type:       "or",
				SearchTerm: "|",
				IsNot:      false,
			},
		}
	}

	// Parse the OR expression
	orGroups := p.parseOrExpr()

	// If there's only one group, return it directly
	if len(orGroups) <= 1 {
		if len(orGroups) == 0 {
			return []SearchToken{}
		}
		return orGroups[0]
	}

	// Otherwise, create OR tokens
	var result []SearchToken

	// Add a special token to indicate the start of an OR group
	for i, group := range orGroups {
		// Add a special token to indicate OR operation
		if i > 0 {
			result = append(result, SearchToken{
				Type:       "or",
				SearchTerm: "|",
				IsNot:      false,
			})
		}

		// Add the tokens from this group
		result = append(result, group...)
	}

	return result
}

// TokenizeSearch splits a search string into tokens using the parser
func TokenizeSearch(searchString string) []SearchToken {
	searchString = strings.TrimSpace(searchString)
	if searchString == "" {
		return []SearchToken{}
	}
	parser := NewParser(searchString)
	return parser.Parse()
}
