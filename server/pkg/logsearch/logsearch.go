package logsearch

import (
	"context"
	"sort"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
)

const (
	// MaxSearchManagers is the maximum number of search managers to keep
	MaxSearchManagers = 5

	// CleanupInterval is how often to run the cleanup routine
	CleanupInterval = 10 * time.Second

	// MaxIdleTime is how long a search manager can be idle before being removed
	MaxIdleTime = 1 * time.Minute
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

// Global map of widget IDs to search managers
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
	
	// Collect managers that aren't too old
	managers := make([]*SearchManager, 0, len(keys))
	
	// First pass: remove idle managers and collect the rest
	for _, key := range keys {
		manager := widgetManagers.Get(key)
		
		// Remove if it hasn't been used for MaxIdleTime
		if now.Sub(manager.LastUsed) > MaxIdleTime {
			widgetManagers.Delete(key)
		} else {
			// Not too old, add to our collection
			managers = append(managers, manager)
		}
	}
	
	// If we still have more than MaxSearchManagers, remove the oldest ones
	if len(managers) > MaxSearchManagers {
		// Sort by last used time (oldest first)
		sort.Slice(managers, func(i, j int) bool {
			return managers[i].LastUsed.Before(managers[j].LastUsed)
		})
		
		// Remove oldest managers until we're at MaxSearchManagers
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

// SearchRequest handles a search request for logs
func (m *SearchManager) SearchRequest(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	// Update the LastUsed timestamp when a search request is made
	m.LastUsed = time.Now()

	// TODO: Implement actual search functionality
	// For now, return an empty result
	return rpctypes.SearchResultData{
		FilteredCount: 0,
		TotalCount:    0,
		Lines:         []ds.LogLine{},
	}, nil
}
