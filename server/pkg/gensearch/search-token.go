// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

// MakeSearcherFromNode creates a searcher from an AST node
func MakeSearcherFromNode(node *searchparser.Node) (Searcher, error) {
	if node == nil {
		return MakeAllSearcher(), nil
	}

	switch node.Type {
	case searchparser.NodeTypeSearch:
		// Create a searcher for a leaf node
		searcher, err := createSearcherFromSearchNode(node)
		if err != nil {
			return nil, err
		}
		
		// If this is a negated search, wrap it with a not searcher
		if node.IsNot {
			return MakeNotSearcher(searcher), nil
		}
		return searcher, nil
		
	case searchparser.NodeTypeAnd:
		// Create an AND searcher with all child searchers
		if len(node.Children) == 0 {
			return MakeAllSearcher(), nil
		}
		
		searchers := make([]Searcher, 0, len(node.Children))
		for i := range node.Children {
			// Get a pointer to the child node
			childPtr := &node.Children[i]
			searcher, err := MakeSearcherFromNode(childPtr)
			if err != nil {
				return nil, err
			}
			searchers = append(searchers, searcher)
		}
		
		// If there's only one searcher, return it directly
		if len(searchers) == 1 {
			return searchers[0], nil
		}
		
		return MakeAndSearcher(searchers), nil
		
	case searchparser.NodeTypeOr:
		// Create an OR searcher with all child searchers
		if len(node.Children) == 0 {
			return MakeAllSearcher(), nil
		}
		
		searchers := make([]Searcher, 0, len(node.Children))
		for i := range node.Children {
			// Get a pointer to the child node
			childPtr := &node.Children[i]
			searcher, err := MakeSearcherFromNode(childPtr)
			if err != nil {
				return nil, err
			}
			searchers = append(searchers, searcher)
		}
		
		// If there's only one searcher, return it directly
		if len(searchers) == 1 {
			return searchers[0], nil
		}
		
		return MakeOrSearcher(searchers), nil
		
	default:
		// Unknown node type, return an all searcher
		return MakeAllSearcher(), nil
	}
}

// createSearcherFromSearchNode creates a searcher from a search node
func createSearcherFromSearchNode(node *searchparser.Node) (Searcher, error) {
	// Handle special cases
	if node.SearchType == SearchTypeMarked {
		return MakeMarkedSearcher(), nil
	} else if node.SearchType == SearchTypeUserQuery {
		return MakeUserQuerySearcher(), nil
	}

	// Handle empty search term
	if node.SearchTerm == "" {
		return MakeAllSearcher(), nil
	}

	// Create searcher based on search type
	switch node.SearchType {
	case SearchTypeExact:
		return MakeExactSearcher(node.Field, node.SearchTerm, false), nil
	case SearchTypeExactCase:
		return MakeExactSearcher(node.Field, node.SearchTerm, true), nil
	case SearchTypeRegexp:
		return MakeRegexpSearcher(node.Field, node.SearchTerm, false)
	case SearchTypeRegexpCase:
		return MakeRegexpSearcher(node.Field, node.SearchTerm, true)
	case SearchTypeFzf:
		return MakeFzfSearcher(node.Field, node.SearchTerm, false)
	case SearchTypeFzfCase:
		return MakeFzfSearcher(node.Field, node.SearchTerm, true)
	case SearchTypeTag:
		return MakeTagSearcher(node.Field, node.SearchTerm), nil
	default:
		// Default to case-insensitive exact search
		return MakeExactSearcher(node.Field, node.SearchTerm, false), nil
	}
}
