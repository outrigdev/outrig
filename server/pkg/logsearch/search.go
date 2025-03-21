// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

const (
	SearchTypeExact      = "exact"
	SearchTypeExactCase  = "exactcase"
	SearchTypeRegexp     = "regexp"
	SearchTypeRegexpCase = "regexpcase"
	SearchTypeFzf        = "fzf"
	SearchTypeFzfCase    = "fzfcase"
	SearchTypeAnd        = "and"
	SearchTypeOr         = "or"
	SearchTypeAll        = "all"
	SearchTypeMarked     = "marked"
	SearchTypeNot        = "not"
)

// LogSearcher defines the interface for different search strategies
type LogSearcher interface {
	// Match checks if a log line matches the search criteria
	Match(line ds.LogLine) bool

	// GetType returns the search type identifier
	GetType() string
}

// GetSearcher returns the appropriate searcher based on the search term
// The manager parameter is required for creating marked searchers
func GetSearcher(searchTerm string, manager *SearchManager) (LogSearcher, error) {
	tokens := searchparser.TokenizeSearch(searchTerm)
	if len(tokens) == 0 {
		return MakeAllSearcher(), nil
	}
	if len(tokens) == 1 {
		return MakeSearcherFromToken(tokens[0], manager)
	}
	
	// Check if we have OR tokens
	hasOrToken := false
	for _, token := range tokens {
		if token.Type == "or" && token.SearchTerm == "|" {
			hasOrToken = true
			break
		}
	}
	
	// Create searchers from tokens
	searchers, err := CreateSearchersFromTokens(tokens, manager)
	if err != nil {
		return nil, err
	}
	
	// If we have OR tokens, the CreateSearchersFromTokens function will have already
	// created the appropriate OR searcher structure, so we can just return the first searcher
	if hasOrToken {
		return searchers[0], nil
	}
	
	// Otherwise, create an AND searcher as before
	return MakeAndSearcher(searchers), nil
}
