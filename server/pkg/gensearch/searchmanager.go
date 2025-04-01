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
	LogPeer  *apppeer.LogLinePeer // Reference to the LogLinePeer for log operations
	LastUsed time.Time            // Timestamp of when this manager was last used

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

	// Convert the log line to a SearchObject
	searchObj := LogLineToSearchObject(line)

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
		LogPeer:     appPeer.Logs,
		LastUsed:    time.Now(),
		UserQuery:   uuid.New().String(), // pick a random value that will never match a real search term
		MarkManager: MakeMarkManager(),
	}

	// Register this manager with the LogLinePeer
	appPeer.Logs.RegisterSearchManager(manager)

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
	appPeer := apppeer.GetAppRunPeer(appRunId, false)
	manager, created := widgetManagers.GetOrCreate(widgetId, func() *SearchManager {
		return NewSearchManager(widgetId, appPeer)
	})
	// If we created a new manager or we're over the limit, run cleanup
	if created || widgetManagers.Len() > MaxSearchManagers {
		cleanupSearchManagers()
	}
	return manager
}

// deleteSearchManager removes a SearchManager from the widgetManagers map and unregisters it from the LogLinePeer
func deleteSearchManager(manager *SearchManager) {
	manager.LogPeer.UnregisterSearchManager(manager)
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

// PerformSearch is a generic function that filters items based on search criteria
// It takes a slice of items of any type T, a total count, a function to convert T to SearchObject,
// a searcher, and a search context, and returns filtered items and search stats
func PerformSearch[T any](allItems []T, totalCount int, toSearchObj func(T) SearchObject, searcher Searcher, sctx *SearchContext) ([]T, *SearchStats, error) {
	startTs := time.Now()
	searchedCount := len(allItems)
	result := []T{}

	// Filter the items based on the search criteria
	for _, item := range allItems {
		if searcher == nil {
			result = append(result, item)
			continue
		}

		// Convert the item to a SearchObject using the provided function
		searchObj := toSearchObj(item)

		if searcher.Match(sctx, searchObj) {
			result = append(result, item)
		}
	}

	// Determine the last line number (if available)
	var lastLineNum int64 = 0
	if len(allItems) > 0 {
		// Try to get the last item's ID, which might be a line number
		if searchObj := toSearchObj(allItems[len(allItems)-1]); searchObj != nil {
			lastLineNum = searchObj.GetId()
		}
	}

	searchDuration := int(time.Since(startTs).Milliseconds())
	stats := &SearchStats{
		TotalCount:     totalCount,
		SearchedCount:  searchedCount,
		LastLineNum:    lastLineNum,
		SearchDuration: searchDuration,
	}
	log.Printf("SearchManager: filtered %d/%d items in %dms\n", len(result), searchedCount, searchDuration)
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
	searcher := MakeMarkedSearcher()
	sctx := &SearchContext{
		MarkedLines: m.MarkManager.GetMarkedIds(),
	}
	// Get all log lines and total count from the LogLinePeer in a single synchronized call
	// We don't need to hold the lock during the search since PerformSearch is a pure function
	allLogs, totalCount := m.LogPeer.GetLogLines()
	result, _, err := PerformSearch(allLogs, totalCount, LogLineToSearchObject, searcher, sctx)
	return result, err
}

// maybeRunNewSearch checks if a new search is needed and performs it if necessary
// Returns an error if the search fails
func (m *SearchManager) maybeRunNewSearch(searchTerm, systemQuery string) error {
	// If the search term and system query haven't changed, no need to run a new search
	if searchTerm == m.UserQuery && systemQuery == m.SystemQuery {
		return nil
	}
	userSearcher, err := GetSearcher(searchTerm)
	if err != nil {
		return fmt.Errorf("failed to create user searcher: %w", err)
	}
	m.UserSearcher = userSearcher
	var effectiveSearcher Searcher = userSearcher
	if systemQuery != "" {
		systemSearcher, err := GetSearcher(systemQuery)
		if err != nil {
			return fmt.Errorf("failed to create system searcher: %w", err)
		}
		m.SystemSearcher = systemSearcher
		// Use the system searcher as the effective searcher
		// The system searcher will use the UserQuery field in the SearchContext if it contains a #userquery token
		effectiveSearcher = systemSearcher
	} else {
		m.SystemSearcher = nil
	}

	// Update the query fields
	m.UserQuery = searchTerm
	m.SystemQuery = systemQuery

	sctx := &SearchContext{
		MarkedLines: m.MarkManager.GetMarkedIds(),
		UserQuery:   userSearcher, // Set the user query searcher for #userquery references
	}
	allLogs, totalCount := m.LogPeer.GetLogLines()
	result, stats, err := PerformSearch(allLogs, totalCount, LogLineToSearchObject, effectiveSearcher, sctx)
	if err != nil {
		m.UserQuery = uuid.New().String() // set to random value to prevent using cache
		m.SystemQuery = ""                // Clear the cached system query
		m.UserSearcher = nil              // Clear the cached searchers on error
		m.SystemSearcher = nil
		m.Stats = SearchStats{}
		return err
	}

	m.CachedResult = result
	m.Stats = *stats
	return nil
}

// SearchLogs handles a search request for logs
func (m *SearchManager) SearchLogs(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	m.LastUsed = time.Now()
	m.RpcSource = rpc.GetRpcSourceFromContext(ctx)
	err := m.maybeRunNewSearch(data.SearchTerm, data.SystemQuery)
	if err != nil {
		return rpctypes.SearchResultData{}, err
	}

	filteredSize := len(m.CachedResult)
	totalPages := (filteredSize + data.PageSize - 1) / data.PageSize // Ceiling division
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
		startIndex := resolvedPage * data.PageSize
		endIndex := utilfn.BoundValue(startIndex+data.PageSize, startIndex, filteredSize)
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
