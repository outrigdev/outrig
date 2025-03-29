// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"strings"
)

// TagSearcher implements tag matching
type TagSearcher struct {
	searchTerm string // The tag to search for (without the #)
	exactMatch bool   // Whether to require exact matching
}

// MakeTagSearcher creates a new tag searcher
func MakeTagSearcher(field string, searchTerm string) Searcher {
	// If the search term ends with a slash, it indicates exact matching
	exactMatch := false
	if strings.HasSuffix(searchTerm, "/") {
		searchTerm = strings.TrimSuffix(searchTerm, "/")
		exactMatch = true
	}

	// Tags are always case-insensitive
	searchTerm = strings.ToLower(searchTerm)

	return &TagSearcher{
		searchTerm: searchTerm,
		exactMatch: exactMatch,
	}
}

// Match checks if the search object contains a tag that matches the search term
func (s *TagSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	tags := obj.GetTags()

	for _, tag := range tags {
		if s.exactMatch {
			// Exact match
			if tag == s.searchTerm {
				return true
			}
		} else {
			// Prefix match
			if strings.HasPrefix(tag, s.searchTerm) {
				return true
			}
		}
	}

	return false
}

// GetType returns the search type identifier
func (s *TagSearcher) GetType() string {
	return SearchTypeTag
}
