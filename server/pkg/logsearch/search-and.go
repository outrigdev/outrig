// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
)

// AndSearcher implements a searcher that requires all contained searchers to match
type AndSearcher struct {
	searchers []LogSearcher
}

// MakeAndSearcher creates a new AND searcher from a slice of searchers
func MakeAndSearcher(searchers []LogSearcher) LogSearcher {
	return &AndSearcher{
		searchers: searchers,
	}
}

// Match checks if the search object matches all contained searchers
func (s *AndSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	// If we have no searchers, everything matches
	if len(s.searchers) == 0 {
		return true
	}
	
	// Check if the object matches all searchers
	for _, searcher := range s.searchers {
		if !searcher.Match(sctx, obj) {
			return false
		}
	}
	
	return true
}

// GetType returns the search type identifier
func (s *AndSearcher) GetType() string {
	return SearchTypeAnd
}
