// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
	"github.com/outrigdev/outrig/pkg/ds"
)

// MarkedSearcher is a searcher that matches lines that are marked
type MarkedSearcher struct {
	manager *SearchManager
}

// MakeMarkedSearcher creates a new MarkedSearcher
func MakeMarkedSearcher(manager *SearchManager) LogSearcher {
	return &MarkedSearcher{
		manager: manager,
	}
}

// Match checks if a log line is marked
func (s *MarkedSearcher) Match(line ds.LogLine) bool {
	return s.manager.IsLineMarked(line.LineNum)
}

// GetType returns the search type identifier
func (s *MarkedSearcher) GetType() string {
	return SearchTypeMarked
}
