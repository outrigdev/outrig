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
)

const (
	NodeTypeSearch = "search"
	NodeTypeAnd    = "and"
	NodeTypeOr     = "or"
)

// Position represents a position in the source text
type Position struct {
	Start int // Start position (inclusive)
	End   int // End position (exclusive)
}

// Node represents a node in the search AST
type Node struct {
	// Common fields for all nodes
	Type     string   // NodeTypeAnd, NodeTypeOr, NodeTypeSearch
	Position Position // Position in source text

	// Union fields - used based on node type
	Children []Node // Used for NodeTypeAnd, NodeTypeOr nodes

	// Fields primarily for leaf nodes (search terms)
	SearchType string // "exact", "regexp", "fzf", etc. (only for NodeTypeSearch)
	SearchTerm string // The actual search text (only for NodeTypeSearch)
	Field      string // Optional field specifier (only for NodeTypeSearch)
	IsNot      bool   // Whether this is a negated search
}

// Parser represents a recursive descent parser for search expressions
type Parser struct {
	tokens   []Token
	position int
	input    string // Keep original input for error reporting
}

// NewParser creates a new parser for the given search expression
func NewParser(input string) *Parser {
	tokenizer := NewTokenizer(input)
	tokens := tokenizer.GetAllTokens()

	return &Parser{
		tokens:   tokens,
		position: 0,
		input:    input,
	}
}

// currentToken returns the current token
func (p *Parser) currentToken() Token {
	if p.position >= len(p.tokens) {
		return Token{Type: TokenEOF, Value: "", Position: Position{Start: len(p.input), End: len(p.input)}}
	}
	return p.tokens[p.position]
}

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.position++
}

// skipWhitespace skips any whitespace tokens
func (p *Parser) skipWhitespace() {
	for p.currentToken().Type == TokenWhitespace {
		p.nextToken()
	}
}

// currentTokenIs checks if the current token is of the specified type
func (p *Parser) currentTokenIs(tokenType TokenType) bool {
	return p.currentToken().Type == tokenType
}

// makeSearchNode creates a search node with the given parameters
func makeSearchNode(searchType, searchTerm, field string, isNot bool, pos Position) *Node {
	return &Node{
		Type:       NodeTypeSearch,
		Position:   pos,
		SearchType: searchType,
		SearchTerm: searchTerm,
		Field:      field,
		IsNot:      isNot,
	}
}

// parseSimpleToken parses a simple token (quoted, single-quoted, or plain)
func (p *Parser) parseSimpleToken() (string, string, bool, Position) {
	var token string
	var tokenType string = "exact" // Default type is "exact"
	startPos := p.currentToken().Position.Start
	endPos := p.currentToken().Position.End

	switch p.currentToken().Type {
	case TokenDoubleQuoted:
		// Double quoted tokens
		token = p.currentToken().Value
		// Skip empty quoted strings
		if token == "" {
			p.nextToken()
			return "", "", false, Position{Start: startPos, End: endPos}
		}
		endPos = p.currentToken().Position.End
		p.nextToken()
	case TokenSingleQuoted:
		// Single quoted tokens are exactcase
		token = p.currentToken().Value
		// Skip empty quoted strings
		if token == "" {
			p.nextToken()
			return "", "", false, Position{Start: startPos, End: endPos}
		}
		tokenType = "exactcase"
		endPos = p.currentToken().Position.End
		p.nextToken()
	case TokenWord:
		// Plain word token
		token = p.currentToken().Value
		endPos = p.currentToken().Position.End
		p.nextToken()
	default:
		// Not a simple token
		return "", "", false, Position{Start: startPos, End: endPos}
	}

	return token, tokenType, true, Position{Start: startPos, End: endPos}
}

// parseFieldPrefix parses a field prefix in the form of "$fieldname:"
// Returns (fieldName, hasField, isComplete, position)
func (p *Parser) parseFieldPrefix() (string, bool, bool, Position) {
	startPos := p.currentToken().Position.Start
	endPos := p.currentToken().Position.End

	// Check for field indicator ($)
	if !p.currentTokenIs("$") {
		return "", false, false, Position{Start: startPos, End: endPos}
	}

	// Skip the $ character
	p.nextToken()

	// If we've reached the end of the input or whitespace, return empty
	if p.currentTokenIs(TokenEOF) || p.currentTokenIs(TokenWhitespace) {
		endPos = p.currentToken().Position.Start
		return "", true, false, Position{Start: startPos, End: endPos}
	}

	// We need a word token for the field name
	if !p.currentTokenIs(TokenWord) {
		endPos = p.currentToken().Position.Start
		return "", true, false, Position{Start: startPos, End: endPos}
	}

	fieldName := p.currentToken().Value
	p.nextToken()
	endPos = p.currentToken().Position.Start

	// If we didn't find a colon, this is an incomplete field prefix
	if !p.currentTokenIs(":") {
		return fieldName, true, false, Position{Start: startPos, End: endPos}
	}

	// Skip the colon
	p.nextToken()
	endPos = p.currentToken().Position.Start

	return fieldName, true, true, Position{Start: startPos, End: endPos}
}

