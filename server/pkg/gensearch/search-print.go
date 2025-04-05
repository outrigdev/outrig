// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"fmt"
	"strings"
)

// PrettyPrint returns a human-readable string representation of a searcher
func PrettyPrint(s Searcher) string {
	if s == nil {
		return "<nil>"
	}

	switch searcher := s.(type) {
	case *ExactSearcher:
		sensitivity := "case-insensitive"
		if searcher.caseSensitive {
			sensitivity = "case-sensitive"
		}
		return fmt.Sprintf("ExactSearcher{field: %q, term: %q, %s}",
			searcher.field, searcher.searchTerm, sensitivity)

	case *RegexpSearcher:
		sensitivity := "case-insensitive"
		if searcher.caseSensitive {
			sensitivity = "case-sensitive"
		}
		return fmt.Sprintf("RegexpSearcher{field: %q, pattern: %q, %s}",
			searcher.field, searcher.searchTerm, sensitivity)

	case *FzfSearcher:
		sensitivity := "case-insensitive"
		if searcher.caseSensitive {
			sensitivity = "case-sensitive"
		}
		return fmt.Sprintf("FzfSearcher{field: %q, term: %q, %s}",
			searcher.field, searcher.searchTerm, sensitivity)

	case *TagSearcher:
		return fmt.Sprintf("TagSearcher{tag: %q, exactMatch: %v}",
			searcher.searchTerm, searcher.exactMatch)

	case *AndSearcher:
		children := make([]string, 0, len(searcher.searchers))
		for _, child := range searcher.searchers {
			children = append(children, PrettyPrint(child))
		}
		return fmt.Sprintf("AndSearcher{%s}", strings.Join(children, " AND "))

	case *OrSearcher:
		children := make([]string, 0, len(searcher.searchers))
		for _, child := range searcher.searchers {
			children = append(children, PrettyPrint(child))
		}
		return fmt.Sprintf("OrSearcher{%s}", strings.Join(children, " OR "))

	case *NotSearcher:
		return fmt.Sprintf("NotSearcher{%s}", PrettyPrint(searcher.searcher))

	case *AllSearcher:
		return "AllSearcher{}"

	case *MarkedSearcher:
		return "MarkedSearcher{}"

	case *UserQuerySearcher:
		return "UserQuerySearcher{}"

	default:
		return fmt.Sprintf("UnknownSearcher{type: %s}", s.GetType())
	}
}

// PrettyPrintMultiline returns a human-readable multi-line string representation of a searcher
func PrettyPrintMultiline(s Searcher) string {
	return prettyPrintWithIndent(s, 0)
}

func prettyPrintWithIndent(s Searcher, indent int) string {
	if s == nil {
		return "<nil>"
	}

	indentStr := strings.Repeat("  ", indent)
	nextIndent := indent + 1

	switch searcher := s.(type) {
	case *AndSearcher:
		if len(searcher.searchers) == 0 {
			return indentStr + "AndSearcher{}"
		}

		var sb strings.Builder
		sb.WriteString(indentStr + "AndSearcher{\n")

		for i, child := range searcher.searchers {
			sb.WriteString(prettyPrintWithIndent(child, nextIndent))
			if i < len(searcher.searchers)-1 {
				sb.WriteString(",\n")
			} else {
				sb.WriteString("\n")
			}
		}

		sb.WriteString(indentStr + "}")
		return sb.String()

	case *OrSearcher:
		if len(searcher.searchers) == 0 {
			return indentStr + "OrSearcher{}"
		}

		var sb strings.Builder
		sb.WriteString(indentStr + "OrSearcher{\n")

		for i, child := range searcher.searchers {
			sb.WriteString(prettyPrintWithIndent(child, nextIndent))
			if i < len(searcher.searchers)-1 {
				sb.WriteString(",\n")
			} else {
				sb.WriteString("\n")
			}
		}

		sb.WriteString(indentStr + "}")
		return sb.String()

	case *NotSearcher:
		var sb strings.Builder
		sb.WriteString(indentStr + "NotSearcher{\n")
		sb.WriteString(prettyPrintWithIndent(searcher.searcher, nextIndent))
		sb.WriteString("\n" + indentStr + "}")
		return sb.String()

	default:
		// For simple searchers, just use the inline format with indentation
		return indentStr + PrettyPrint(s)
	}
}
