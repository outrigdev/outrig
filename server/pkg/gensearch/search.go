// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

const (
	SearchTypeExact      = "exact"
	SearchTypeExactCase  = "exactcase"
	SearchTypeRegexp     = "regexp"
	SearchTypeRegexpCase = "regexpcase"
	SearchTypeFzf        = "fzf"
	SearchTypeFzfCase    = "fzfcase"
	SearchTypeAnd        = "and"
	SearchTypeOr         = "or"
	SearchTypeAll        = "all"
	SearchTypeMarked     = "marked"
	SearchTypeNot        = "not"
	SearchTypeTag        = "tag"
	SearchTypeUserQuery  = "userquery"
)

const (
	FieldMod_ToLower = 1
)

// SearchContext contains runtime context for search operations
type SearchContext struct {
	MarkedLines map[int64]bool
	UserQuery   Searcher
	// Future fields can be added here without changing the interface
}

type SearchObject interface {
	GetField(fieldName string, fieldMods int) string
	GetTags() []string
	GetId() int64
}

// Searcher defines the interface for different search strategies
type Searcher interface {
	// Match checks if a search object matches the search criteria
	Match(sctx *SearchContext, obj SearchObject) bool

	// GetType returns the search type identifier
	GetType() string
}

// GetSearcher returns the appropriate searcher based on the search term
func GetSearcher(searchTerm string) (Searcher, error) {
	tokens := searchparser.TokenizeSearch(searchTerm)
	if len(tokens) == 0 {
		return MakeAllSearcher(), nil
	}
	if len(tokens) == 1 {
		return MakeSearcherFromToken(tokens[0])
	}

	// Check if we have OR tokens
	hasOrToken := false
	for _, token := range tokens {
		if token.Type == "or" && token.SearchTerm == "|" {
			hasOrToken = true
			break
		}
	}

	// Create searchers from tokens
	searchers, err := CreateSearchersFromTokens(tokens)
	if err != nil {
		return nil, err
	}

	// If we have OR tokens, the CreateSearchersFromTokens function will have already
	// created the appropriate OR searcher structure, so we can just return the first searcher
	if hasOrToken {
		return searchers[0], nil
	}

	// Otherwise, create an AND searcher as before
	return MakeAndSearcher(searchers), nil
}
