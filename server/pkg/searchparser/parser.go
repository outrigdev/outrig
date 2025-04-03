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

func (p *Parser) tryParse(fn func() *Node) (*Node, bool) {
	pos := p.save()
	node := fn()
	if node != nil {
		return node, true
	}
	p.restore(pos)
	return nil, false
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
	node := p.parseToken()
	if node != nil {
		if node.Type == NodeTypeError {
			// if we have an error node, we will consume to the next delimiter
			p.addToErrorUntilDelimiter(node)
		}
		return node
	}
	// so we didn't get a valid token, that means there was an error, restore and then produce an error node
	p.restore(savePoint)
	errMsg := fmt.Sprintf("unexpected token %s", p.current().Type)
	errNode := p.createErrorToDelimiter(errMsg, Position{Start: startPos, End: startPos})
	return errNode
}

func (p *Parser) parseToken() *Node {
	cur := p.current()
	if cur.Type == TokenWord {
		p.advance()
		return &Node{
			Type:       NodeTypeSearch,
			Position:   cur.Position,
			SearchType: SearchTypeExact,
			SearchTerm: cur.Value,
		}
	}
	return nil
}
