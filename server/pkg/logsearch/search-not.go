// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
)

// NotSearcher implements a searcher that inverts the result of another searcher
type NotSearcher struct {
	searcher LogSearcher
}

// MakeNotSearcher creates a new NOT searcher that inverts the result of the provided searcher
func MakeNotSearcher(searcher LogSearcher) LogSearcher {
	return &NotSearcher{
		searcher: searcher,
	}
}

// Match checks if the log line does NOT match the contained searcher
func (s *NotSearcher) Match(sctx *SearchContext, line ds.LogLine) bool {
	// Invert the match result of the contained searcher
	return !s.searcher.Match(sctx, line)
}

// GetType returns the search type identifier
func (s *NotSearcher) GetType() string {
	return SearchTypeNot
}
