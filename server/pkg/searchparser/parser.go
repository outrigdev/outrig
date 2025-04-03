// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Search Parser Grammar (EBNF):

// search           = WS? or_expr WS? EOF ;
// or_expr          = and_expr { WS? "|" WS? and_expr } ;
// and_expr         = token { WS token } ;
// token            = not_token | field_token ;
// not_token        = "-" field_token ;
// field_token      = [ field_prefix ] unmodified_token ;
// field_prefix     = "$" WORD ":" ;
// unmodified_token = fuzzy_token | regexp_token | tag_token | simple_token ;
// fuzzy_token      = "~" simple_token ;
// regexp_token     = REGEXP | CREGEXP;
// tag_token        = "#" WORD [ "/" ] ;
// simple_token     = DQUOTE | SQUOTE | WORD
//
// Notes:
// - Empty control tokens (like "~", "$", ":", "-" or "#") followed by whitespace are errors
// - Single quoted tokens are treated as case-sensitive (exactcase)
// - Fuzzy tokens with single quotes (~'...') are treated as case-sensitive fuzzy search (fzfcase)
// - Tag tokens with trailing slash (#foo/) require exact matches
// - Special case: #marked or #m uses the marked searcher to find marked lines
// - Not token (-) negates the search result of the token that follows it
// - A literal "-" at the start of a token must be quoted: "-hello" searches for "-hello" literally

package searchparser

import (
	"fmt"
)

// --- Node Types & Constants ---

const (
	NodeTypeSearch = "search"
	NodeTypeAnd    = "and"
	NodeTypeOr     = "or"
	NodeTypeError  = "error"
)

const (
	SearchTypeExact      = "exact"
	SearchTypeExactCase  = "exactcase"
	SearchTypeRegexp     = "regexp"
	SearchTypeRegexpCase = "regexpcase"
	SearchTypeFzf        = "fzf"
	SearchTypeFzfCase    = "fzfcase"
	SearchTypeNot        = "not"
	SearchTypeTag        = "tag"
	SearchTypeUserQuery  = "userquery"
)

// --- AST Node Definition ---

type Position struct {
	Start int // Start position in the input string
	End   int // End position in the input string
}

type Node struct {
	Type         string   // NodeTypeAnd, NodeTypeOr, NodeTypeSearch, NodeTypeError
	Position     Position // Position in the source text
	Children     []*Node  // For composite nodes (AND/OR)
	SearchType   string   // e.g., "exact", "regexp", "fzf", etc. (only for search nodes)
	SearchTerm   string   // The actual search text (only for search nodes)
	Field        string   // Optional field specifier (only for search nodes)
	IsNot        bool     // Set to true if preceded by '-' (for not tokens)
	ErrorMessage string   // For error nodes, a simple error message
}

// --- Parser Definition ---

type Parser struct {
	tokens   []Token // from the tokenizer
	position int
	input    string // original input (for error reporting)
}

// NewParser creates a parser and tokenizes the input.
func NewParser(input string) *Parser {
	tokenizer := NewTokenizer(input)
	tokens := tokenizer.GetAllTokens()
	return &Parser{
		tokens:   tokens,
		position: 0,
		input:    input,
	}
}

// --- Helper Functions ---

func (p *Parser) isCurrentADelimiter() bool {
	switch p.current().Type {
	case TokenWhitespace, TokenPipe, TokenLParen, TokenRParen, TokenEOF:
		return true
	default:
		return false
	}
}

func (p *Parser) current() Token {
	if p.position < len(p.tokens) {
		return p.tokens[p.position]
	}
	// Return an EOF token if out of tokens.
	return Token{Type: TokenEOF, Value: "", Position: Position{Start: len(p.input), End: len(p.input)}}
}

func (p *Parser) getCurrentStartPos() int {
	return p.current().Position.Start
}

func (p *Parser) atEOF() bool {
	return p.current().Type == TokenEOF
}

func (p *Parser) advance() {
	if !p.atEOF() {
		p.position++
	}
}

func (p *Parser) save() int {
	return p.position
}

func (p *Parser) restore(pos int) {
	p.position = pos
}

// skipOptionalWhitespace advances over WS tokens.
func (p *Parser) skipOptionalWhitespace() {
	for !p.atEOF() && p.current().Type == TokenWhitespace {
		p.advance()
	}
}

