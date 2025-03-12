// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
)

const (
	SearchTypeExact     = "exact"
	SearchTypeExactCase = "exactcase"
	SearchTypeRegexp    = "regexp"
	SearchTypeFzf       = "fzf"
)

// LogSearcher defines the interface for different search strategies
type LogSearcher interface {
	// Match checks if a log line matches the search criteria
	Match(line ds.LogLine) bool
	
	// GetType returns the search type identifier
	GetType() string
}

// GetSearcher returns the appropriate searcher based on the search type
func GetSearcher(searchType string, searchTerm string) (LogSearcher, error) {
	switch searchType {
	case SearchTypeExact:
		return MakeExactSearcher(searchTerm, false), nil
	case SearchTypeExactCase:
		return MakeExactSearcher(searchTerm, true), nil
	case SearchTypeRegexp:
		return MakeRegexpSearcher(searchTerm)
	case SearchTypeFzf:
		return MakeFzfSearcher(searchTerm)
	default:
		// Default to case-insensitive exact search
		return MakeExactSearcher(searchTerm, false), nil
	}
}
