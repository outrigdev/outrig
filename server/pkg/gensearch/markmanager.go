// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
)

// MarkManager handles management of marked log lines
type MarkManager struct {
	Lock      *sync.Mutex
	MarkedIds map[int64]bool // Map of line numbers that are marked
}

// MakeMarkManager creates a new MarkManager
func MakeMarkManager() *MarkManager {
	return &MarkManager{
		Lock:      &sync.Mutex{},
		MarkedIds: make(map[int64]bool),
	}
}

// ClearMarks clears all marked lines
func (m *MarkManager) ClearMarks() {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	m.MarkedIds = make(map[int64]bool)
}

// GetNumMarks returns the number of marked lines
func (m *MarkManager) GetNumMarks() int {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	return len(m.MarkedIds)
}

// GetMarkedIds returns a copy of the marked lines map
func (m *MarkManager) GetMarkedIds() map[int64]bool {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	markedLinesCopy := make(map[int64]bool, len(m.MarkedIds))
	for lineNum, isMarked := range m.MarkedIds {
		markedLinesCopy[lineNum] = isMarked
	}

	return markedLinesCopy
}

// UpdateMarkedLines updates the marked status of lines based on the provided map
// If the value is true, the line is marked; if false, the mark is removed
func (m *MarkManager) UpdateMarkedLines(marks map[int64]bool) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	for lineNum, isMarked := range marks {
		if isMarked {
			m.MarkedIds[lineNum] = true
		} else {
			delete(m.MarkedIds, lineNum)
		}
	}
}

// GetMarkedLogLines returns all marked log lines from the provided logs
func (m *MarkManager) GetMarkedLogLines(allLogs []ds.LogLine) []ds.LogLine {
	// If no lines are marked, return empty result
	if m.GetNumMarks() == 0 {
		return nil
	}

	// Get a copy of the marked IDs
	markedIds := m.GetMarkedIds()

	// Filter the logs to only include marked lines
	var markedLines []ds.LogLine
	for _, line := range allLogs {
		if markedIds[line.LineNum] {
			markedLines = append(markedLines, line)
		}
	}

	return markedLines
}