// parseUnmodifiedToken parses a token that is not negated (not preceded by -)
// This includes fuzzy tokens, regexp tokens, hash tokens, and simple tokens
func (p *Parser) parseUnmodifiedToken() *Node {
	startPos := p.currentToken().Position.Start
	var token string
	var tokenType string
	var endPos int

	// Check for fuzzy search indicator (~)
	if p.currentTokenIs("~") {
		// Skip the ~ character
		p.nextToken()
		endPos = p.currentToken().Position.Start

		// If we've reached the end of the input or whitespace, skip this token
		if p.currentTokenIs(TokenEOF) || p.currentTokenIs(TokenWhitespace) {
			return nil
		}

		// Parse the simple token
		simpleToken, simpleType, valid, simplePos := p.parseSimpleToken()
		if !valid {
			return nil
		}

		// Convert to fuzzy search type
		if simpleType == "exactcase" {
			tokenType = "fzfcase"
		} else {
			tokenType = "fzf"
		}
		token = simpleToken
		endPos = simplePos.End
	} else if p.currentTokenIs(TokenRegexp) {
		// Handle regexp token (case-insensitive by default)
		token = p.currentToken().Value
		tokenType = "regexp"
		endPos = p.currentToken().Position.End
		p.nextToken()
	} else if p.currentTokenIs(TokenCaseRegexp) {
		// Handle case-sensitive regexp token (c/Foo/)
		token = p.currentToken().Value
		tokenType = "regexpcase"
		endPos = p.currentToken().Position.End
		p.nextToken()
	} else if p.currentTokenIs("#") {
		// Handle # special character
		// Skip the # character
		p.nextToken()
		endPos = p.currentToken().Position.Start

		// If we've reached the end of the input or whitespace, create a token for just "#"
		if p.currentTokenIs(TokenEOF) || p.currentTokenIs(TokenWhitespace) {
			tokenType = "exact" // Default to exact search
			token = "#"
		} else if p.currentTokenIs(TokenWord) {
			token = p.currentToken().Value
			endPos = p.currentToken().Position.End
			p.nextToken()

			// Check for trailing slash indicating exact match
			if p.currentTokenIs("/") {
				endPos = p.currentToken().Position.End
				p.nextToken()
			}

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
		} else {
			// Invalid token after #
			tokenType = "exact" // Default to exact search
			token = "#"
		}
	} else {
		// Parse a regular simple token
		var valid bool
		var simplePos Position
		token, tokenType, valid, simplePos = p.parseSimpleToken()
		if !valid {
			return nil
		}
		endPos = simplePos.End
	}

	pos := Position{Start: startPos, End: endPos}
	return makeSearchNode(tokenType, token, "", false, pos)
}

// parseNotToken parses a token that is negated (preceded by -)
func (p *Parser) parseNotToken() *Node {
	startPos := p.currentToken().Position.Start

	// Skip the - character
	p.nextToken()
	endPos := p.currentToken().Position.Start

	// If we've reached the end of the input or whitespace, treat '-' as a literal token
	if p.currentTokenIs(TokenEOF) || p.currentTokenIs(TokenWhitespace) {
		pos := Position{Start: startPos, End: endPos}
		return makeSearchNode("exact", "-", "", false, pos)
	}

	// Check for field prefix
	field, hasField, isComplete, fieldPos := p.parseFieldPrefix()

	// If we have a field prefix but it's incomplete, return a token with empty search term
	if hasField && !isComplete {
		pos := Position{Start: startPos, End: fieldPos.End}
		return makeSearchNode("exact", "", field, false, pos)
	}

	// Parse the unmodified token
	node := p.parseUnmodifiedToken()
	if node == nil {
		return nil
	}

	if hasField {
		node.Field = field
	}

	// Set the IsNot flag to true
	node.IsNot = true

	// Update the position to include the '-' prefix
	node.Position.Start = startPos

	return node
}

// parseToken parses a single token, which can be either a not_token or an unmodified_token
func (p *Parser) parseToken() *Node {
	startPos := p.currentToken().Position.Start

	// Check for not operator (-)
	if p.currentTokenIs("-") {
		return p.parseNotToken()
	}

	// Check for field prefix
	field, hasField, isComplete, fieldPos := p.parseFieldPrefix()

	// If we have a field prefix but it's incomplete, return a token with empty search term
	if hasField && !isComplete {
		pos := Position{Start: startPos, End: fieldPos.End}
		return makeSearchNode("exact", "", field, false, pos)
	}

	// Parse an unmodified token
	node := p.parseUnmodifiedToken()
	if node == nil {
		return nil
	}

	if hasField {
		node.Field = field
		// Update the position to include the field prefix
		node.Position.Start = startPos
	}

	return node
}

