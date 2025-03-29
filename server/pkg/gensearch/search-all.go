// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

// AllSearcher implements a searcher that matches everything
type AllSearcher struct{}

// MakeAllSearcher creates a new searcher that matches all log lines
func MakeAllSearcher() Searcher {
	return &AllSearcher{}
}

// Match always returns true
func (s *AllSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	return true
}

// GetType returns the search type identifier
func (s *AllSearcher) GetType() string {
	return SearchTypeAll
}
