package logsearch

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
)

const (
	MaxSearchManagers = 5
	CleanupInterval   = 10 * time.Second
	MaxIdleTime       = 1 * time.Minute
)

// SearchManager handles search functionality for a specific widget
type SearchManager struct {
	Lock        *sync.Mutex
	WidgetId    string
	AppRunId    string
	AppPeer     *apppeer.AppRunPeer
	LastUsed    time.Time // Timestamp of when this manager was last used
	SearchTerm  string
	SearchType  string
	Cache       *LogCache
	MarkedLines map[int64]bool // Map of line numbers that are marked
}

// NewSearchManager creates a new SearchManager for a specific widget
func NewSearchManager(widgetId string, appPeer *apppeer.AppRunPeer) *SearchManager {
	return &SearchManager{
		Lock:        &sync.Mutex{},
		WidgetId:    widgetId,
		AppPeer:     appPeer,
		LastUsed:    time.Now(),
		SearchTerm:  uuid.New().String(), // pick a random value that will never match a real search term
		MarkedLines: make(map[int64]bool),
	}
}

// WidgetId => SearchManager
var widgetManagers = utilds.MakeSyncMap[*SearchManager]()

// init starts the background cleanup routine
func init() {
	go cleanupRoutine()
}

// cleanupRoutine periodically checks for and removes unused search managers
func cleanupRoutine() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cleanupSearchManagers()
	}
}

// cleanupSearchManagers removes search managers that haven't been used for MaxIdleTime
// and ensures we don't exceed MaxSearchManagers
func cleanupSearchManagers() {
	now := time.Now()
	keys := widgetManagers.Keys()
	managers := make([]*SearchManager, 0, len(keys))
	for _, key := range keys {
		manager := widgetManagers.Get(key)
		if now.Sub(manager.GetLastUsed()) > MaxIdleTime {
			widgetManagers.Delete(key)
		} else {
			managers = append(managers, manager)
		}
	}
	if len(managers) > MaxSearchManagers {
		sort.Slice(managers, func(i, j int) bool {
			return managers[i].GetLastUsed().Before(managers[j].GetLastUsed())
		})
		for _, manager := range managers[:len(managers)-MaxSearchManagers] {
			widgetManagers.Delete(manager.WidgetId)
		}
	}
}

func GetManager(widgetId string) *SearchManager {
	return widgetManagers.Get(widgetId)
}

// GetOrCreateManager gets or creates a SearchManager for the given widget ID and app peer
func GetOrCreateManager(widgetId string, appRunId string) *SearchManager {
	// Get the app peer
	appPeer := apppeer.GetAppRunPeer(appRunId)

	manager, created := widgetManagers.GetOrCreate(widgetId, func() *SearchManager {
		return NewSearchManager(widgetId, appPeer)
	})

	// Update the AppRunId and AppPeer in case they've changed
	manager.AppRunId = appRunId
	manager.AppPeer = appPeer

	// If we created a new manager or we're over the limit, run cleanup
	if created || widgetManagers.Len() > MaxSearchManagers {
		cleanupSearchManagers()
	}

	return manager
}

// DropManager removes a SearchManager for the given widget ID
func DropManager(widgetId string) {
	widgetManagers.Delete(widgetId)
}

// UpdateLastUsed updates the LastUsed timestamp for a SearchManager
func (m *SearchManager) UpdateLastUsed() {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.LastUsed = time.Now()
}

func (m *SearchManager) GetLastUsed() time.Time {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	return m.LastUsed
}

func (m *SearchManager) setUpNewLogCache_nolock(searchTerm string, searchType string, searcher LogSearcher) error {
	m.SearchTerm = searchTerm
	m.SearchType = searchType
	rawSource := MakeAppPeerLogSource(m.AppPeer)
	rawSource.InitSource(searcher, 0, DefaultBackendChunkSize)
	logCache, err := MakeLogCache(rawSource)
	if err != nil {
		m.SearchTerm = uuid.New().String() // set to random value to prevent using cache
		return fmt.Errorf("failed to create log cache: %w", err)
	}
	m.Cache = logCache
	doneCh := make(chan bool)
	m.Cache.RunSearch(func() {
		if m.Cache.IsDone() {
			close(doneCh)
		}
	})
	<-doneCh
	return nil
}

// MergeMarkedLines updates the marked status of lines based on the provided map
// If the value is true, the line is marked; if false, the mark is removed
func (m *SearchManager) MergeMarkedLines(marks map[int64]bool) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	for lineNum, isMarked := range marks {
		if isMarked {
			m.MarkedLines[lineNum] = true
		} else {
			delete(m.MarkedLines, lineNum)
		}
	}
	m.LastUsed = time.Now()
}

// IsLineMarked checks if a line is marked
func (m *SearchManager) IsLineMarked(lineNum int64) bool {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	_, exists := m.MarkedLines[lineNum]
	return exists
}

