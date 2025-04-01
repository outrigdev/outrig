// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"strings"
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
)

const LogLineBufferSize = 10000

// LogLinePeer manages log lines for an AppRunPeer
type LogLinePeer struct {
	logLines      *utilds.CirBuf[ds.LogLine]
	lineNum       int64                    // Counter for log line numbers
	logLineLock   sync.Mutex               // Lock for synchronizing log line operations
	searchMgr     []SearchManagerInterface // Registered search managers
	logSearchLock sync.RWMutex             // Lock for search managers
}

// MakeLogLinePeer creates a new LogLinePeer instance
func MakeLogLinePeer() *LogLinePeer {
	return &LogLinePeer{
		logLines: utilds.MakeCirBuf[ds.LogLine](LogLineBufferSize),
		lineNum:  0,
	}
}

// addLogLine adds a log line to the buffer with proper synchronization
func (lp *LogLinePeer) addLogLine(line *ds.LogLine) {
	lp.logLineLock.Lock()
	defer lp.logLineLock.Unlock()
	
	// Increment line number and assign it to the log line
	lp.lineNum++
	line.LineNum = lp.lineNum
	
	// Add log line to the circular buffer
	lp.logLines.Write(*line)
}

// ProcessLogLine processes a log line
func (lp *LogLinePeer) ProcessLogLine(line ds.LogLine) {
	// Normalize line endings in the message
	line.Msg = normalizeLineEndings(line.Msg)

	// Add the log line to the buffer and update its line number
	lp.addLogLine(&line)

	// Notify search managers with the updated line
	lp.NotifySearchManagers(line)
}

// normalizeLineEndings ensures consistent line endings in log messages
func normalizeLineEndings(msg string) string {
	// remove all \r characters (converts windows-style line endings to unix-style)
	// internal \r characters are also likely problematic
	msg = strings.ReplaceAll(msg, "\r", "")

	// Ensure the message has at least one newline at the end
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}

	// Replace multiple consecutive newlines at the end with a single newline
	for strings.HasSuffix(msg, "\n\n") {
		msg = msg[:len(msg)-1]
	}

	return msg
}

// GetLogLines returns all log lines and the total count
func (lp *LogLinePeer) GetLogLines() ([]ds.LogLine, int) {
	// Get all log lines from the circular buffer and the head offset
	// CirBuf.GetAll() is already thread-safe, so we don't need additional locking
	lines, headOffset := lp.logLines.GetAll()

	// Return the lines and the total count (current lines + those that were trimmed)
	return lines, len(lines) + headOffset
}

// RegisterSearchManager registers a search manager with this LogLinePeer
func (lp *LogLinePeer) RegisterSearchManager(manager SearchManagerInterface) {
	lp.logSearchLock.Lock()
	defer lp.logSearchLock.Unlock()

	// Add the search manager to the list
	lp.searchMgr = append(lp.searchMgr, manager)
}

// UnregisterSearchManager removes a search manager from this LogLinePeer
func (lp *LogLinePeer) UnregisterSearchManager(manager SearchManagerInterface) {
	lp.logSearchLock.Lock()
	defer lp.logSearchLock.Unlock()

	// Find and remove the search manager
	for i, m := range lp.searchMgr {
		if m == manager {
			// Remove by swapping with the last element and truncating
			lp.searchMgr[i] = lp.searchMgr[len(lp.searchMgr)-1]
			lp.searchMgr = lp.searchMgr[:len(lp.searchMgr)-1]
			break
		}
	}
}

// NotifySearchManagers notifies all registered search managers about a new log line
func (lp *LogLinePeer) NotifySearchManagers(line ds.LogLine) {
	lp.logSearchLock.RLock()
	defer lp.logSearchLock.RUnlock()

	// Notify all registered search managers
	for _, manager := range lp.searchMgr {
		manager.ProcessNewLine(line)
	}
}
