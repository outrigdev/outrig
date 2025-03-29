package gensearch

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

// SearchStats contains statistics about a search operation
type SearchStats struct {
	TotalCount     int   `json:"totalcount"`     // Total number of log lines in the AppRunPeer
	SearchedCount  int   `json:"searchedcount"`  // Number of log lines that were actually searched
	LastLineNum    int64 `json:"lastlinenum"`    // Last line number processed to avoid duplicates
	SearchDuration int   `json:"searchduration"` // Duration of the search operation in milliseconds
}

// SearchManagerInfo contains thread-safe information about a SearchManager
type SearchManagerInfo struct {
	WidgetId         string      `json:"widgetid"`
	AppRunId         string      `json:"apprunid"`
	LastUsedTime     time.Time   `json:"lastusedtime"`
	SearchTerm       string      `json:"searchterm"`
	FilteredLogCount int         `json:"filteredlogcount"`
	MarkedLinesCount int         `json:"markedlinescount"`
	RpcSource        string      `json:"rpcsource,omitempty"`
	TrimmedCount     int         `json:"trimmedcount,omitempty"`
	Stats            SearchStats `json:"stats"`
}

// SearchManager handles search functionality for a specific widget
type SearchManager struct {
	Lock     *sync.Mutex
	WidgetId string
	AppRunId string
	AppPeer  *apppeer.AppRunPeer // Will never be nil
	LastUsed time.Time           // Timestamp of when this manager was last used

	// User search components
	UserQuery    string   // The user's search term
	UserSearcher Searcher // Searcher for the user's search term

	// System search components
	SystemQuery    string   // System-generated query that may reference UserQuery
	SystemSearcher Searcher // Searcher for the system query

	CachedResult []ds.LogLine // Filtered log lines matching the search criteria
	Stats        SearchStats  // Statistics about the search operation
	TrimmedCount int          // Number of lines trimmed from the filtered logs

	MarkManager *MarkManager // Manager for marked lines
	RpcSource   string       // Source of the last RPC request that used this manager
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
		FilteredLogCount: len(m.CachedResult),
		MarkedLinesCount: m.MarkManager.GetNumMarks(),
		RpcSource:        m.RpcSource,
		TrimmedCount:     m.TrimmedCount,
		Stats:            m.Stats,
	}
}

// ProcessNewLine processes a new log line and adds it to FilteredLogs if it matches the search criteria
func (m *SearchManager) ProcessNewLine(line ds.LogLine) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Skip if we've already processed this line or an earlier one
	if line.LineNum <= m.Stats.LastLineNum {
		return
	}
	m.Stats.LastLineNum = line.LineNum

	// Update counts
	m.Stats.TotalCount++
	m.Stats.SearchedCount++

	// Determine which searcher to use - prefer system searcher if available
	var effectiveSearcher Searcher
	if m.SystemSearcher != nil {
		effectiveSearcher = m.SystemSearcher
	} else if m.UserSearcher != nil {
		effectiveSearcher = m.UserSearcher
	} else {
		return // No searcher available
	}

	// Create search context with marked lines and user query
	sctx := &SearchContext{
		MarkedLines: m.MarkManager.GetMarkedIds(),
		UserQuery:   m.UserSearcher, // Set the user query searcher for #userquery references
	}

	// Create a LogSearchObject from the log line
	searchObj := &LogSearchObject{
		Msg:     line.Msg,
		Source:  line.Source,
		LineNum: line.LineNum,
	}

	if !effectiveSearcher.Match(sctx, searchObj) {
		return
	}
	m.CachedResult = append(m.CachedResult, line)
	if len(m.CachedResult) > apppeer.LogLineBufferSize+TrimSize {
		m.TrimmedCount += TrimSize
		// create a new slice with the last N elements
		oldLogs := m.CachedResult
		m.CachedResult = make([]ds.LogLine, apppeer.LogLineBufferSize)
		copy(m.CachedResult, oldLogs[len(oldLogs)-apppeer.LogLineBufferSize:])
	}

	streamUpdate := rpctypes.StreamUpdateData{
		WidgetId:      m.WidgetId,
		FilteredCount: len(m.CachedResult),
		SearchedCount: m.Stats.SearchedCount,
		TotalCount:    m.Stats.TotalCount,
		TrimmedLines:  m.TrimmedCount,
		Offset:        len(m.CachedResult) - 1 + m.TrimmedCount,
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
		MarkManager: MakeMarkManager(),
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

// performSearch is a pure function that filters log lines based on search criteria
func performSearch(allLogs []ds.LogLine, totalCount int, searcher Searcher, sctx *SearchContext) ([]ds.LogLine, *SearchStats, error) {
	startTs := time.Now()

	searchedCount := len(allLogs)

	// Create a new result slice
	result := []ds.LogLine{}

	// Filter the logs based on the search criteria
	for _, line := range allLogs {
		if searcher == nil {
			result = append(result, line)
			continue
		}

		// Create a LogSearchObject from the log line
		searchObj := &LogSearchObject{
			Msg:     line.Msg,
			Source:  line.Source,
			LineNum: line.LineNum,
		}

		if searcher.Match(sctx, searchObj) {
			result = append(result, line)
		}
	}

	// Determine the last line number
	var lastLineNum int64 = 0
	if len(allLogs) > 0 {
		lastLineNum = allLogs[len(allLogs)-1].LineNum
	}

	// Calculate search duration
	searchDuration := int(time.Since(startTs).Milliseconds())

	// Create search stats
	stats := &SearchStats{
		TotalCount:     totalCount,
		SearchedCount:  searchedCount,
		LastLineNum:    lastLineNum,
		SearchDuration: searchDuration,
	}

	log.Printf("SearchManager: filtered %d/%d lines in %dms\n", len(result), searchedCount, searchDuration)
	return result, stats, nil
}

// GetMarkManager returns the MarkManager for the given widget ID
func GetMarkManager(widgetId string) *MarkManager {
	manager := GetManager(widgetId)
	if manager == nil {
		return nil
	}
	return manager.MarkManager
}

// SetRpcSource updates the RpcSource field with proper synchronization
func (m *SearchManager) SetRpcSource(ctx context.Context) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	m.RpcSource = rpc.GetRpcSourceFromContext(ctx)
}

