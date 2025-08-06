// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

// ExtractErrorSpans extracts all error nodes from the AST
func ExtractErrorSpans(node *searchparser.Node) []rpctypes.SearchErrorSpan {
	if node == nil {
		return nil
	}

	var spans []rpctypes.SearchErrorSpan

	// Check if this node is an error node
	if node.Type == searchparser.NodeTypeError {
		spans = append(spans, rpctypes.SearchErrorSpan{
			Start:        node.Position.Start,
			End:          node.Position.End,
			ErrorMessage: node.ErrorMessage,
		})
	}

	// Recursively check children (for AND/OR nodes)
	for _, child := range node.Children {
		childSpans := ExtractErrorSpans(child)
		spans = append(spans, childSpans...)
	}

	return spans
}

// MakeSearcherFromNode creates a searcher from an AST node
func MakeSearcherFromNode(node *searchparser.Node) (Searcher, error) {
	if node == nil {
		return nil, nil
	}

	switch node.Type {
	case searchparser.NodeTypeSearch:
		// Create a searcher for a leaf node
		searcher, err := createSearcherFromSearchNode(node)
		if err != nil {
			return nil, err
		}
		if searcher == nil {
			return nil, nil
		}
		if node.IsNot {
			return MakeNotSearcher(searcher), nil
		}
		return searcher, nil

	case searchparser.NodeTypeError:
		return nil, nil

	case searchparser.NodeTypeAnd:
		var searchers []Searcher
		for _, child := range node.Children {
			searcher, err := MakeSearcherFromNode(child)
			if err != nil {
				return nil, err
			}
			if searcher == nil {
				continue
			}
			searchers = append(searchers, searcher)
		}
		if len(searchers) == 0 {
			return nil, nil
		}
		if len(searchers) == 1 {
			return searchers[0], nil
		}
		return MakeAndSearcher(searchers), nil

	case searchparser.NodeTypeOr:
		var searchers []Searcher
		for _, child := range node.Children {
			searcher, err := MakeSearcherFromNode(child)
			if err != nil {
				return nil, err
			}
			if searcher == nil {
				continue
			}
			searchers = append(searchers, searcher)
		}
		if len(searchers) == 0 {
			return nil, nil
		}
		if len(searchers) == 1 {
			return searchers[0], nil
		}
		return MakeOrSearcher(searchers), nil

	default:
		return nil, nil
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
	case SearchTypeNumeric:
		return MakeNumericSearcher(node.Field, node.SearchTerm, node.Op)
	case SearchTypeColorFilter:
		return MakeColorFilterSearcher(), nil
	default:
		// Default to case-insensitive exact search
		return MakeExactSearcher(node.Field, node.SearchTerm, false), nil
	}
}
