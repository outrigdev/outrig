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
	// Parse the search term into an AST
	p := searchparser.NewParser(searchTerm)
	node := p.Parse()

	// Create a searcher from the AST node
	return MakeSearcherFromNode(&node)
}
