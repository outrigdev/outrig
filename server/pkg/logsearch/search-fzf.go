// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// FzfSearcher implements fuzzy matching using the fzf algorithm
type FzfSearcher struct {
	field         string
	searchTerm    string
	pattern       []rune
	slab          *util.Slab
	caseSensitive bool
}

// MakeFzfSearcher creates a new FZF searcher
func MakeFzfSearcher(field string, searchTerm string, caseSensitive bool) (LogSearcher, error) {
	pattern := []rune(searchTerm)
	slab := util.MakeSlab(64, 4096)

	return &FzfSearcher{
		field:         field,
		searchTerm:    searchTerm,
		pattern:       pattern,
		slab:          slab,
		caseSensitive: caseSensitive,
	}, nil
}

// Match checks if the search object matches the fuzzy search pattern
func (s *FzfSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	var fieldText string
	
	// Apply case sensitivity
	if s.caseSensitive {
		fieldText = obj.GetField(s.field, 0)
	} else {
		fieldText = obj.GetField(s.field, FieldMod_ToLower)
	}
	
	// Convert the field to the format expected by fzf
	chars := util.ToChars([]byte(fieldText))

	// Perform fuzzy matching
	result, _ := algo.FuzzyMatchV2(false, true, true, &chars, s.pattern, true, s.slab)

	// If the score is positive, we have a match
	return result.Score > 0
}

// GetType returns the search type identifier
func (s *FzfSearcher) GetType() string {
	if s.caseSensitive {
		return SearchTypeFzfCase
	}
	return SearchTypeFzf
}
