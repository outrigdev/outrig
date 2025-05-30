// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Search Parser Grammar (EBNF):

// search           = WS? or_expr WS? EOF ;
// or_expr          = and_expr { WS? "|" WS? and_expr } ;
// and_expr         = group { WS group } ;
// group            = "(" WS? or_expr WS? ")" | token
// token            = not_token | field_token | unmodified_token ;
// not_token        = "-" field_token | "-" unmodified_token ;
// field_token      = "$" WORD | "$" WORD unmodified_token ;
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
// - Numeric field search supports operators: >, <, >=, <= (e.g., $goid:>500, $goid:<=200)
// Once parsing a WORD the only characters that break a WORD are whitespace, "|", "(", ")", "\"", "'", and EOF

package searchparser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/outrigdev/outrig/pkg/utilfn"
)

// numericSearchRegex matches numeric comparison operators (>, <, >=, <=) followed by digits
var numericSearchRegex = regexp.MustCompile(`^([><]=?)(\d+)$`)

// numericOperatorRegex matches just the numeric comparison operators (>, <, >=, <=)
var numericOperatorRegex = regexp.MustCompile(`^([><]=?)(.*)$`)

// TagRegexp is the regular expression pattern for valid tag names
// Uses SimpleTagRegexStr from utilfn/util.go for consistency
var TagRegexp = regexp.MustCompile(`^` + utilfn.SimpleTagRegexStr + `$`)

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
	SearchTypeMarked     = "marked"
	SearchTypeNumeric    = "numeric"
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
	Op           string   // Optional operator for numeric searches (>, <, >=, <=)
	IsNot        bool     // Set to true if preceded by '-' (for not tokens)
	ErrorMessage string   // For error nodes, a simple error message
}

