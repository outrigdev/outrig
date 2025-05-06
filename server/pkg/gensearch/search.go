// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/searchparser"
)

// Re-export search type constants from searchparser package
const (
	SearchTypeExact      = searchparser.SearchTypeExact
	SearchTypeExactCase  = searchparser.SearchTypeExactCase
	SearchTypeRegexp     = searchparser.SearchTypeRegexp
	SearchTypeRegexpCase = searchparser.SearchTypeRegexpCase
	SearchTypeFzf        = searchparser.SearchTypeFzf
	SearchTypeFzfCase    = searchparser.SearchTypeFzfCase
	SearchTypeNot        = searchparser.SearchTypeNot
	SearchTypeTag        = searchparser.SearchTypeTag
	SearchTypeUserQuery  = searchparser.SearchTypeUserQuery
	SearchTypeMarked     = searchparser.SearchTypeMarked
	SearchTypeNumeric    = searchparser.SearchTypeNumeric

	// Additional constants not in searchparser
	SearchTypeAnd = "and"
	SearchTypeOr  = "or"
	SearchTypeAll = "all"
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
	p := searchparser.NewParser(searchTerm)
	node := p.Parse()
	searcher, err := MakeSearcherFromNode(node)
	if err != nil {
		return nil, err
	}
	if searcher == nil {
		return MakeAllSearcher(), nil
	}
	return searcher, nil
}

// GetSearcherWithErrors returns both a searcher and any error spans found in the search term
func GetSearcherWithErrors(searchTerm string) (Searcher, []rpctypes.SearchErrorSpan, error) {
	p := searchparser.NewParser(searchTerm)
	node := p.Parse()

	// Extract error spans from the AST
	errorSpans := ExtractErrorSpans(node)

	// Create a searcher from the AST
	searcher, err := MakeSearcherFromNode(node)
	if err != nil {
		return nil, errorSpans, err
	}

	if searcher == nil {
		return MakeAllSearcher(), errorSpans, nil
	}

	return searcher, errorSpans, nil
}
