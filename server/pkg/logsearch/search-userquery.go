// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logsearch

import (
)

// UserQuerySearcher is a searcher that delegates to the UserQuery field in SearchContext
type UserQuerySearcher struct{}

// MakeUserQuerySearcher creates a new UserQuerySearcher
func MakeUserQuerySearcher() LogSearcher {
	return &UserQuerySearcher{}
}

// Match delegates to the UserQuery searcher in SearchContext
func (s *UserQuerySearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	if sctx.UserQuery == nil {
		return true
	}
	return sctx.UserQuery.Match(sctx, obj)
}

// GetType returns the search type identifier
func (s *UserQuerySearcher) GetType() string {
	return SearchTypeUserQuery
}