// PrettyPrint formats a Node structure in a concise way
func (n *Node) PrettyPrint(indent string, originalQuery string) string {
	if n == nil {
		return indent + "nil"
	}

	var sb strings.Builder

	// Format node type and position consistently with token format
	sb.WriteString(fmt.Sprintf("%s%-8s [%2d:%2d]", indent, n.Type, n.Position.Start, n.Position.End))

	// Add node-specific attributes on the same line when possible
	if n.Type == NodeTypeSearch {
		sb.WriteString(fmt.Sprintf(" %s %q", n.SearchType, n.SearchTerm))
		if n.Field != "" {
			sb.WriteString(fmt.Sprintf(" field:%q", n.Field))
		}
		if n.Op != "" {
			sb.WriteString(fmt.Sprintf(" op:%q", n.Op))
		}
		if n.IsNot {
			sb.WriteString(" not:true")
		}
	} else if n.Type == NodeTypeError {
		sb.WriteString(fmt.Sprintf(" %q", n.ErrorMessage))
	}

	// Add substring visualization at the end
	substring := utilfn.SafeSubstring(originalQuery, n.Position.Start, n.Position.End)
	sb.WriteString(fmt.Sprintf(" | [%s]", substring))

	sb.WriteString("\n")

	// Handle children with increased indentation
	if len(n.Children) > 0 {
		for _, child := range n.Children {
			sb.WriteString(child.PrettyPrint(indent+"  ", originalQuery))
		}
	}

	return sb.String()
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

			// Check for right parenthesis - it will be handled by parseGroup
			if p.current().Type == TokenRParen {
				break
			}

			// If we find a right parenthesis instead of a pipe, that's not an error
			if p.current().Type == TokenRParen {
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

// and_expr = group { WS group } ;
func (p *Parser) parseAndExpr() *Node {
	var nodes []*Node
	for !p.atEOF() {
		// Stop if we encounter a right parenthesis - it will be handled by parseGroup
		if p.current().Type == TokenRParen {
			break
		}

		if len(nodes) > 0 {
			p.consumeToken(TokenWhitespace)
			// Check again after consuming whitespace
			if p.current().Type == TokenRParen {
				break
			}
		}

		node := p.parseGroup()
		if node == nil {
			break
		}
		nodes = append(nodes, node)
	}
	return makeAndNode(nodes...)
}

// group = "(" WS? or_expr WS? ")" | token
func (p *Parser) parseGroup() *Node {
	if p.atEOF() {
		return nil
	}

	// Check if this is a parenthesized expression
	if p.current().Type == TokenLParen {
		startPos := p.getCurrentStartPos()
		p.advance() // Consume the left parenthesis

		// Skip optional whitespace
		p.skipOptionalWhitespace()

		// Parse the or_expr inside the parentheses
		node := p.parseOrExpr()

		// Skip optional whitespace
		p.skipOptionalWhitespace()

		// Expect a right parenthesis
		if p.current().Type != TokenRParen {
			// If we're at EOF, treat it as if the parenthesis was closed (for typeahead search)
			if p.atEOF() {
				// Just return the node without an error, as if the parenthesis was closed
				if node != nil {
					// Update the position to include the entire expression
					node.Position.Start = startPos
					node.Position.End = p.current().Position.Start
				}
				return node
			}

			// Not at EOF, so this is an error - missing closing parenthesis
			currentPos := p.current().Position.Start
			errNode := makeErrorNode(Position{Start: currentPos, End: currentPos}, "Expected closing parenthesis ')'")

			// Create an AND node with the correct position
			result := makeAndNode(node, errNode)

			// Ensure the AND node has the correct position
			if result != nil && result.Type == NodeTypeAnd {
				result.Position = Position{Start: startPos, End: currentPos}
			}

			return result
		}

		// Update the position to include the closing parenthesis
		if node != nil {
			node.Position.Start = startPos
			node.Position.End = p.current().Position.End
		}

		p.advance() // Consume the right parenthesis
		return node
	}

	// If not a parenthesized expression, parse a token
	return p.parseTokenWithErrorSync()
}

// token is where we're going to implement error sync points
// token            = not_token | field_token | unmodified_token ;
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
// token = not_token | field_token | unmodified_token
func (p *Parser) parseToken() (*Node, error) {
	// Check for "-" to parse a not token
	if p.current().Type == TokenMinus {
		return p.parseNotToken()
	}

	// Check for "$" to parse a field token
	if p.current().Type == TokenDollar {
		return p.parseFieldToken()
	}

	// Otherwise, parse an unmodified token directly
	return p.parseUnmodifiedToken()
}

// parseNotToken parses a not token according to the grammar:
// not_token = "-" field_token | "-" unmodified_token
func (p *Parser) parseNotToken() (*Node, error) {
	startPos := p.getCurrentStartPos()

	// Consume the "-" token (we already checked it exists in parseToken)
	_, hasMinus := p.consumeToken(TokenMinus)
	if !hasMinus {
		return nil, fmt.Errorf("expected '-' token")
	}

	// Check if the next token is "$" for a field token
	if p.current().Type == TokenDollar {
		// Parse a field token
		node, err := p.parseFieldToken()
		if err != nil {
			return nil, fmt.Errorf("after '-': %w", err)
		}

		// Set the IsNot flag and update position
		node.IsNot = true
		node.Position.Start = startPos

		return node, nil
	} else {
		// Parse an unmodified token
		node, err := p.parseUnmodifiedToken()
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
}

// parseFieldToken parses a field token according to the grammar:
// field_token = "$" WORD | "$" WORD unmodified_token
func (p *Parser) parseFieldToken() (*Node, error) {
	startPos := p.getCurrentStartPos()

	// Consume the "$" token (we already checked it exists in parseToken)
	_, hasDollar := p.consumeToken(TokenDollar)
	if !hasDollar {
		return nil, fmt.Errorf("expected '$' token")
	}

	// Must be followed by a WORD
	wordToken, hasWord := p.consumeToken(TokenWord)
	if !hasWord {
		return nil, fmt.Errorf("'$' must be followed by a field name")
	}

	// Check if the word contains a colon
	fieldValue := wordToken.Value
	colonPos := strings.Index(fieldValue, ":")

	if colonPos == -1 {
		return nil, fmt.Errorf("field name must contain a colon to separate field and value")
	}

	// Extract field name
	fieldName := fieldValue[:colonPos]

	// Check if there's exactly one colon and it's the last character in the word
	if colonPos == len(fieldValue)-1 && strings.Count(fieldValue, ":") == 1 {
		// The colon is the last character, so we need to parse an unmodified_token
		// to get the search term
		unmodifiedNode, err := p.parseUnmodifiedToken()
		if err != nil {
			return nil, fmt.Errorf("after field name: %w", err)
		}
		if unmodifiedNode == nil {
			return nil, fmt.Errorf("field name with trailing colon must be followed by a search term")
		}

		// Create a search node with the field and the search term from the unmodified token
		return &Node{
			Type:       NodeTypeSearch,
			Position:   Position{Start: startPos, End: unmodifiedNode.Position.End},
			SearchType: unmodifiedNode.SearchType, // Preserve the search type from the unmodified token
			SearchTerm: unmodifiedNode.SearchTerm,
			Field:      fieldName,
		}, nil
	} else {
		// The colon is not the last character, so the search term is part of the word
		searchTerm := fieldValue[colonPos+1:]

		// Check if this is a numeric search term
		isNumeric, operator, numericValue, err := parseNumericSearchTerm(searchTerm)
		if err != nil {
			return nil, err
		}
		if isNumeric {
			return &Node{
				Type:       NodeTypeSearch,
				Position:   Position{Start: startPos, End: wordToken.Position.End},
				SearchType: SearchTypeNumeric,
				SearchTerm: numericValue,
				Field:      fieldName,
				Op:         operator,
			}, nil
		}

		// Not a numeric search, create a regular search node with the field
		return &Node{
			Type:       NodeTypeSearch,
			Position:   Position{Start: startPos, End: wordToken.Position.End},
			SearchType: SearchTypeExact,
			SearchTerm: searchTerm,
			Field:      fieldName,
		}, nil
	}
}

// parseNumericSearchTerm checks if a search term is a numeric comparison
// Returns:
// - ok: true if the search term is a valid numeric comparison
// - operator: the comparison operator (>, <, >=, <=)
// - value: the numeric value as a string
// - err: error if there's an operator but no valid numeric value
func parseNumericSearchTerm(searchTerm string) (ok bool, operator string, value string, err error) {
	// First check if it starts with a numeric operator
	if numericOperatorRegex.MatchString(searchTerm) {
		// We have an operator, now check if the rest is a valid number
		matches := numericSearchRegex.FindStringSubmatch(searchTerm)
		if matches == nil || len(matches) != 3 {
			// We have an operator but not a valid number
			opMatches := numericOperatorRegex.FindStringSubmatch(searchTerm)
			return false, opMatches[1], "", fmt.Errorf("numeric operator '%s' must be followed by a number", opMatches[1])
		}
		return true, matches[1], matches[2], nil
	}

	// No operator found
	return false, "", "", nil
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
		// Validate the regexp
		if _, err := regexp.Compile(cur.Value); err != nil {
			return nil, fmt.Errorf("invalid regular expression: %w", err)
		}
		return &Node{
			Type:       NodeTypeSearch,
			Position:   cur.Position,
			SearchType: SearchTypeRegexp,
			SearchTerm: cur.Value,
		}, nil
	}

	if cur.Type == TokenCRegexp {
		p.advance()
		// Validate the regexp
		if _, err := regexp.Compile(cur.Value); err != nil {
			return nil, fmt.Errorf("invalid case-sensitive regular expression: %w", err)
		}
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

	// Validate the tag name against the regex pattern
	if !TagRegexp.MatchString(wordToken.Value) {
		return nil, fmt.Errorf("invalid tag name: must match pattern %s", utilfn.SimpleTagRegexStr)
	}

	// Create the tag node
	// Special cases: #marked or #m uses the marked searcher, #userquery uses userquery searcher
	searchType := SearchTypeTag
	searchTerm := wordToken.Value

	if wordToken.Value == "marked" || wordToken.Value == "m" {
		// Special case for marked searcher
		searchType = SearchTypeMarked
		searchTerm = "" // Unset the value for marked searcher
	} else if wordToken.Value == "userquery" {
		// Special case for userquery searcher
		searchType = SearchTypeUserQuery
		searchTerm = "" // Unset the value for userquery searcher
	}

	node := &Node{
		Type:       NodeTypeSearch,
		Position:   Position{Start: startPos, End: wordToken.Position.End},
		SearchType: searchType,
		SearchTerm: searchTerm,
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
