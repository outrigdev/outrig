// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"regexp"
	"strings"
)

// TagSearcher implements tag matching with word boundary checking
type TagSearcher struct {
	searchTerm string         // The tag to search for (without the #)
	exactMatch bool           // Whether to require exact matching
	tagRegexp  *regexp.Regexp // Pre-compiled regexp for matching
}

// MakeTagSearcher creates a new tag searcher
func MakeTagSearcher(searchTerm string) LogSearcher {
	// If the search term ends with a slash, it indicates exact matching
	exactMatch := false
	if strings.HasSuffix(searchTerm, "/") {
		searchTerm = strings.TrimSuffix(searchTerm, "/")
		exactMatch = true
	}

	// Tags are always case-insensitive
	searchTerm = strings.ToLower(searchTerm)

	// Escape any regexp special characters in the search term
	escapedTerm := regexp.QuoteMeta(searchTerm)

	// Create the regexp pattern
	var pattern string
	if exactMatch {
		// For exact match, use word boundary on both sides
		// (?i) makes it case-insensitive
		// (^|\\s) matches start of line or whitespace
		// (#escapedTerm) matches the tag exactly
		// ($|\\s) matches end of line or whitespace
		pattern = `(?i)(^|\s)(#` + escapedTerm + `)($|\s)`
	} else {
		// For non-exact match, only require word boundary at start
		// This will match both "#foo" and "#foobar" when searching for "#foo"
		pattern = `(?i)(^|\s)(#` + escapedTerm + `)`
	}

	// Compile the regexp
	re := regexp.MustCompile(pattern)

	return &TagSearcher{
		searchTerm: searchTerm,
		exactMatch: exactMatch,
		tagRegexp:  re,
	}
}

// Match checks if the search object contains a tag that matches the search term
func (s *TagSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	// Use the pre-compiled regexp to find matches
	field := obj.GetField("", 0)
	return s.tagRegexp.MatchString(field)
}

// GetType returns the search type identifier
func (s *TagSearcher) GetType() string {
	return SearchTypeTag
}
