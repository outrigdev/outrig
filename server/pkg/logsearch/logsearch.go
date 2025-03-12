package logsearch

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
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
	Lock       *sync.Mutex
	WidgetId   string
	AppRunId   string
	AppPeer    *apppeer.AppRunPeer
	LastUsed   time.Time // Timestamp of when this manager was last used
	SearchTerm string
	SearchType string
	Cache      *LogCache
}

// NewSearchManager creates a new SearchManager for a specific widget
func NewSearchManager(widgetId string, appPeer *apppeer.AppRunPeer) *SearchManager {
	return &SearchManager{
		Lock:       &sync.Mutex{},
		WidgetId:   widgetId,
		AppPeer:    appPeer,
		LastUsed:   time.Now(),
		SearchTerm: uuid.New().String(), // pick a random value that will never match a real search term
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

func (m *SearchManager) setUpNewLogCache_nolock(searchTerm string, searchType string) error {
	m.SearchTerm = searchTerm
	m.SearchType = searchType
	rawSource := MakeAppPeerLogSource(m.AppPeer)
	rawSource.InitSourceWithType(searchTerm, searchType, 0, DefaultBackendChunkSize)
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

// SearchRequest handles a search request for logs
func (m *SearchManager) SearchRequest(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	// Check if the AppPeer is valid
	if m.AppPeer == nil {
		return rpctypes.SearchResultData{}, fmt.Errorf("app peer not found for app run ID: %s", data.AppRunId)
	}
	m.LastUsed = time.Now()
	
	// If either the search term or search type has changed, create a new cache
	if data.SearchTerm != m.SearchTerm || data.SearchType != m.SearchType {
		err := m.setUpNewLogCache_nolock(data.SearchTerm, data.SearchType)
		if err != nil {
			return rpctypes.SearchResultData{}, err
		}
	}
	
	return rpctypes.SearchResultData{
		FilteredCount: m.Cache.GetFilteredSize(),
		TotalCount:    m.Cache.GetTotalSize(),
		Lines:         m.Cache.GetRange(data.RequestWindow.Start, data.RequestWindow.End()),
	}, nil
}
