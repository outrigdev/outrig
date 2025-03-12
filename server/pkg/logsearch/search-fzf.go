// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"strings"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	"github.com/outrigdev/outrig/pkg/ds"
)

// FzfSearcher implements fuzzy matching using the fzf algorithm
type FzfSearcher struct {
	searchTerm string
	pattern    []rune
	slab       *util.Slab
}

// MakeFzfSearcher creates a new FZF searcher
func MakeFzfSearcher(searchTerm string) (*FzfSearcher, error) {
	pattern := []rune(searchTerm)
	slab := util.MakeSlab(64, 4096)

	return &FzfSearcher{
		searchTerm: searchTerm,
		pattern:    pattern,
		slab:       slab,
	}, nil
}

// Match checks if the log line matches the fuzzy search pattern
func (s *FzfSearcher) Match(line ds.LogLine) bool {
	// Convert the message to the format expected by fzf
	msg := strings.ToLower(line.Msg)
	chars := util.ToChars([]byte(msg))

	// Perform fuzzy matching
	result, _ := algo.FuzzyMatchV2(false, true, true, &chars, s.pattern, true, s.slab)

	// If the score is positive, we have a match
	return result.Score > 0
}

// GetType returns the search type identifier
func (s *FzfSearcher) GetType() string {
	return SearchTypeFzf
}