// matchToken returns true if the current token’s value matches.
func (p *Parser) matchToken(val string) bool {
	if p.atEOF() {
		return false
	}
	return p.current().Value == val
}

func (p *Parser) consumeToken(tokenType TokenType) (*Token, bool) {
	if p.atEOF() {
		return nil, false
	}
	cur := p.current()
	if cur.Type == tokenType {
		p.advance()
		return &cur, true
	}
	return nil, false
}

func makeErrorNode(pos Position, msg string) *Node {
	return &Node{
		Type:         NodeTypeError,
		Position:     pos,
		ErrorMessage: msg,
	}
}

func (n *Node) addTokenToErrorNode(token Token) {
	if n.Type != NodeTypeError {
		return
	}
	if n.Position.Start == 0 && n.Position.End == 0 {
		n.Position = token.Position
	} else {
		n.Position.End = token.Position.End
	}
}

func (p *Parser) addToErrorUntilDelimiter(errNode *Node) {
	for !p.atEOF() && !p.isCurrentADelimiter() {
		token := p.current()
		errNode.addTokenToErrorNode(token)
		p.advance()
	}
}

func (p *Parser) createErrorToDelimiter(msg string, pos Position) *Node {
	errNode := makeErrorNode(pos, msg)
	p.addToErrorUntilDelimiter(errNode)
	return errNode
}

func (p *Parser) createErrorToEOF(msg string) *Node {
	cur := p.current()
	if cur.Type == TokenEOF {
		return nil
	}
	errNode := makeErrorNode(cur.Position, msg)
	for !p.atEOF() {
		token := p.current()
		errNode.addTokenToErrorNode(token)
		p.advance()
	}
	return errNode
}

func removeNilNodes(nodes []*Node) []*Node {
	var result []*Node
	for _, node := range nodes {
		if node == nil {
			continue
		}
		result = append(result, node)
	}
	return result
}

func makeAndNode(children ...*Node) *Node {
	children = removeNilNodes(children)
	if len(children) == 0 {
		return nil
	}
	if len(children) == 1 {
		return children[0]
	}
	return &Node{
		Type:     NodeTypeAnd,
		Children: children,
		Position: Position{Start: children[0].Position.Start, End: children[len(children)-1].Position.End},
	}
}

func makeOrNode(children ...*Node) *Node {
	children = removeNilNodes(children)
	if len(children) == 0 {
		return nil
	}
	if len(children) == 1 {
		return children[0]
	}
	return &Node{
		Type:     NodeTypeOr,
		Children: children,
		Position: Position{Start: children[0].Position.Start, End: children[len(children)-1].Position.End},
	}
}

// --- Top-Level Parse Function ---

// Parse builds the AST for the entire search expression.
// search           = WS? or_expr WS? EOF ;
func (p *Parser) Parse() *Node {
	p.skipOptionalWhitespace()
	node := p.parseOrExpr()
	if !p.atEOF() {
		errNode := p.createErrorToEOF("Unparsed input remaining")
		return makeAndNode(node, errNode)
	}
	return node
}

// --- Parsing Functions Corresponding to the EBNF ---

// parseOrExpr parses an OR expression.
// or_expr = and_expr { WS? "|" WS? and_expr } ;
func (p *Parser) parseOrExpr() *Node {
	var nodes []*Node
	for !p.atEOF() {
		if len(nodes) > 0 {
			p.consumeToken(TokenWhitespace)
			if p.atEOF() {
				break
			}
			_, ok := p.consumeToken(TokenPipe)
			if !ok {
				errToken := p.createErrorToDelimiter("Expected '|'", p.current().Position)
				nodes = append(nodes, errToken)
				continue
			}
			p.consumeToken(TokenWhitespace)
		}
		node := p.parseAndExpr()
		if node == nil {
			break
		}
		nodes = append(nodes, node)
	}
	return makeOrNode(nodes...)
}

// token { WS token }
func (p *Parser) parseAndExpr() *Node {
	var nodes []*Node
	for !p.atEOF() {
		if len(nodes) > 0 {
			p.consumeToken(TokenWhitespace)
		}
		node := p.parseTokenWithErrorSync()
		if node == nil {
			break
		}
		nodes = append(nodes, node)
	}
	return makeAndNode(nodes...)
}