// ClearMarkedLines clears all marked lines
func (m *SearchManager) ClearMarkedLines() {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	m.MarkedLines = make(map[int64]bool)
	m.LastUsed = time.Now()
}

// GetNumMarkedLines returns the number of marked lines
func (m *SearchManager) GetNumMarkedLines() int {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	return len(m.MarkedLines)
}

// GetMarkedLines returns a slice of all marked line numbers
func (m *SearchManager) GetMarkedLines() []int64 {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	lines := make([]int64, 0, len(m.MarkedLines))
	for lineNum := range m.MarkedLines {
		lines = append(lines, lineNum)
	}

	// Sort the line numbers for consistent results
	sort.Slice(lines, func(i, j int) bool {
		return lines[i] < lines[j]
	})

	return lines
}

// GetMarkedLinesMap returns a copy of the marked lines map
func (m *SearchManager) GetMarkedLinesMap() map[int64]bool {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	markedLinesCopy := make(map[int64]bool, len(m.MarkedLines))
	for lineNum, isMarked := range m.MarkedLines {
		markedLinesCopy[lineNum] = isMarked
	}

	return markedLinesCopy
}

// RunFullSearch performs a full search using the provided searcher and returns all matching log lines
func (m *SearchManager) RunFullSearch(searcher LogSearcher) ([]ds.LogLine, error) {
	// Create a log source
	logSource := MakeAppPeerLogSource(m.AppPeer)

	// Initialize the source with the searcher
	logSource.InitSource(searcher, 0, DefaultBackendChunkSize)

	// Collect all matching log lines
	var matchingLines []ds.LogLine
	for {
		lines, eof, err := logSource.SearchNextChunk()
		if err != nil {
			return nil, fmt.Errorf("error searching logs: %w", err)
		}

		matchingLines = append(matchingLines, lines...)

		if eof {
			break
		}
	}

	return matchingLines, nil
}

// GetMarkedLogLines returns all marked log lines using a MarkedSearcher
func (m *SearchManager) GetMarkedLogLines() ([]ds.LogLine, error) {
	// If no lines are marked, return empty result
	if m.GetNumMarkedLines() == 0 {
		return []ds.LogLine{}, nil
	}

	// Create a marked searcher
	searcher := MakeMarkedSearcher(m)

	// Run the full search with the marked searcher
	markedLines, err := m.RunFullSearch(searcher)
	if err != nil {
		return nil, err
	}

	return markedLines, nil
}

// SearchRequest handles a search request for logs
func (m *SearchManager) SearchRequest(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	// Create searcher before acquiring the lock
	searcher, err := GetSearcher(data.SearchType, data.SearchTerm, m)
	if err != nil {
		return rpctypes.SearchResultData{}, fmt.Errorf("failed to create searcher: %w", err)
	}

	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Check if the AppPeer is valid
	if m.AppPeer == nil {
		return rpctypes.SearchResultData{}, fmt.Errorf("app peer not found for app run ID: %s", data.AppRunId)
	}
	m.LastUsed = time.Now()

	// If the search term has changed, create a new cache
	// Note: searchType is now optional and will be determined by the parser
	if data.SearchTerm != m.SearchTerm {
		err := m.setUpNewLogCache_nolock(data.SearchTerm, data.SearchType, searcher)
		if err != nil {
			return rpctypes.SearchResultData{}, err
		}
	}

	// Calculate total number of pages
	filteredSize := m.Cache.GetFilteredSize()
	totalPages := filteredSize / data.PageSize
	if filteredSize % data.PageSize != 0 {
		totalPages++
	}

	// Resolve page numbers (handle negative indices)
	resolvedPages := make(map[int]bool)
	resolvedPageNums := make([]int, 0, len(data.RequestPages))

	for _, pageNum := range data.RequestPages {
		resolvedPageNum := pageNum
		
		// Handle negative page numbers (counting from the end)
		if pageNum < 0 {
			resolvedPageNum = totalPages + pageNum // e.g., -1 becomes totalPages-1 (last page)
		}
		
		// Ensure page number is within valid range [0, totalPages-1]
		if resolvedPageNum >= 0 && resolvedPageNum < totalPages {
			// Only add each page once (avoid duplicates)
			if _, exists := resolvedPages[resolvedPageNum]; !exists {
				resolvedPages[resolvedPageNum] = true
				resolvedPageNums = append(resolvedPageNums, resolvedPageNum)
			}
		}
	}

	// Collect pages of log lines
	pages := make([]rpctypes.PageData, 0, len(resolvedPageNums))
	for _, pageNum := range resolvedPageNums {
		startIndex := pageNum * data.PageSize
		endIndex := startIndex + data.PageSize
		pageLines := m.Cache.GetRange(startIndex, endIndex)
		pages = append(pages, rpctypes.PageData{
			PageNum: pageNum,
			Lines:   pageLines,
		})
	}

	return rpctypes.SearchResultData{
		FilteredCount: filteredSize,
		TotalCount:    m.Cache.GetTotalSize(),
		Pages:         pages,
	}, nil
}
