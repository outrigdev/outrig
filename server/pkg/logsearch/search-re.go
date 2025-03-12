// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"fmt"
	"regexp"

	"github.com/outrigdev/outrig/pkg/ds"
)

// RegexpSearcher implements regular expression matching
type RegexpSearcher struct {
	searchTerm string
	regex      *regexp.Regexp
}

// MakeRegexpSearcher creates a new regexp searcher
func MakeRegexpSearcher(searchTerm string) (*RegexpSearcher, error) {
	// Compile the regex and return error if it fails
	regex, err := regexp.Compile(searchTerm)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", err)
	}
	
	return &RegexpSearcher{
		searchTerm: searchTerm,
		regex:      regex,
	}, nil
}

// Match checks if the log line matches the regular expression
func (s *RegexpSearcher) Match(line ds.LogLine) bool {
	return s.regex.MatchString(line.Msg)
}

// GetType returns the search type identifier
func (s *RegexpSearcher) GetType() string {
	return SearchTypeRegexp
}
