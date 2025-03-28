// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// FzfSearcher implements fuzzy matching using the fzf algorithm
type FzfSearcher struct {
	searchTerm    string
	pattern       []rune
	slab          *util.Slab
	caseSensitive bool
}

// MakeFzfSearcher creates a new FZF searcher
func MakeFzfSearcher(searchTerm string, caseSensitive bool) (LogSearcher, error) {
	pattern := []rune(searchTerm)
	slab := util.MakeSlab(64, 4096)

	return &FzfSearcher{
		searchTerm:    searchTerm,
		pattern:       pattern,
		slab:          slab,
		caseSensitive: caseSensitive,
	}, nil
}

// Match checks if the search object matches the fuzzy search pattern
func (s *FzfSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	var field string
	
	// Apply case sensitivity
	if s.caseSensitive {
		field = obj.GetField("", 0)
	} else {
		field = obj.GetField("", FieldMod_ToLower)
	}
	
	// Convert the field to the format expected by fzf
	chars := util.ToChars([]byte(field))

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
