// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

// MarkedSearcher is a searcher that matches lines that are marked
type MarkedSearcher struct{}

// MakeMarkedSearcher creates a new MarkedSearcher
func MakeMarkedSearcher() Searcher {
	return &MarkedSearcher{}
}

// Match checks if a search object is marked
func (s *MarkedSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	lineNum := obj.GetId()
	_, exists := sctx.MarkedLines[lineNum]
	return exists
}

// GetType returns the search type identifier
func (s *MarkedSearcher) GetType() string {
	return SearchTypeMarked
}
