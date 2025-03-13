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
	caseSensitive bool
}

// MakeRegexpSearcher creates a new regexp searcher
func MakeRegexpSearcher(searchTerm string, caseSensitive bool) (*RegexpSearcher, error) {
	var regex *regexp.Regexp
	var err error
	
	if caseSensitive {
		// Case-sensitive regexp
		regex, err = regexp.Compile(searchTerm)
	} else {
		// Case-insensitive regexp
		regex, err = regexp.Compile("(?i)" + searchTerm)
	}
	
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", err)
	}
	
	return &RegexpSearcher{
		searchTerm: searchTerm,
		regex:      regex,
		caseSensitive: caseSensitive,
	}, nil
}

// Match checks if the log line matches the regular expression
func (s *RegexpSearcher) Match(line ds.LogLine) bool {
	return s.regex.MatchString(line.Msg)
}

// GetType returns the search type identifier
func (s *RegexpSearcher) GetType() string {
	if s.caseSensitive {
		return SearchTypeRegexpCase
	}
	return SearchTypeRegexp
}