// token is where we're going to implement error sync points
// token            = not_token | field_token ;
func (p *Parser) parseTokenWithErrorSync() *Node {
	if p.atEOF() || p.isCurrentADelimiter() {
		return nil
	}
	savePoint := p.save()
	startPos := p.getCurrentStartPos()
	node, err := p.parseToken()
	if err != nil {
		// We have a specific error from the parser
		p.restore(savePoint)
		errNode := makeErrorNode(Position{Start: startPos, End: startPos}, err.Error())
		p.addToErrorUntilDelimiter(errNode)
		return errNode
	}
	if node != nil {
		// Check if the next token is a valid delimiter
		if !p.isCurrentADelimiter() {
			// Not a valid delimiter, convert to error node
			currentToken := p.current()
			errMsg := fmt.Sprintf("invalid token sequence: unexpected '%s' after search term", currentToken.Type)
			errNode := makeErrorNode(Position{Start: startPos, End: node.Position.End}, errMsg)
			// Add everything up to the next delimiter
			p.addToErrorUntilDelimiter(errNode)
			return errNode
		}
		return node
	}
	// so we didn't get a valid token, that means there was an error, restore and then produce an error node
	p.restore(savePoint)
	errMsg := fmt.Sprintf("unexpected token %s", p.current().Type)
	errNode := p.createErrorToDelimiter(errMsg, Position{Start: startPos, End: startPos})
	return errNode
}

// parseToken parses a token according to the grammar:
// token = not_token | field_token
func (p *Parser) parseToken() (*Node, error) {
	// Try to parse a not token first
	node, err := p.parseNotToken()
	if err != nil {
		return nil, err
	}
	if node != nil {
		return node, nil
	}

	// If that fails, try to parse a field token
	return p.parseFieldToken()
}

// parseNotToken parses a not token according to the grammar:
// not_token = "-" field_token
func (p *Parser) parseNotToken() (*Node, error) {
	startPos := p.getCurrentStartPos()

	// Check for "-" token
	_, ok := p.consumeToken(TokenMinus)
	if !ok {
		return nil, nil // Not a not_token, no error
	}

	// Parse a field token
	node, err := p.parseFieldToken()
	if err != nil {
		return nil, fmt.Errorf("after '-': %w", err)
	}
	if node == nil {
		return nil, fmt.Errorf("'-' must be followed by a search term")
	}

	// Set the IsNot flag and update position
	node.IsNot = true
	node.Position.Start = startPos

	return node, nil
}

// parseFieldToken parses a field token according to the grammar:
// field_token = [ field_prefix ] unmodified_token
func (p *Parser) parseFieldToken() (*Node, error) {
	startPos := p.getCurrentStartPos()

	// Try to parse a field prefix
	var field string
	var hadPrefix bool
	var prefixErr error

	field, prefixErr = p.parseFieldPrefix()
	if prefixErr != nil {
		return nil, prefixErr // Return the field prefix error
	}
	if field != "" {
		hadPrefix = true
	}

	// Parse an unmodified token
	node, err := p.parseUnmodifiedToken()
	if err != nil {
		return nil, err
	}
	if node == nil {
		if hadPrefix {
			return nil, fmt.Errorf("field prefix must be followed by a search term")
		}
		return nil, nil // Not a field_token, no error
	}

	// Set the field if we have one
	if hadPrefix {
		node.Field = field
		// Update position to include the field prefix
		node.Position.Start = startPos
	}

	return node, nil
}

// parseFieldPrefix parses a field prefix according to the grammar:
// field_prefix = "$" WORD ":"
func (p *Parser) parseFieldPrefix() (string, error) {
	// Check for "$" token
	_, ok := p.consumeToken(TokenDollar)
	if !ok {
		return "", nil // Not a field prefix, no error
	}

	// Check for WORD token
	wordToken, ok := p.consumeToken(TokenWord)
	if !ok {
		return "", fmt.Errorf("'$' must be followed by a field name")
	}

	// Check for ":" token
	_, ok = p.consumeToken(TokenColon)
	if !ok {
		return "", fmt.Errorf("field name must be followed by ':'")
	}

	// Return field name
	return wordToken.Value, nil
}

