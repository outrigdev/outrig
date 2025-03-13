// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
)

// MarkedSearcher is a searcher that matches lines that are marked
type MarkedSearcher struct {
	markedLines map[int64]bool
}

// MakeMarkedSearcher creates a new MarkedSearcher
func MakeMarkedSearcher(manager *SearchManager) LogSearcher {
	// Get a copy of the marked lines map
	markedLines := manager.GetMarkedLinesMap()

	return &MarkedSearcher{
		markedLines: markedLines,
	}
}

// Match checks if a log line is marked
func (s *MarkedSearcher) Match(line ds.LogLine) bool {
	_, exists := s.markedLines[line.LineNum]
	return exists
}

// GetType returns the search type identifier
func (s *MarkedSearcher) GetType() string {
	return SearchTypeMarked
}
