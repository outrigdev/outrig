package logsearch

import (
	"log"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
)

// LogCache provides a simple cache for filtered log lines from an AppRunPeer
type LogCache struct {
	Lock          *sync.Mutex
	TotalCount    int          // Total number of log lines in the AppRunPeer
	SearchedCount int          // Number of log lines that were actually searched
	FilteredLogs  []ds.LogLine // Flat array of filtered log lines
	AppPeer       *apppeer.AppRunPeer
}

// MakeLogCache creates a new LogCache for the given AppRunPeer and immediately performs the search
func MakeLogCache(appPeer *apppeer.AppRunPeer, searcher LogSearcher) (*LogCache, error) {
	lc := &LogCache{
		Lock:         &sync.Mutex{},
		AppPeer:      appPeer,
		FilteredLogs: []ds.LogLine{},
	}

	// Immediately perform the search
	startTs := time.Now()
	lc.Lock.Lock()
	defer lc.Lock.Unlock()

	// Get total count of logs
	totalCount, _ := appPeer.Logs.GetTotalCountAndHeadOffset()
	lc.TotalCount = totalCount

	// Get all log lines from the circular buffer
	allLogs := appPeer.Logs.GetAll()
	lc.SearchedCount = len(allLogs)

	// Filter the logs based on the search criteria
	for _, line := range allLogs {
		if searcher == nil || searcher.Match(line) {
			lc.FilteredLogs = append(lc.FilteredLogs, line)
		}
	}
	log.Printf("LogCache: filtered %d lines in %dms\n", len(lc.FilteredLogs), time.Since(startTs).Milliseconds())

	return lc, nil
}

// GetRange returns a slice of filtered log lines within the specified range
func (lc *LogCache) GetRange(startIndex int, endIndex int) []ds.LogLine {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()

	// Ensure indices are within valid bounds
	startIndex = utilfn.BoundValue(startIndex, 0, len(lc.FilteredLogs))
	endIndex = utilfn.BoundValue(endIndex, startIndex, len(lc.FilteredLogs))

	// If after bounds checking we have nothing to return, exit early
	if startIndex == endIndex {
		return []ds.LogLine{}
	}

	// Return the requested slice of filtered logs
	return lc.FilteredLogs[startIndex:endIndex]
}

// GetFilteredSize returns the number of filtered log lines
func (lc *LogCache) GetFilteredSize() int {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	return len(lc.FilteredLogs)
}

// GetTotalSize returns the total number of log lines in the AppRunPeer
func (lc *LogCache) GetTotalSize() int {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	return lc.TotalCount
}

// GetSearchedSize returns the number of log lines that were actually searched
func (lc *LogCache) GetSearchedSize() int {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	return lc.SearchedCount
}