// parseUnmodifiedToken parses an unmodified token according to the grammar:
// unmodified_token = fuzzy_token | regexp_token | tag_token | simple_token
func (p *Parser) parseUnmodifiedToken() (*Node, error) {
	// Try each type of unmodified token in order
	node, err := p.parseFuzzyToken()
	if err != nil {
		return nil, err
	}
	if node != nil {
		return node, nil
	}

	node, err = p.parseRegexpToken()
	if err != nil {
		return nil, err
	}
	if node != nil {
		return node, nil
	}

	node, err = p.parseTagToken()
	if err != nil {
		return nil, err
	}
	if node != nil {
		return node, nil
	}

	return p.parseSimpleToken()
}

// parseFuzzyToken parses a fuzzy token according to the grammar:
// fuzzy_token = "~" simple_token
func (p *Parser) parseFuzzyToken() (*Node, error) {
	startPos := p.getCurrentStartPos()

	// Check for "~" token
	_, ok := p.consumeToken(TokenTilde)
	if !ok {
		return nil, nil // Not a fuzzy token, no error
	}

	// Parse a simple token
	node, err := p.parseSimpleToken()
	if err != nil {
		return nil, fmt.Errorf("after '~': %w", err)
	}
	if node == nil {
		return nil, fmt.Errorf("'~' must be followed by a search term")
	}

	// Update the search type and position
	if node.SearchType == SearchTypeExactCase {
		node.SearchType = SearchTypeFzfCase
	} else {
		node.SearchType = SearchTypeFzf
	}

	node.Position.Start = startPos

	return node, nil
}

// parseRegexpToken parses a regexp token according to the grammar:
// regexp_token = REGEXP | CREGEXP
func (p *Parser) parseRegexpToken() (*Node, error) {
	cur := p.current()

	if cur.Type == TokenRegexp {
		p.advance()
		return &Node{
			Type:       NodeTypeSearch,
			Position:   cur.Position,
			SearchType: SearchTypeRegexp,
			SearchTerm: cur.Value,
		}, nil
	}

	if cur.Type == TokenCRegexp {
		p.advance()
		return &Node{
			Type:       NodeTypeSearch,
			Position:   cur.Position,
			SearchType: SearchTypeRegexpCase,
			SearchTerm: cur.Value,
		}, nil
	}

	return nil, nil // Not a regexp token, no error
}

// parseTagToken parses a tag token according to the grammar:
// tag_token = "#" WORD [ "/" ]
func (p *Parser) parseTagToken() (*Node, error) {
	startPos := p.getCurrentStartPos()

	// Check for "#" token
	_, ok := p.consumeToken(TokenHash)
	if !ok {
		return nil, nil // Not a tag token, no error
	}

	// Check for WORD token
	wordToken, ok := p.consumeToken(TokenWord)
	if !ok {
		return nil, fmt.Errorf("'#' must be followed by a tag name")
	}

	// Create the tag node
	node := &Node{
		Type:       NodeTypeSearch,
		Position:   Position{Start: startPos, End: wordToken.Position.End},
		SearchType: SearchTypeTag,
		SearchTerm: wordToken.Value,
	}

	// Check for optional trailing slash
	if p.matchToken("/") {
		slashToken, _ := p.consumeToken(TokenWord) // "/" is tokenized as a word
		node.Position.End = slashToken.Position.End
		// Exact tag match required when slash is present
	}

	return node, nil
}

// parseSimpleToken parses a simple token according to the grammar:
// simple_token = DQUOTE | SQUOTE | WORD
func (p *Parser) parseSimpleToken() (*Node, error) {
	cur := p.current()

	switch cur.Type {
	case TokenDQuote:
		p.advance()
		return &Node{
			Type:       NodeTypeSearch,
			Position:   cur.Position,
			SearchType: SearchTypeExact,
			SearchTerm: cur.Value,
		}, nil
	case TokenSQuote:
		p.advance()
		return &Node{
			Type:       NodeTypeSearch,
			Position:   cur.Position,
			SearchType: SearchTypeExactCase,
			SearchTerm: cur.Value,
		}, nil
	case TokenWord:
		p.advance()
		return &Node{
			Type:       NodeTypeSearch,
			Position:   cur.Position,
			SearchType: SearchTypeExact,
			SearchTerm: cur.Value,
		}, nil
	default:
		return nil, nil // Not a simple token, no error
	}
}
