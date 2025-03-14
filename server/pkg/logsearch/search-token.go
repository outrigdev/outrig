// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"fmt"

	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

// Use SearchToken from searchparser package
type SearchToken = searchparser.SearchToken

// createSearcherFromUnmodifiedToken creates a searcher from a token without considering the IsNot field
func createSearcherFromUnmodifiedToken(token SearchToken, manager *SearchManager) (LogSearcher, error) {
	// Handle empty search term and special case for marked searcher
	if token.Type == SearchTypeMarked {
		if manager == nil {
			return nil, fmt.Errorf("marked searcher requires a search manager")
		}
		return MakeMarkedSearcher(manager), nil
	}

	// Handle empty search term
	if token.SearchTerm == "" {
		return MakeAllSearcher(), nil
	}

	// Create searcher based on token type
	switch token.Type {
	case SearchTypeExact:
		return MakeExactSearcher(token.SearchTerm, false), nil
	case SearchTypeExactCase:
		return MakeExactSearcher(token.SearchTerm, true), nil
	case SearchTypeRegexp:
		return MakeRegexpSearcher(token.SearchTerm, false)
	case SearchTypeRegexpCase:
		return MakeRegexpSearcher(token.SearchTerm, true)
	case SearchTypeFzf:
		return MakeFzfSearcher(token.SearchTerm, false)
	case SearchTypeFzfCase:
		return MakeFzfSearcher(token.SearchTerm, true)
	default:
		// Default to case-insensitive exact search
		return MakeExactSearcher(token.SearchTerm, false), nil
	}
}

// MakeSearcherFromToken creates a searcher from a single token
func MakeSearcherFromToken(token SearchToken, manager *SearchManager) (LogSearcher, error) {
	// Create the base searcher
	searcher, err := createSearcherFromUnmodifiedToken(token, manager)
	if err != nil {
		return nil, err
	}
	
	// If this is a not token, wrap it with a not searcher
	if token.IsNot {
		return MakeNotSearcher(searcher), nil
	}
	
	return searcher, nil
}

// CreateSearchersFromTokens creates a slice of searchers from tokens
func CreateSearchersFromTokens(tokens []SearchToken, manager *SearchManager) ([]LogSearcher, error) {
	// Handle empty tokens list
	if len(tokens) == 0 {
		return []LogSearcher{MakeAllSearcher()}, nil
	}

	searchers := make([]LogSearcher, len(tokens))

	for i, token := range tokens {
		searcher, err := MakeSearcherFromToken(token, manager)
		if err != nil {
			return nil, err
		}
		searchers[i] = searcher
	}

	return searchers, nil
}
