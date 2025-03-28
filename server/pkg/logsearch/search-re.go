// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"fmt"
	"regexp"
)

// RegexpSearcher implements regular expression matching
type RegexpSearcher struct {
	field      string
	searchTerm string
	regex      *regexp.Regexp
	caseSensitive bool
}

// MakeRegexpSearcher creates a new regexp searcher
func MakeRegexpSearcher(field string, searchTerm string, caseSensitive bool) (LogSearcher, error) {
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
		field:      field,
		searchTerm: searchTerm,
		regex:      regex,
		caseSensitive: caseSensitive,
	}, nil
}

// Match checks if the search object matches the regular expression
func (s *RegexpSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	fieldText := obj.GetField(s.field, 0)
	return s.regex.MatchString(fieldText)
}

// GetType returns the search type identifier
func (s *RegexpSearcher) GetType() string {
	if s.caseSensitive {
		return SearchTypeRegexpCase
	}
	return SearchTypeRegexp
}