// parseAndExpr parses a sequence of tokens (AND expression)
func (p *Parser) parseAndExpr() *Node {
	startPos := p.currentToken().Position.Start
	var children []*Node
	var lastTokenEnd int

	for !p.currentTokenIs(TokenEOF) && !p.currentTokenIs("|") {
		// Skip whitespace
		p.skipWhitespace()

		// If we've reached the end of the input or a pipe, break
		if p.currentTokenIs(TokenEOF) || p.currentTokenIs("|") {
			break
		}

		// Parse a token
		token := p.parseToken()
		if token == nil {
			continue
		}

		// Add the token to the children
		children = append(children, token)

		// Update the last token end position
		lastTokenEnd = token.Position.End

		// Require whitespace between tokens (except before EOF or pipe)
		if !p.currentTokenIs(TokenEOF) && !p.currentTokenIs("|") && !p.currentTokenIs(TokenWhitespace) {
			// No whitespace between tokens - this is an error
			// For now, we'll just stop parsing this AND expression
			break
		}
	}

	// If there are no children, return an empty AND node
	if len(children) == 0 {
		endPos := p.currentToken().Position.Start
		return &Node{
			Type:     NodeTypeAnd,
			Position: Position{Start: startPos, End: endPos},
			Children: make([]Node, 0),
		}
	}

	// If there's only one child, return it directly
	if len(children) == 1 {
		return children[0]
	}

	// Create an AND node with the children
	// Use the last token's end position as the end position for the AND node

	// Convert []*Node to []Node for the Children field
	nodeChildren := make([]Node, len(children))
	for i, child := range children {
		nodeChildren[i] = *child
	}

	return &Node{
		Type:     NodeTypeAnd,
		Position: Position{Start: startPos, End: lastTokenEnd},
		Children: nodeChildren,
	}
}

// parseOrExpr parses an OR expression (and_expr { "|" and_expr })
func (p *Parser) parseOrExpr() *Node {
	startPos := p.currentToken().Position.Start
	var children []*Node

	// Handle the case where the expression starts with a pipe
	if p.currentTokenIs("|") {
		// Skip the "|" character
		p.nextToken()

		// Skip any whitespace after the "|"
		p.skipWhitespace()

		// Add an empty AND node before the pipe
		children = append(children, &Node{
			Type:     NodeTypeAnd,
			Position: Position{Start: startPos, End: startPos},
			Children: []Node{},
		})

		// Parse the expression after the pipe
		andNode := p.parseAndExpr()
		children = append(children, andNode)

		// Continue parsing if there are more pipes
		for p.currentTokenIs("|") {
			// Skip the "|" character
			p.nextToken()

			// Skip any whitespace after the "|"
			p.skipWhitespace()

			// Parse the next AND expression
			andNode = p.parseAndExpr()
			children = append(children, andNode)
		}

		// Convert []*Node to []Node for the Children field
		nodeChildren := make([]Node, len(children))
		for i, child := range children {
			nodeChildren[i] = *child
		}

		// Create an OR node with the children
		endPos := p.currentToken().Position.Start
		return &Node{
			Type:     NodeTypeOr,
			Position: Position{Start: startPos, End: endPos},
			Children: nodeChildren,
		}
	}

	// Parse the first AND expression
	andNode := p.parseAndExpr()
	children = append(children, andNode)

	// If there are no OR operators, return the AND node directly
	if !p.currentTokenIs("|") {
		return andNode
	}

	// Parse additional AND expressions separated by "|"
	for p.currentTokenIs("|") {
		// Skip the "|" character
		p.nextToken()

		// Skip any whitespace after the "|"
		p.skipWhitespace()

		// Parse the next AND expression
		andNode = p.parseAndExpr()
		children = append(children, andNode)
	}

	// Convert []*Node to []Node for the Children field
	nodeChildren := make([]Node, len(children))
	for i, child := range children {
		nodeChildren[i] = *child
	}

	// Create an OR node with the children
	endPos := p.currentToken().Position.Start
	return &Node{
		Type:     NodeTypeOr,
		Position: Position{Start: startPos, End: endPos},
		Children: nodeChildren,
	}
}

// ParseAST parses the input string into an AST
func (p *Parser) ParseAST() *Node {
	// Special case for a single pipe character
	if len(p.tokens) == 2 && p.tokens[0].Type == "|" && p.tokens[1].Type == TokenEOF {
		return makeSearchNode(NodeTypeOr, "|", "", false, Position{Start: 0, End: 1})
	}

	// Parse the OR expression
	return p.parseOrExpr()
}

// ParseSearch parses a search string into an AST
func ParseSearch(searchString string) *Node {
	searchString = strings.TrimSpace(searchString)
	if searchString == "" {
		return &Node{
			Type:     NodeTypeAnd,
			Position: Position{Start: 0, End: 0},
			Children: []Node{},
		}
	}
	parser := NewParser(searchString)
	return parser.ParseAST()
}
