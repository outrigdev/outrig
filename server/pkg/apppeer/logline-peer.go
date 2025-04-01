// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package apppeer

import (
	"strings"
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/server/pkg/gensearch"
)

const LogLineBufferSize = 10000

// LogLinePeer manages log lines for an AppRunPeer
type LogLinePeer struct {
	logLines      *utilds.CirBuf[ds.LogLine]
	lineNum       int64                            // Counter for log line numbers
	logLineLock   sync.Mutex                       // Lock for synchronizing log line operations
	searchMgr     []gensearch.SearchManagerInterface // Registered search managers
	logSearchLock sync.RWMutex                     // Lock for search managers
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
	
	lp.lineNum++
	line.LineNum = lp.lineNum
	
	lp.logLines.Write(*line)
}

// ProcessLogLine processes a log line
func (lp *LogLinePeer) ProcessLogLine(line ds.LogLine) {
	line.Msg = normalizeLineEndings(line.Msg)
	lp.addLogLine(&line)
	lp.NotifySearchManagers(line)
}

// normalizeLineEndings ensures consistent line endings in log messages
func normalizeLineEndings(msg string) string {
	msg = strings.ReplaceAll(msg, "\r", "")

	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	for strings.HasSuffix(msg, "\n\n") {
		msg = msg[:len(msg)-1]
	}

	return msg
}

// GetLogLines returns all log lines and the total count
func (lp *LogLinePeer) GetLogLines() ([]ds.LogLine, int) {
	lines, headOffset := lp.logLines.GetAll()
	return lines, len(lines) + headOffset
}

// RegisterSearchManager registers a search manager with this LogLinePeer
func (lp *LogLinePeer) RegisterSearchManager(manager gensearch.SearchManagerInterface) {
	lp.logSearchLock.Lock()
	defer lp.logSearchLock.Unlock()

	lp.searchMgr = append(lp.searchMgr, manager)
}

// UnregisterSearchManager removes a search manager from this LogLinePeer
func (lp *LogLinePeer) UnregisterSearchManager(manager gensearch.SearchManagerInterface) {
	lp.logSearchLock.Lock()
	defer lp.logSearchLock.Unlock()

	for i, m := range lp.searchMgr {
		if m == manager {
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

	for _, manager := range lp.searchMgr {
		manager.ProcessNewLine(line)
	}
}

// GetTotalCount returns the total count of log lines
func (lp *LogLinePeer) GetTotalCount() int {
	totalCount, _ := lp.logLines.GetTotalCountAndHeadOffset()
	return totalCount
}
