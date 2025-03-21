// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
)

// MarkedSearcher is a searcher that matches lines that are marked
type MarkedSearcher struct {}

// MakeMarkedSearcher creates a new MarkedSearcher
func MakeMarkedSearcher() LogSearcher {
	return &MarkedSearcher{}
}

// Match checks if a log line is marked
func (s *MarkedSearcher) Match(sctx *SearchContext, line ds.LogLine) bool {
	_, exists := sctx.MarkedLines[line.LineNum]
	return exists
}

// GetType returns the search type identifier
func (s *MarkedSearcher) GetType() string {
	return SearchTypeMarked
}
