// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"slices"
)

const (
	ReplacementModeInsert = "insert"
	ReplacementModeDelete = "delete"
)

type Replacement struct {
	Mode     string
	StartPos int64
	EndPos   int64
	NewText  []byte
}

func sortReplacementsByStartPos(rs []Replacement) []Replacement {
	result := make([]Replacement, len(rs))
	copy(result, rs)
	slices.SortStableFunc(result, func(a, b Replacement) int {
		if a.StartPos < b.StartPos {
			return -1
		} else if a.StartPos > b.StartPos {
			return 1
		}
		return 0
	})
	return result
}

func ApplyReplacements(fileBytes []byte, rs []Replacement) []byte {
	var chunks [][]byte
	pos := int64(0)
	fileLen := int64(len(fileBytes))

	rs = sortReplacementsByStartPos(rs)
	for _, r := range rs {
		if r.Mode == ReplacementModeInsert {
			// Skip insertions that are before our current position (in deleted regions)
			// or beyond the end of the file
			if r.StartPos < pos || r.StartPos > fileLen {
				continue
			}

			// Add chunk of original content before insertion point
			if r.StartPos > pos {
				chunks = append(chunks, fileBytes[pos:r.StartPos])
			}

			// Add the new text as a chunk
			if len(r.NewText) > 0 {
				chunks = append(chunks, r.NewText)
			}

			// Update position to continue after insertion point
			pos = r.StartPos

		} else if r.Mode == ReplacementModeDelete {
			// Add chunk of original content before deletion
			if r.StartPos > pos {
				chunks = append(chunks, fileBytes[pos:r.StartPos])
			}

			// Skip the deleted content, update position to after deletion
			pos = r.EndPos
		}
	}

	// Add remaining content after all replacements
	if pos < fileLen {
		chunks = append(chunks, fileBytes[pos:])
	}

	// Join all chunks together
	totalLen := 0
	for _, chunk := range chunks {
		totalLen += len(chunk)
	}

	result := make([]byte, 0, totalLen)
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}

	return result
}
