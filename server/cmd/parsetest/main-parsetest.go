// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

// prettyPrintTokens formats tokens in a concise one-line format
func prettyPrintTokens(tokens []searchparser.Token) string {
	var sb strings.Builder

	for _, token := range tokens {
		// Format: TOKENTYPE [position] "value" with aligned columns
		// Token type padded to 7 chars (max length), positions use %2d, and 2-space indent
		line := fmt.Sprintf("  %-7s [%2d:%2d]", token.Type, token.Position.Start, token.Position.End)

		if token.Value != "" {
			line += fmt.Sprintf(" %q", token.Value)
		}
		if token.Incomplete {
			line += " (incomplete)"
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// nodeToJSON converts a Node to a JSON string for visualization
func nodeToJSON(node *searchparser.Node) string {
	type jsonNode struct {
		Type         string     `json:"type"`
		Position     string     `json:"position"`
		SearchType   string     `json:"searchType,omitempty"`
		SearchTerm   string     `json:"searchTerm,omitempty"`
		Field        string     `json:"field,omitempty"`
		Op           string     `json:"op,omitempty"`
		IsNot        bool       `json:"isNot,omitempty"`
		ErrorMessage string     `json:"errorMessage,omitempty"`
		Children     []jsonNode `json:"children,omitempty"`
	}

	var convertNode func(*searchparser.Node) jsonNode
	convertNode = func(n *searchparser.Node) jsonNode {
		if n == nil {
			return jsonNode{}
		}

		jn := jsonNode{
			Type:     n.Type,
			Position: fmt.Sprintf("[%d:%d]", n.Position.Start, n.Position.End),
		}

		if n.Type == "search" {
			jn.SearchType = n.SearchType
			jn.SearchTerm = n.SearchTerm
			jn.Field = n.Field
			jn.Op = n.Op
			jn.IsNot = n.IsNot
		} else if n.Type == "error" {
			jn.ErrorMessage = n.ErrorMessage
		} else if len(n.Children) > 0 {
			jn.Children = make([]jsonNode, len(n.Children))
			for i, child := range n.Children {
				childCopy := child // Create a copy to get address
				jn.Children[i] = convertNode(childCopy)
			}
		}

		return jn
	}

	jn := convertNode(node)
	jsonBytes, err := json.MarshalIndent(jn, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling to JSON: %v", err)
	}

	return string(jsonBytes)
}

func main() {
	// Define command line flags
	showJSON := flag.Bool("json", false, "Show JSON tree output")
	flag.Parse()

	// Get queries from command line arguments or use defaults
	var queries []string
	if flag.NArg() > 0 {
		queries = flag.Args()
	} else {
		// Default test queries if none provided
		queries = []string{
			// "simple",
			// "hello world",
			// `"hello" world | 'foo'`,
			// `"hello"mike`,
			// `$ name: hello`,
			`(hello world`,
			`(line 1 | line 2 | line 3) (line (3 | 2)) ((3))`,
			`$goid:'823'`,
			`:>500`,
			`$goid:>=500`,
			`#m`,
			// `"abc"def"hello"/foo/"bar"`,
			// `"hello" mike`,
			// `$name:hello`,
			// `$name: hello`,
		}
	}

	// Process each query
	for i, query := range queries {
		fmt.Printf("\n=== Test Query %d: %q ===\n\n", i+1, query)

		// Tokenization
		tokenizer := searchparser.NewTokenizer(query)
		tokens := tokenizer.GetAllTokens()

		fmt.Println("tokenization:")
		fmt.Println(prettyPrintTokens(tokens))
		fmt.Println()

		// Parsing
		parser := searchparser.NewParser(query)
		ast := parser.Parse()

		fmt.Println("parse tree:")
		fmt.Println(ast.PrettyPrint("  ", query))
		fmt.Println()

		// Only show JSON output if --json flag is provided
		if *showJSON {
			fmt.Println("json tree:")
			fmt.Println(nodeToJSON(ast))
			fmt.Println()
		}

		fmt.Println(strings.Repeat("=", 80))
	}
}
