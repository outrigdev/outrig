// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"strings"

	"github.com/outrigdev/outrig/pkg/ds"
)

const (
	SearchTypeExact     = "exact"
	SearchTypeExactCase = "exactcase"
	SearchTypeRegexp    = "regexp"
	SearchTypeFzf       = "fzf"
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

// GetSearcher returns the appropriate searcher based on the search type and term
func GetSearcher(searchType string, searchTerm string) (LogSearcher, error) {
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		return MakeAllSearcher(), nil
	}
	tokens := TokenizeSearch(searchType, searchTerm)
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
