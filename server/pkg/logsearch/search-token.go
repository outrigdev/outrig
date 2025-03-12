// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"strings"
)

// SearchToken represents a single token in a search query
type SearchToken struct {
	Type       string // The search type (exact, regexp, fzf, etc.)
	SearchTerm string // The actual search term
}

// TokenizeSearch splits a search string into tokens based on whitespace
func TokenizeSearch(searchType string, searchString string) []SearchToken {
	if searchString == "" {
		return []SearchToken{}
	}

	// Split the search string by whitespace
	terms := strings.Fields(searchString)
	tokens := make([]SearchToken, len(terms))

	// Create a token for each term with the specified search type
	for i, term := range terms {
		tokens[i] = SearchToken{
			Type:       searchType,
			SearchTerm: term,
		}
	}

	return tokens
}

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
