// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
)


// AllSearcher implements a searcher that matches everything
type AllSearcher struct{}

// MakeAllSearcher creates a new searcher that matches all log lines
func MakeAllSearcher() LogSearcher {
	return &AllSearcher{}
}

// Match always returns true
func (s *AllSearcher) Match(sctx *SearchContext, line ds.LogLine) bool {
	return true
}

// GetType returns the search type identifier
func (s *AllSearcher) GetType() string {
	return SearchTypeAll
}
