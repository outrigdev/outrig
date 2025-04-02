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
// regexp_token     = REGEXP | CASEREGEXP;
// tag_token        = "#" WORD [ "/" ] ;
// simple_token     = DOUBLEQUOTED | SINGLEQUOTED | WORD
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
	Children     []Node   // For composite nodes (AND/OR)
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

func (p *Parser) current() Token {
	if p.position < len(p.tokens) {
		return p.tokens[p.position]
	}
	// Return an EOF token if out of tokens.
	return Token{Type: TokenEOF, Value: "", Position: Position{Start: len(p.input), End: len(p.input)}}
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

func combinePositions(a, b Position) Position {
	return Position{Start: a.Start, End: b.End}
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

// --- Top-Level Parse Function ---

// Parse builds the AST for the entire search expression.
func (p *Parser) Parse() Node {
	node := p.parseOrExpr()
	// If there are leftover tokens (ignoring whitespace), produce an error node.
	p.skipOptionalWhitespace()
	if !p.atEOF() {
		extra := p.current()
		errNode := Node{
			Type:         NodeTypeError,
			Position:     extra.Position,
			ErrorMessage: "Unexpected extra tokens",
		}
		// Combine valid AST and error into an AND node.
		node = Node{
			Type:     NodeTypeAnd,
			Children: []Node{node, errNode},
			Position: combinePositions(node.Position, errNode.Position),
		}
	}
	return node
}

// --- Parsing Functions Corresponding to the EBNF ---

// or_expr = and_expr { WS? "|" WS? and_expr }
func (p *Parser) parseOrExpr() Node {
	left := p.parseAndExpr()
	for {
		p.skipOptionalWhitespace()
		if p.matchToken("|") {
			// Consume the '|' token.
			opToken := p.current()
			p.advance()
			p.skipOptionalWhitespace()
			right := p.parseAndExpr()
			left = Node{
				Type:     NodeTypeOr,
				Children: []Node{left, right},
				Position: combinePositions(left.Position, right.Position),
			}
			_ = opToken // (could be used for more detailed reporting)
		} else {
			break
		}
	}
	return left
}

// and_expr = token { WS token }
// When two tokens are contiguous (i.e. no WS between), report a single error.
func (p *Parser) parseAndExpr() Node {
	var nodes []Node

	// Parse the first token.
	tokenNode := p.parseToken()
	nodes = append(nodes, tokenNode)
	lastPos := tokenNode.Position

	for {
		// If next token is WS or a delimiter, handle accordingly.
		if p.atEOF() || p.matchToken("|") {
			break
		}

		// Check if there is whitespace between the last token and the current token.
		cur := p.current()
		if lastPos.End == cur.Position.Start {
			// No whitespace: collect all contiguous tokens.
			errStart := lastPos.End
			errEnd := cur.Position.End
			for !p.atEOF() && p.current().Position.Start == errEnd && !p.matchToken("|") {
				errEnd = p.current().Position.End
				p.advance()
			}
			errNode := Node{
				Type:         NodeTypeError,
				Position:     Position{Start: errStart, End: errEnd},
				ErrorMessage: "Search tokens require whitespace to separate them",
			}
			nodes = append(nodes, errNode)
		} else {
			// There is whitespace: skip it and parse the next token.
			p.skipOptionalWhitespace()
			if p.atEOF() || p.matchToken("|") {
				break
			}
			nextToken := p.parseToken()
			nodes = append(nodes, nextToken)
			lastPos = nextToken.Position
		}
	}

	// If there's only one node, return it directly.
	if len(nodes) == 1 {
		return nodes[0]
	}
	// Otherwise, combine them into an AND node.
	return Node{
		Type:     NodeTypeAnd,
		Children: nodes,
		Position: Position{Start: nodes[0].Position.Start, End: nodes[len(nodes)-1].Position.End},
	}
}

// token = not_token | field_token
func (p *Parser) parseToken() Node {
	// Check for a NOT token first.
	if p.current().Type == TokenMinus {
		return p.parseNotToken()
	}
	// Otherwise, parse as a field_token.
	return p.parseFieldToken()
}

// not_token = "-" field_token
func (p *Parser) parseNotToken() Node {
	minusToken := p.current()
	p.advance() // consume '-'
	// Try to parse a field_token following the '-'
	tokenNode := p.parseFieldToken()
	// If the field token was an error, wrap the '-' itself as an error.
	if tokenNode.Type == NodeTypeError {
		return Node{
			Type:         NodeTypeError,
			Position:     minusToken.Position,
			ErrorMessage: "Invalid token following '-'",
		}
	}
	// Mark the resulting node as a NOT token.
	tokenNode.IsNot = true
	tokenNode.SearchType = SearchTypeNot
	tokenNode.Position = Position{Start: minusToken.Position.Start, End: tokenNode.Position.End}
	return tokenNode
}

// field_token = [ field_prefix ] unmodified_token
// field_prefix = "$" WORD ":"
func (p *Parser) parseFieldToken() Node {
	startPos := p.save()
	var field string

	// Attempt to parse an optional field prefix.
	if p.current().Type == TokenDollar {
		dollarToken := p.current()
		p.advance() // consume '$'
		if p.current().Type == TokenWord {
			fieldToken := p.current()
			p.advance() // consume the field name
			if p.current().Type == TokenColon {
				colonToken := p.current()
				p.advance() // consume ':'
				field = fieldToken.Value
				_ = colonToken // (could be used in error reporting)
			} else {
				// Missing colon after field name.
				return Node{
					Type:         NodeTypeError,
					Position:     Position{Start: dollarToken.Position.Start, End: p.current().Position.Start},
					ErrorMessage: "Expected ':' after field name",
				}
			}
		} else {
			// Expected a field name (WORD) after '$'
			return Node{
				Type:         NodeTypeError,
				Position:     dollarToken.Position,
				ErrorMessage: "Expected field name after '$'",
			}
		}
	}

	// Now try to parse an unmodified token.
	node, ok := p.tryParseUnmodifiedToken()
	if !ok {
		// If we had a field prefix but no valid search term, produce an error node.
		return Node{
			Type:         NodeTypeError,
			Position:     p.tokens[startPos].Position,
			ErrorMessage: "Invalid field token: missing search term after field prefix",
		}
	}
	// Attach the field if one was found.
	node.Field = field
	return node
}

// unmodified_token = fuzzy_token | regexp_token | tag_token | simple_token
func (p *Parser) tryParseUnmodifiedToken() (Node, bool) {
	pos := p.save()
	if node, ok := p.tryParseFuzzyToken(); ok {
		return node, true
	}
	p.restore(pos)
	if node, ok := p.tryParseRegexpToken(); ok {
		return node, true
	}
	p.restore(pos)
	if node, ok := p.tryParseTagToken(); ok {
		return node, true
	}
	p.restore(pos)
	if node, ok := p.tryParseSimpleToken(); ok {
		return node, true
	}
	p.restore(pos)
	// If nothing matches, return an error node.
	return Node{
		Type:         NodeTypeError,
		Position:     p.current().Position,
		ErrorMessage: "Expected search term",
	}, false
}

// fuzzy_token = "~" simple_token
func (p *Parser) tryParseFuzzyToken() (Node, bool) {
	if p.current().Type == TokenTilde {
		tildeToken := p.current()
		p.advance() // consume '~'
		if node, ok := p.tryParseSimpleToken(); ok {
			// Choose search type based on the simple token’s case.
			if node.SearchType == SearchTypeExactCase {
				node.SearchType = SearchTypeFzfCase
			} else {
				node.SearchType = SearchTypeFzf
			}
			node.Position = Position{Start: tildeToken.Position.Start, End: node.Position.End}
			return node, true
		}
		// If no simple token follows, return an error node.
		errorNode := Node{
			Type:         NodeTypeError,
			Position:     tildeToken.Position,
			ErrorMessage: "Expected search term after '~'",
		}
		return errorNode, true
	}
	return Node{}, false
}

// regexp_token = REGEXP | CASEREGEXP
func (p *Parser) tryParseRegexpToken() (Node, bool) {
	if p.current().Type == TokenRegexp || p.current().Type == TokenCaseRegexp {
		token := p.current()
		p.advance()
		searchType := SearchTypeRegexp
		if token.Type == TokenCaseRegexp {
			searchType = SearchTypeRegexpCase
		}
		return Node{
			Type:       NodeTypeSearch,
			SearchType: searchType,
			SearchTerm: token.Value,
			Position:   token.Position,
		}, true
	}
	return Node{}, false
}

// tag_token = "#" WORD [ "/" ]
func (p *Parser) tryParseTagToken() (Node, bool) {
	if p.current().Type == TokenHash {
		hashToken := p.current()
		p.advance() // consume '#'
		if p.current().Type == TokenWord {
			wordToken := p.current()
			p.advance() // consume the tag name
			// Optionally, check for a trailing "/" with no intervening whitespace.
			if !p.atEOF() && p.current().Value == "/" {
				p.advance() // consume the '/'
			}
			return Node{
				Type:       NodeTypeSearch,
				SearchType: SearchTypeTag,
				SearchTerm: wordToken.Value,
				Position:   combinePositions(hashToken.Position, wordToken.Position),
			}, true
		}
		errorNode := Node{
			Type:         NodeTypeError,
			Position:     hashToken.Position,
			ErrorMessage: "Expected tag name after '#'",
		}
		return errorNode, true
	}
	return Node{}, false
}

// simple_token = DOUBLEQUOTED | SINGLEQUOTED | WORD
func (p *Parser) tryParseSimpleToken() (Node, bool) {
	current := p.current()
	if current.Type == TokenWord || current.Type == TokenDoubleQuoted || current.Type == TokenSingleQuoted {
		p.advance()
		searchType := SearchTypeExact
		if current.Type == TokenSingleQuoted {
			searchType = SearchTypeExactCase
		}
		return Node{
			Type:       NodeTypeSearch,
			SearchType: searchType,
			SearchTerm: current.Value,
			Position:   current.Position,
		}, true
	}
	return Node{}, false
}

// --- End of Parser Implementation ---

// For debugging: a simple print of the AST.
func (n Node) String() string {
	switch n.Type {
	case NodeTypeSearch:
		return fmt.Sprintf("Search(%s:%s)", n.SearchType, n.SearchTerm)
	case NodeTypeError:
		return fmt.Sprintf("Error(%s)", n.ErrorMessage)
	default:
		s := fmt.Sprintf("%s[", n.Type)
		for i, child := range n.Children {
			if i > 0 {
				s += ", "
			}
			s += child.String()
		}
		s += "]"
		return s
	}
}
