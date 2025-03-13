// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

const (
	SearchTypeExact     = "exact"
	SearchTypeExactCase = "exactcase"
	SearchTypeRegexp    = "regexp"
	SearchTypeRegexpCase = "regexpcase"
	SearchTypeFzf       = "fzf"
	SearchTypeFzfCase   = "fzfcase"
	SearchTypeAnd       = "and"
	SearchTypeAll       = "all"
)

// LogSearcher defines the interface for different search strategies
type LogSearcher interface {
	// Match checks if a log line matches the search criteria
	Match(line ds.LogLine) bool

	// GetType returns the search type identifier
	GetType() string
}

// GetSearcher returns the appropriate searcher based on the search term
// If searchType is provided, it will be used as the default type for all tokens
// Otherwise, "exact" will be used as the default type
func GetSearcher(searchType string, searchTerm string) (LogSearcher, error) {
	// If searchType is empty, default to "exact"
	if searchType == "" {
		searchType = SearchTypeExact
	}
	
	tokens := searchparser.TokenizeSearch(searchType, searchTerm)
	if len(tokens) == 0 {
		return MakeAllSearcher(), nil
	}
	if len(tokens) == 1 {
		return MakeSearcherFromToken(tokens[0])
	}
	searchers, err := CreateSearchersFromTokens(tokens)
	if err != nil {
		return nil, err
	}
	return MakeAndSearcher(searchers), nil
}
