// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

// ColorFilterSearcher implements a searcher that always returns true
// This is used for color filtering which is handled separately
type ColorFilterSearcher struct{}

// MakeColorFilterSearcher creates a new ColorFilter searcher
func MakeColorFilterSearcher() Searcher {
	return &ColorFilterSearcher{}
}

// Match always returns true as color filtering doesn't filter out results
func (s *ColorFilterSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	return true
}

// GetType returns the search type identifier
func (s *ColorFilterSearcher) GetType() string {
	return SearchTypeColorFilter
}