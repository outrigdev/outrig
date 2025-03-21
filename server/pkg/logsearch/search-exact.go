// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"strings"

	"github.com/outrigdev/outrig/pkg/ds"
)

// ExactSearcher implements exact string matching with case sensitivity option
type ExactSearcher struct {
	searchTerm    string
	caseSensitive bool
}

// MakeExactSearcher creates a new exact match searcher
func MakeExactSearcher(searchTerm string, caseSensitive bool) LogSearcher {
	if !caseSensitive {
		searchTerm = strings.ToLower(searchTerm)
	}
	return &ExactSearcher{
		searchTerm:    searchTerm,
		caseSensitive: caseSensitive,
	}
}

// Match checks if the log line contains the search term
func (s *ExactSearcher) Match(sctx *SearchContext, line ds.LogLine) bool {
	msg := line.Msg
	if !s.caseSensitive {
		msg = strings.ToLower(msg)
	}
	
	return strings.Contains(msg, s.searchTerm)
}

// GetType returns the search type identifier
func (s *ExactSearcher) GetType() string {
	if s.caseSensitive {
		return SearchTypeExactCase
	}
	return SearchTypeExact
}
