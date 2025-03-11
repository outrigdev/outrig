package logsearch

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
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

// SearchManager handles search functionality for a specific widget
type SearchManager struct {
	WidgetId string
	AppRunId string
	AppPeer  *apppeer.AppRunPeer
	LastUsed time.Time // Timestamp of when this manager was last used
}

// NewSearchManager creates a new SearchManager for a specific widget
func NewSearchManager(widgetId string, appPeer *apppeer.AppRunPeer) *SearchManager {
	return &SearchManager{
		WidgetId: widgetId,
		AppPeer:  appPeer,
		LastUsed: time.Now(),
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
		if now.Sub(manager.LastUsed) > MaxIdleTime {
			widgetManagers.Delete(key)
		} else {
			managers = append(managers, manager)
		}
	}
	if len(managers) > MaxSearchManagers {
		sort.Slice(managers, func(i, j int) bool {
			return managers[i].LastUsed.Before(managers[j].LastUsed)
		})
		for _, manager := range managers[:len(managers)-MaxSearchManagers] {
			widgetManagers.Delete(manager.WidgetId)
		}
	}
}

// GetManager gets or creates a SearchManager for the given widget ID and app peer
func GetManager(widgetId string, appRunId string) *SearchManager {
	// Get the app peer
	appPeer := apppeer.GetAppRunPeer(appRunId)

	manager, created := widgetManagers.GetOrCreate(widgetId, func() *SearchManager {
		return NewSearchManager(widgetId, appPeer)
	})

	// Update the AppRunId and AppPeer in case they've changed
	manager.AppRunId = appRunId
	manager.AppPeer = appPeer

	// Update the LastUsed timestamp
	manager.LastUsed = time.Now()

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
func UpdateLastUsed(widgetId string) {
	manager := widgetManagers.Get(widgetId)
	if manager != nil {
		manager.LastUsed = time.Now()
	}
}

// SearchRequest handles a search request for logs
func (m *SearchManager) SearchRequest(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	// Update the LastUsed timestamp when a search request is made
	m.LastUsed = time.Now()

	// Check if the AppPeer is valid
	if m.AppPeer == nil {
		return rpctypes.SearchResultData{
			FilteredCount: 0,
			TotalCount:    0,
			Lines:         []ds.LogLine{},
		}, fmt.Errorf("app peer not found for app run ID: %s", data.AppRunId)
	}

	// Get all log lines from the AppPeer
	allLogs := m.AppPeer.Logs.GetAll()
	totalCount := len(allLogs)

	// Since we're not filtering by search term yet, filteredCount equals totalCount
	filteredCount := totalCount

	// Apply view offset and limit with bounds checking
	startIndex := utilfn.BoundValue(data.ViewOffset, 0, totalCount)
	endIndex := utilfn.BoundValue(startIndex+data.ViewLimit, startIndex, totalCount)

	// Get the subset of logs based on offset and limit
	var resultLogs []ds.LogLine
	if startIndex < endIndex {
		resultLogs = allLogs[startIndex:endIndex]
	} else {
		resultLogs = []ds.LogLine{}
	}

	return rpctypes.SearchResultData{
		FilteredCount: filteredCount,
		TotalCount:    totalCount,
		Lines:         resultLogs,
	}, nil
}
