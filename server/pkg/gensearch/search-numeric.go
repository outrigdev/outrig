// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"strconv"
)

// NumericSearcher implements numeric comparison operations (>, <, >=, <=)
type NumericSearcher struct {
	field     string
	searchNum int
	operator  string
}

// MakeNumericSearcher creates a new numeric comparison searcher
func MakeNumericSearcher(field string, searchTerm string, operator string) (Searcher, error) {
	// Convert the search term to an integer
	searchNum, err := strconv.Atoi(searchTerm)
	if err != nil {
		return nil, err
	}

	return &NumericSearcher{
		field:     field,
		searchNum: searchNum,
		operator:  operator,
	}, nil
}

// Match checks if the numeric field value satisfies the comparison
func (s *NumericSearcher) Match(sctx *SearchContext, obj SearchObject) bool {
	// Get the field value as a string
	fieldText := obj.GetField(s.field, 0)
	if fieldText == "" {
		return false
	}

	// Try to convert the field value to an integer
	fieldNum, err := strconv.Atoi(fieldText)
	if err != nil {
		return false
	}
	// Perform the comparison based on the operator
	switch s.operator {
	case ">":
		return fieldNum > s.searchNum
	case "<":
		return fieldNum < s.searchNum
	case ">=":
		return fieldNum >= s.searchNum
	case "<=":
		return fieldNum <= s.searchNum
	default:
		return false
	}
}

// GetType returns the search type identifier
func (s *NumericSearcher) GetType() string {
	return SearchTypeNumeric
}
