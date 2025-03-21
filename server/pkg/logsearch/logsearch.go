package logsearch

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
)

const (
	MaxSearchManagers = 5
	CleanupInterval   = 10 * time.Second
	MaxIdleTime       = 1 * time.Minute
)

// SearchManagerInfo contains thread-safe information about a SearchManager
type SearchManagerInfo struct {
	WidgetId         string    `json:"widgetid"`
	AppRunId         string    `json:"apprunid"`
	LastUsedTime     time.Time `json:"lastusedtime"`
	SearchTerm       string    `json:"searchterm"`
	FilteredLogCount int       `json:"filteredlogcount"`
	MarkedLinesCount int       `json:"markedlinescount"`
	RpcSource        string    `json:"rpcsource,omitempty"`
}

// SearchManager handles search functionality for a specific widget
type SearchManager struct {
	Lock          *sync.Mutex
	WidgetId      string
	AppRunId      string
	AppPeer       *apppeer.AppRunPeer // Will never be nil
	LastUsed      time.Time           // Timestamp of when this manager was last used
	SearchTerm    string
	Searcher      LogSearcher    // Cached searcher for the current search term
	TotalCount    int            // Total number of log lines in the AppRunPeer
	SearchedCount int            // Number of log lines that were actually searched
	FilteredLogs  []ds.LogLine   // Filtered log lines matching the search criteria
	MarkedLines   map[int64]bool // Map of line numbers that are marked
	LastLineNum   int64          // Last line number processed to avoid duplicates
	RpcSource     string         // Source of the last RPC request that used this manager
}

// GetInfo returns a thread-safe copy of the SearchManager's information
func (m *SearchManager) GetInfo() SearchManagerInfo {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	
	return SearchManagerInfo{
		WidgetId:         m.WidgetId,
		AppRunId:         m.AppRunId,
		LastUsedTime:     m.LastUsed,
		SearchTerm:       m.SearchTerm,
		FilteredLogCount: len(m.FilteredLogs),
		MarkedLinesCount: len(m.MarkedLines),
		RpcSource:        m.RpcSource,
	}
}

// ProcessNewLine processes a new log line and adds it to FilteredLogs if it matches the search criteria
func (m *SearchManager) ProcessNewLine(line ds.LogLine) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Skip if we've already processed this line or an earlier one
	if line.LineNum <= m.LastLineNum {
		return
	}

	// Update the last line number processed
	m.LastLineNum = line.LineNum

	// Update counts
	m.TotalCount++
	m.SearchedCount++

	// If we have a searcher, check if the line matches
	if m.Searcher != nil {
		// Create search context with marked lines
		sctx := &SearchContext{
			MarkedLines: m.MarkedLines,
		}

		// If the line matches, add it to FilteredLogs
		if m.Searcher.Match(sctx, line) {
			m.FilteredLogs = append(m.FilteredLogs, line)
		}
	}
}

// NewSearchManager creates a new SearchManager for a specific widget
func NewSearchManager(widgetId string, appPeer *apppeer.AppRunPeer) *SearchManager {
	manager := &SearchManager{
		Lock:        &sync.Mutex{},
		WidgetId:    widgetId,
		AppPeer:     appPeer,
		LastUsed:    time.Now(),
		SearchTerm:  uuid.New().String(), // pick a random value that will never match a real search term
		MarkedLines: make(map[int64]bool),
	}

	// Register this manager with the AppRunPeer
	appPeer.RegisterSearchManager(manager)

	return manager
}

// WidgetId => SearchManager
var widgetManagers = utilds.MakeSyncMap[*SearchManager]()

// init starts the background cleanup routine and registers watches
func init() {
	go cleanupRoutine()
	
	// Register a watch function that returns a map of widget ID to SearchManagerInfo
	outrig.WatchFunc("searchmanagers", func() map[string]SearchManagerInfo {
		return GetAllSearchManagerInfos()
	}, nil)
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
			deleteSearchManager(manager)
		} else {
			managers = append(managers, manager)
		}
	}
	if len(managers) > MaxSearchManagers {
		sort.Slice(managers, func(i, j int) bool {
			return managers[i].GetLastUsed().Before(managers[j].GetLastUsed())
		})
		for _, manager := range managers[:len(managers)-MaxSearchManagers] {
			deleteSearchManager(manager)
		}
	}
}

func GetManager(widgetId string) *SearchManager {
	return widgetManagers.Get(widgetId)
}

