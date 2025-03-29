// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

// createSearcherFromUnmodifiedToken creates a searcher from a token without considering the IsNot field
func createSearcherFromUnmodifiedToken(token searchparser.SearchToken) (Searcher, error) {
	// Handle empty search term and special case for marked searcher
	if token.Type == SearchTypeMarked {
		return MakeMarkedSearcher(), nil
	} else if token.Type == SearchTypeUserQuery {
		return MakeUserQuerySearcher(), nil
	}

	// Handle empty search term
	if token.SearchTerm == "" {
		return MakeAllSearcher(), nil
	}

	// Create searcher based on token type
	switch token.Type {
	case SearchTypeExact:
		return MakeExactSearcher(token.Field, token.SearchTerm, false), nil
	case SearchTypeExactCase:
		return MakeExactSearcher(token.Field, token.SearchTerm, true), nil
	case SearchTypeRegexp:
		return MakeRegexpSearcher(token.Field, token.SearchTerm, false)
	case SearchTypeRegexpCase:
		return MakeRegexpSearcher(token.Field, token.SearchTerm, true)
	case SearchTypeFzf:
		return MakeFzfSearcher(token.Field, token.SearchTerm, false)
	case SearchTypeFzfCase:
		return MakeFzfSearcher(token.Field, token.SearchTerm, true)
	case SearchTypeTag:
		return MakeTagSearcher(token.Field, token.SearchTerm), nil
	default:
		// Default to case-insensitive exact search
		return MakeExactSearcher(token.Field, token.SearchTerm, false), nil
	}
}

// MakeSearcherFromToken creates a searcher from a single token
func MakeSearcherFromToken(token searchparser.SearchToken) (Searcher, error) {
	// Create the base searcher
	searcher, err := createSearcherFromUnmodifiedToken(token)
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
func CreateSearchersFromTokens(tokens []searchparser.SearchToken) ([]Searcher, error) {
	// Handle empty tokens list
	if len(tokens) == 0 {
		return []Searcher{MakeAllSearcher()}, nil
	}

	// Check if we have OR tokens
	hasOrToken := false
	for _, token := range tokens {
		if token.Type == "or" && token.SearchTerm == "|" {
			hasOrToken = true
			break
		}
	}

	// If no OR tokens, process normally
	if !hasOrToken {
		searchers := make([]Searcher, len(tokens))
		for i, token := range tokens {
			searcher, err := MakeSearcherFromToken(token)
			if err != nil {
				return nil, err
			}
			searchers[i] = searcher
		}
		return searchers, nil
	}

	// Process OR tokens
	var orSearchers []Searcher
	var currentGroup []Searcher

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// If this is an OR token, add the current group to the OR searchers
		if token.Type == "or" && token.SearchTerm == "|" {
			// If we have searchers in the current group, add them as an AND searcher
			if len(currentGroup) > 0 {
				orSearchers = append(orSearchers, MakeAndSearcher(currentGroup))
				currentGroup = nil
			} else {
				// Empty group, add an AllSearcher
				orSearchers = append(orSearchers, MakeAllSearcher())
			}
			continue
		}

		// Regular token, add to current group
		searcher, err := MakeSearcherFromToken(token)
		if err != nil {
			return nil, err
		}
		currentGroup = append(currentGroup, searcher)
	}

	// Add the last group if it's not empty
	if len(currentGroup) > 0 {
		orSearchers = append(orSearchers, MakeAndSearcher(currentGroup))
	}

	// If we only have one searcher, return it directly
	if len(orSearchers) == 1 {
		return []Searcher{orSearchers[0]}, nil
	}

	// Create an OR searcher with all the groups
	return []Searcher{MakeOrSearcher(orSearchers)}, nil
}
