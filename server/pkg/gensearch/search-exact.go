// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"strings"
)

// ExactSearcher implements exact string matching with case sensitivity option
type ExactSearcher struct {
	field         string
	searchTerm    string
	caseSensitive bool
}

// MakeExactSearcher creates a new exact match searcher
func MakeExactSearcher(field string, searchTerm string, caseSensitive bool) Searcher {
	if !caseSensitive {
		searchTerm = strings.ToLower(searchTerm)
	}
	return &ExactSearcher{
		field:         field,
		searchTerm:    searchTerm,
		caseSensitive: caseSensitive,
	}
}

// Match checks if the search object contains the search term
func (s *ExactSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	var fieldText string
	if !s.caseSensitive {
		fieldText = obj.GetField(s.field, FieldMod_ToLower)
	} else {
		fieldText = obj.GetField(s.field, 0)
	}

	return strings.Contains(fieldText, s.searchTerm)
}

// GetType returns the search type identifier
func (s *ExactSearcher) GetType() string {
	if s.caseSensitive {
		return SearchTypeExactCase
	}
	return SearchTypeExact
}