// GetOrCreateManager gets or creates a SearchManager for the given widget ID and app peer
func GetOrCreateManager(widgetId string, appRunId string) *SearchManager {
	// Get the app peer
	appPeer := apppeer.GetAppRunPeer(appRunId, false)

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

// deleteSearchManager removes a SearchManager from the widgetManagers map and unregisters it from the AppRunPeer
func deleteSearchManager(manager *SearchManager) {
	// Unregister from AppRunPeer
	manager.AppPeer.UnregisterSearchManager(manager)

	// Delete from widgetManagers map
	widgetManagers.Delete(manager.WidgetId)
}

// DropManager removes a SearchManager for the given widget ID
func DropManager(widgetId string) {
	manager := widgetManagers.Get(widgetId)
	if manager != nil {
		deleteSearchManager(manager)
	}
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

func (m *SearchManager) performSearch_nolock(searchTerm string, searcher LogSearcher) error {
	startTs := time.Now()
	m.SearchTerm = searchTerm

	// Get total count of logs
	totalCount, _ := m.AppPeer.Logs.GetTotalCountAndHeadOffset()
	m.TotalCount = totalCount

	// Get all log lines from the circular buffer
	allLogs := m.AppPeer.Logs.GetAll()
	m.SearchedCount = len(allLogs)

	// Clear previous filtered logs
	m.FilteredLogs = []ds.LogLine{}

	// Create search context with marked lines
	// We can use the MarkedLines map directly since we're holding the lock
	sctx := &SearchContext{
		MarkedLines: m.MarkedLines,
	}

	// Filter the logs based on the search criteria
	for _, line := range allLogs {
		if searcher == nil || searcher.Match(sctx, line) {
			m.FilteredLogs = append(m.FilteredLogs, line)
		}
	}

	// Reset LastLineNum and set it to the highest line number (last line)
	// since logs are stored in line number order
	if len(allLogs) > 0 {
		m.LastLineNum = allLogs[len(allLogs)-1].LineNum
	} else {
		m.LastLineNum = 0
	}

	log.Printf("SearchManager: filtered %d/%d lines in %dms\n", len(m.FilteredLogs), m.SearchedCount, time.Since(startTs).Milliseconds())
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

// SetRpcSource updates the RpcSource field with proper synchronization
func (m *SearchManager) SetRpcSource(ctx context.Context) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	
	m.RpcSource = rpc.GetRpcSourceFromContext(ctx)
}

// RunFullSearch performs a full search using the provided searcher and returns all matching log lines
func (m *SearchManager) RunFullSearch(searcher LogSearcher) ([]ds.LogLine, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Get all log lines from the AppPeer
	allLogs := m.AppPeer.Logs.GetAll()

	// Create search context with marked lines
	// We can use the MarkedLines map directly since we're holding the lock
	sctx := &SearchContext{
		MarkedLines: m.MarkedLines,
	}

	// Filter the logs based on the search criteria
	var matchingLines []ds.LogLine
	for _, line := range allLogs {
		if searcher == nil || searcher.Match(sctx, line) {
			matchingLines = append(matchingLines, line)
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
	searcher := MakeMarkedSearcher()

	// Run the full search with the marked searcher
	markedLines, err := m.RunFullSearch(searcher)
	if err != nil {
		return nil, err
	}

	return markedLines, nil
}

// SearchRequest handles a search request for logs
func (m *SearchManager) SearchRequest(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Check if the AppPeer is valid
	if m.AppPeer == nil {
		return rpctypes.SearchResultData{}, fmt.Errorf("app peer not found for app run ID: %s", data.AppRunId)
	}
	m.LastUsed = time.Now()
	
	// Store the RPC source
	m.RpcSource = rpc.GetRpcSourceFromContext(ctx)

	// If the search term has changed, perform a new search
	if data.SearchTerm != m.SearchTerm {
		searcher, err := GetSearcher(data.SearchTerm)
		if err != nil {
			return rpctypes.SearchResultData{}, fmt.Errorf("failed to create searcher: %w", err)
		}
		// Cache the searcher
		m.Searcher = searcher
		err = m.performSearch_nolock(data.SearchTerm, searcher)
		if err != nil {
			m.SearchTerm = uuid.New().String() // set to random value to prevent using cache
			m.Searcher = nil                   // Clear the cached searcher on error
			return rpctypes.SearchResultData{}, err
		}
	}

	// Get requested pages of log lines
	filteredSize := len(m.FilteredLogs)
	totalPages := (filteredSize + data.PageSize - 1) / data.PageSize // Ceiling division

	// Process requested pages and collect results
	pages := make([]rpctypes.PageData, 0, len(data.RequestPages))
	seenPages := make(map[int]bool)

	for _, pageNum := range data.RequestPages {
		// Handle negative indices (counting from end)
		resolvedPage := pageNum
		if pageNum < 0 {
			resolvedPage = totalPages + pageNum
		}

		// Skip if out of range or already processed
		if resolvedPage < 0 || resolvedPage >= totalPages || seenPages[resolvedPage] {
			continue
		}
		seenPages[resolvedPage] = true

		// Calculate slice bounds
		startIndex := resolvedPage * data.PageSize
		endIndex := utilfn.BoundValue(startIndex+data.PageSize, startIndex, filteredSize)

		// Add page to results
		pages = append(pages, rpctypes.PageData{
			PageNum: resolvedPage,
			Lines:   m.FilteredLogs[startIndex:endIndex],
		})
	}

	return rpctypes.SearchResultData{
		FilteredCount: filteredSize,
		SearchedCount: m.SearchedCount,
		TotalCount:    m.TotalCount,
		Pages:         pages,
	}, nil
}

// GetAllSearchManagerInfos returns a map of widget ID to SearchManagerInfo for all search managers
func GetAllSearchManagerInfos() map[string]SearchManagerInfo {
	keys := widgetManagers.Keys()
	result := make(map[string]SearchManagerInfo, len(keys))
	
	for _, key := range keys {
		manager := widgetManagers.Get(key)
		if manager != nil {
			info := manager.GetInfo()
			result[key] = info
		}
	}
	
	return result
}