// GetMarkedLogLines returns all marked log lines using a MarkedSearcher
func (m *SearchManager) GetMarkedLogLines() ([]ds.LogLine, error) {
	// If no lines are marked, return empty result
	if m.MarkManager.GetNumMarks() == 0 {
		return nil, nil
	}

	// Create a marked searcher
	searcher := MakeMarkedSearcher()

	// Create search context with marked lines
	sctx := &SearchContext{
		MarkedLines: m.MarkManager.GetMarkedIds(),
	}

	// Get all log lines from the circular buffer
	// We don't need to hold the lock during the search since performSearch is a pure function
	allLogs, headOffset := m.AppPeer.Logs.GetAll()
	totalCount := len(allLogs) + headOffset

	// Use the performSearch function to filter logs
	result, _, err := performSearch(allLogs, totalCount, searcher, sctx)
	return result, err
}

// maybeRunNewSearch checks if a new search is needed and performs it if necessary
// Returns an error if the search fails
func (m *SearchManager) maybeRunNewSearch(searchTerm, systemQuery string) error {
	// If the search term and system query haven't changed, no need to run a new search
	if searchTerm == m.UserQuery && systemQuery == m.SystemQuery {
		return nil
	}

	// Create searcher for the search term
	userSearcher, err := GetSearcher(searchTerm)
	if err != nil {
		return fmt.Errorf("failed to create user searcher: %w", err)
	}

	// Store the user searcher
	m.UserSearcher = userSearcher

	// Determine which searcher to use for the search
	var effectiveSearcher Searcher = userSearcher

	// If we have a system query, create a searcher for it
	if systemQuery != "" {
		systemSearcher, err := GetSearcher(systemQuery)
		if err != nil {
			return fmt.Errorf("failed to create system searcher: %w", err)
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
	m.UserQuery = searchTerm
	m.SystemQuery = systemQuery

	// Create search context with marked lines and user query
	sctx := &SearchContext{
		MarkedLines: m.MarkManager.GetMarkedIds(),
		UserQuery:   userSearcher, // Set the user query searcher for #userquery references
	}

	// Get all log lines from the circular buffer
	allLogs, headOffset := m.AppPeer.Logs.GetAll()
	totalCount := len(allLogs) + headOffset

	// Perform the search
	result, stats, err := performSearch(allLogs, totalCount, effectiveSearcher, sctx)
	if err != nil {
		m.UserQuery = uuid.New().String() // set to random value to prevent using cache
		m.SystemQuery = ""                // Clear the cached system query
		m.UserSearcher = nil              // Clear the cached searchers on error
		m.SystemSearcher = nil
		m.Stats = SearchStats{}
		return err
	}

	// Update the manager with the search results and stats
	m.CachedResult = result
	m.Stats = *stats

	return nil
}

// SearchLogs handles a search request for logs
func (m *SearchManager) SearchLogs(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Check if the AppPeer is valid
	if m.AppPeer == nil {
		return rpctypes.SearchResultData{}, fmt.Errorf("app peer not found for app run ID: %s", data.AppRunId)
	}
	m.LastUsed = time.Now()

	// Store the RPC source
	m.RpcSource = rpc.GetRpcSourceFromContext(ctx)

	// Run a new search if needed
	err := m.maybeRunNewSearch(data.SearchTerm, data.SystemQuery)
	if err != nil {
		return rpctypes.SearchResultData{}, err
	}

	// Get requested pages of log lines
	filteredSize := len(m.CachedResult)
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
			Lines:   m.CachedResult[startIndex:endIndex],
		})
	}

	return rpctypes.SearchResultData{
		FilteredCount: filteredSize,
		SearchedCount: m.Stats.SearchedCount,
		TotalCount:    m.Stats.TotalCount,
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
