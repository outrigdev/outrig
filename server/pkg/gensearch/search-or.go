// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

// OrSearcher implements a searcher that matches if any contained searcher matches
type OrSearcher struct {
	searchers []Searcher
}

// MakeOrSearcher creates a new OR searcher from a slice of searchers
func MakeOrSearcher(searchers []Searcher) Searcher {
	return &OrSearcher{
		searchers: searchers,
	}
}

// Match checks if the search object matches any contained searcher
func (s *OrSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	// If we have no searchers, nothing matches
	if len(s.searchers) == 0 {
		return false
	}

	// Check if the object matches any searcher
	for _, searcher := range s.searchers {
		if searcher.Match(sctx, obj) {
			return true
		}
	}

	return false
}

// GetType returns the search type identifier
func (s *OrSearcher) GetType() string {
	return SearchTypeOr
}
