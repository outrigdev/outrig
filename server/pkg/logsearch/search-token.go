// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

// Use SearchToken from searchparser package
type SearchToken = searchparser.SearchToken

// MakeSearcherFromToken creates a searcher from a single token
func MakeSearcherFromToken(token SearchToken) (LogSearcher, error) {
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
		return MakeRegexpSearcher(token.SearchTerm)
	case SearchTypeFzf:
		return MakeFzfSearcher(token.SearchTerm)
	default:
		// Default to case-insensitive exact search
		return MakeExactSearcher(token.SearchTerm, false), nil
	}
}

// CreateSearchersFromTokens creates a slice of searchers from tokens
func CreateSearchersFromTokens(tokens []SearchToken) ([]LogSearcher, error) {
	// Handle empty tokens list
	if len(tokens) == 0 {
		return []LogSearcher{MakeAllSearcher()}, nil
	}
	
	searchers := make([]LogSearcher, len(tokens))
	
	for i, token := range tokens {
		searcher, err := MakeSearcherFromToken(token)
		if err != nil {
			return nil, err
		}
		searchers[i] = searcher
	}
	
	return searchers, nil
}
