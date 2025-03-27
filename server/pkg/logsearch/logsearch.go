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
	"github.com/outrigdev/outrig/pkg/rpcclient"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
)

const (
	MaxSearchManagers = 5
	CleanupInterval   = 10 * time.Second
	MaxIdleTime       = 1 * time.Minute
	TrimSize          = 1000
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
	TrimmedCount     int       `json:"trimmedcount,omitempty"`
}

// SearchManager handles search functionality for a specific widget
type SearchManager struct {
	Lock          *sync.Mutex
	WidgetId      string
	AppRunId      string
	AppPeer       *apppeer.AppRunPeer // Will never be nil
	LastUsed      time.Time           // Timestamp of when this manager was last used
	
	// User search components
	UserQuery    string              // The user's search term
	UserSearcher LogSearcher         // Searcher for the user's search term
	
	// System search components
	SystemQuery    string            // System-generated query that may reference UserQuery
	SystemSearcher LogSearcher       // Searcher for the system query
	
	TotalCount    int                // Total number of log lines in the AppRunPeer
	SearchedCount int                // Number of log lines that were actually searched
	FilteredLogs  []ds.LogLine       // Filtered log lines matching the search criteria
	MarkedLines   map[int64]bool     // Map of line numbers that are marked
	LastLineNum   int64              // Last line number processed to avoid duplicates
	RpcSource     string             // Source of the last RPC request that used this manager
	TrimmedCount  int                // Number of lines trimmed from the filtered logs
}

// GetInfo returns a thread-safe copy of the SearchManager's information
func (m *SearchManager) GetInfo() SearchManagerInfo {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	return SearchManagerInfo{
		WidgetId:         m.WidgetId,
		AppRunId:         m.AppRunId,
		LastUsedTime:     m.LastUsed,
		SearchTerm:       m.UserQuery,
		FilteredLogCount: len(m.FilteredLogs),
		MarkedLinesCount: len(m.MarkedLines),
		RpcSource:        m.RpcSource,
		TrimmedCount:     m.TrimmedCount,
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
	m.LastLineNum = line.LineNum

	// Update counts
	m.TotalCount++
	m.SearchedCount++

	// Determine which searcher to use - prefer system searcher if available
	var effectiveSearcher LogSearcher
	if m.SystemSearcher != nil {
		effectiveSearcher = m.SystemSearcher
	} else if m.UserSearcher != nil {
		effectiveSearcher = m.UserSearcher
	} else {
		return // No searcher available
	}

	// Create search context with marked lines and user query
	sctx := &SearchContext{
		MarkedLines: m.MarkedLines,
		UserQuery:   m.UserSearcher, // Set the user query searcher for #userquery references
	}

	if !effectiveSearcher.Match(sctx, line) {
		return
	}
	m.FilteredLogs = append(m.FilteredLogs, line)
	if len(m.FilteredLogs) > apppeer.LogLineBufferSize+TrimSize {
		m.TrimmedCount += TrimSize
		// create a new slice with the last N elements
		oldLogs := m.FilteredLogs
		m.FilteredLogs = make([]ds.LogLine, apppeer.LogLineBufferSize)
		copy(m.FilteredLogs, oldLogs[len(oldLogs)-apppeer.LogLineBufferSize:])
	}

	streamUpdate := rpctypes.StreamUpdateData{
		WidgetId:      m.WidgetId,
		FilteredCount: len(m.FilteredLogs),
		SearchedCount: m.SearchedCount,
		TotalCount:    m.TotalCount,
		TrimmedLines:  m.TrimmedCount,
		Offset:        len(m.FilteredLogs) - 1 + m.TrimmedCount,
		Lines:         []ds.LogLine{line},
	}
	go rpcclient.LogStreamUpdateCommand(rpcclient.BareClient, streamUpdate, &rpc.RpcOpts{Route: m.RpcSource, NoResponse: true})
}

// NewSearchManager creates a new SearchManager for a specific widget
func NewSearchManager(widgetId string, appPeer *apppeer.AppRunPeer) *SearchManager {
	manager := &SearchManager{
		Lock:        &sync.Mutex{},
		WidgetId:    widgetId,
		AppPeer:     appPeer,
		LastUsed:    time.Now(),
		UserQuery:   uuid.New().String(), // pick a random value that will never match a real search term
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

func (m *SearchManager) performSearch_nolock(searcher LogSearcher, sctx *SearchContext) error {
	startTs := time.Now()

	// Get all log lines from the circular buffer
	allLogs, headOffset := m.AppPeer.Logs.GetAll()
	m.TotalCount = len(allLogs) + headOffset

	m.SearchedCount = len(allLogs)

	// Clear previous filtered logs
	m.FilteredLogs = []ds.LogLine{}

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

// RunFullSearch performs a full search using the provided searcher and search context
// Returns all matching log lines
func (m *SearchManager) RunFullSearch(searcher LogSearcher, sctx *SearchContext) ([]ds.LogLine, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Get all log lines from the AppPeer
	allLogs, _ := m.AppPeer.Logs.GetAll()

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

	// Create search context with marked lines
	sctx := &SearchContext{
		MarkedLines: m.GetMarkedLinesMap(),
	}

	// Run the full search with the marked searcher and search context
	markedLines, err := m.RunFullSearch(searcher, sctx)
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

	// If the search term or system query has changed, perform a new search
	if data.SearchTerm != m.UserQuery || data.SystemQuery != m.SystemQuery {
		// Create searcher for the search term
		userSearcher, err := GetSearcher(data.SearchTerm)
		if err != nil {
			return rpctypes.SearchResultData{}, fmt.Errorf("failed to create user searcher: %w", err)
		}

		// Store the user searcher
		m.UserSearcher = userSearcher

		// Determine which searcher to use for the search
		var effectiveSearcher LogSearcher = userSearcher

		// If we have a system query, create a searcher for it
		if data.SystemQuery != "" {
			systemSearcher, err := GetSearcher(data.SystemQuery)
			if err != nil {
				return rpctypes.SearchResultData{}, fmt.Errorf("failed to create system searcher: %w", err)
			}

			// Store the system searcher
			m.SystemSearcher = systemSearcher
			
			// Use the system searcher as the effective searcher
			// The system searcher will use the UserQuery field in the SearchContext if it contains a #userquery token
			effectiveSearcher = systemSearcher
		} else {
			// No system query, clear the system searcher
			m.SystemSearcher = nil
		}

		// Update the query fields
		m.UserQuery = data.SearchTerm
		m.SystemQuery = data.SystemQuery
		
		// Create search context with marked lines and user query
		sctx := &SearchContext{
			MarkedLines: m.MarkedLines,
			UserQuery:   userSearcher, // Set the user query searcher for #userquery references
		}
		
		// Perform the search
		err = m.performSearch_nolock(effectiveSearcher, sctx)
		if err != nil {
			m.UserQuery = uuid.New().String() // set to random value to prevent using cache
			m.SystemQuery = ""                // Clear the cached system query
			m.UserSearcher = nil              // Clear the cached searchers on error
			m.SystemSearcher = nil
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
		MaxCount:      apppeer.LogLineBufferSize,
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
