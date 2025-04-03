package rpcserver

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
	"github.com/outrigdev/outrig/server/pkg/browsertabs"
	"github.com/outrigdev/outrig/server/pkg/gensearch"
)

type RpcServerImpl struct{}

func (*RpcServerImpl) RpcServerImpl() {}

func (*RpcServerImpl) EventPublishCommand(ctx context.Context, data rpctypes.EventType) error {
	rpcSource := rpc.GetRpcSourceFromContext(ctx)
	if rpcSource == "" {
		return fmt.Errorf("no rpc source set")
	}
	if data.Sender == "" {
		data.Sender = rpcSource
	}
	rpc.Broker.Publish(data)
	return nil
}

func (*RpcServerImpl) EventSubCommand(ctx context.Context, data rpctypes.SubscriptionRequest) error {
	rpcSource := rpc.GetRpcSourceFromContext(ctx)
	if rpcSource == "" {
		return fmt.Errorf("no rpc source set")
	}
	rpc.Broker.Subscribe(rpcSource, data)
	return nil
}

func (*RpcServerImpl) EventUnsubCommand(ctx context.Context, data string) error {
	rpcSource := rpc.GetRpcSourceFromContext(ctx)
	if rpcSource == "" {
		return fmt.Errorf("no rpc source set")
	}
	rpc.Broker.Unsubscribe(rpcSource, data)
	return nil
}

func (*RpcServerImpl) EventUnsubAllCommand(ctx context.Context) error {
	rpcSource := rpc.GetRpcSourceFromContext(ctx)
	if rpcSource == "" {
		return fmt.Errorf("no rpc source set")
	}
	rpc.Broker.UnsubscribeAll(rpcSource)
	return nil
}

func (*RpcServerImpl) EventReadHistoryCommand(ctx context.Context, data rpctypes.EventReadHistoryData) ([]*rpctypes.EventType, error) {
	events := rpc.Broker.ReadEventHistory(data.Event, data.Scope, data.MaxItems)
	return events, nil
}

// GetAppRunsCommand returns a list of app runs
// If since > 0, only returns app runs that have been updated since the given timestamp
func (*RpcServerImpl) GetAppRunsCommand(ctx context.Context, data rpctypes.AppRunUpdatesRequest) (rpctypes.AppRunsData, error) {
	// Get app run infos directly from the apppeer package
	appRuns := apppeer.GetAllAppRunPeerInfos(data.Since)

	return rpctypes.AppRunsData{
		AppRuns: appRuns,
	}, nil
}

// GetAppRunGoRoutinesCommand returns goroutines for a specific app run
func (*RpcServerImpl) GetAppRunGoRoutinesCommand(ctx context.Context, data rpctypes.AppRunRequest) (rpctypes.AppRunGoRoutinesData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunGoRoutinesData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get module name from AppInfo
	moduleName := ""
	if peer.AppInfo != nil {
		moduleName = peer.AppInfo.ModuleName
	}

	// Get parsed goroutines from the GoRoutinePeer
	parsedGoRoutines := peer.GoRoutines.GetParsedGoRoutines(moduleName)

	return rpctypes.AppRunGoRoutinesData{
		AppRunId:   peer.AppRunId,
		AppName:    peer.AppInfo.AppName,
		GoRoutines: parsedGoRoutines,
	}, nil
}

// GetAppRunGoRoutinesByIdsCommand returns specific goroutines by their IDs for a specific app run
func (*RpcServerImpl) GetAppRunGoRoutinesByIdsCommand(ctx context.Context, data rpctypes.AppRunGoRoutinesByIdsRequest) (rpctypes.AppRunGoRoutinesData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunGoRoutinesData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get module name from AppInfo
	moduleName := ""
	if peer.AppInfo != nil {
		moduleName = peer.AppInfo.ModuleName
	}

	// Get parsed goroutines by IDs from the GoRoutinePeer
	parsedGoRoutines := peer.GoRoutines.GetParsedGoRoutinesByIds(moduleName, data.GoIds)

	return rpctypes.AppRunGoRoutinesData{
		AppRunId:   peer.AppRunId,
		AppName:    peer.AppInfo.AppName,
		GoRoutines: parsedGoRoutines,
	}, nil
}

// GetAppRunWatchesCommand returns watches for a specific app run
func (*RpcServerImpl) GetAppRunWatchesCommand(ctx context.Context, data rpctypes.AppRunRequest) (rpctypes.AppRunWatchesData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunWatchesData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get all watches using the WatchesPeer method
	watches := peer.Watches.GetAllWatches()

	// Create and return AppRunWatchesData
	return rpctypes.AppRunWatchesData{
		AppRunId: peer.AppRunId,
		AppName:  peer.AppInfo.AppName,
		Watches:  watches,
	}, nil
}

// GetAppRunRuntimeStatsCommand returns runtime stats for a specific app run
func (*RpcServerImpl) GetAppRunRuntimeStatsCommand(ctx context.Context, data rpctypes.AppRunRequest) (rpctypes.AppRunRuntimeStatsData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunRuntimeStatsData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Initialize empty result
	result := rpctypes.AppRunRuntimeStatsData{
		AppRunId: peer.AppRunId,
		AppName:  peer.AppInfo.AppName,
		Stats:    peer.RuntimeStats.GetRuntimeStats(data.Since),
	}

	return result, nil
}

// GoRoutineSearchRequestCommand handles search requests for goroutines
func (*RpcServerImpl) GoRoutineSearchRequestCommand(ctx context.Context, data rpctypes.GoRoutineSearchRequestData) (rpctypes.GoRoutineSearchResultData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.GoRoutineSearchResultData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get module name from AppInfo (needed for proper goroutine parsing)
	moduleName := ""
	if peer.AppInfo != nil {
		moduleName = peer.AppInfo.ModuleName
	}

	// Get all goroutines
	allGoRoutines := peer.GoRoutines.GetParsedGoRoutines(moduleName)
	totalCount := len(allGoRoutines)

	// Create user searcher based on search term and get any error spans
	userSearcher, errorSpans, err := gensearch.GetSearcherWithErrors(data.SearchTerm)
	if err != nil {
		return rpctypes.GoRoutineSearchResultData{}, fmt.Errorf("invalid search term: %w", err)
	}

	// Determine which searcher to use (user or system)
	var effectiveSearcher gensearch.Searcher = userSearcher
	
	// If system query is provided, use it as the effective searcher
	if data.SystemQuery != "" {
		// System queries are generated by the system, so they shouldn't have syntax errors
		systemSearcher, err := gensearch.GetSearcher(data.SystemQuery)
		if err != nil {
			return rpctypes.GoRoutineSearchResultData{}, fmt.Errorf("invalid system query: %w", err)
		}
		effectiveSearcher = systemSearcher
	}

	// Create search context with user searcher for #userquery references
	sctx := &gensearch.SearchContext{
		UserQuery: userSearcher,
	}

	// Perform the search
	filteredGoRoutines, stats, err := gensearch.PerformSearch(
		allGoRoutines, 
		totalCount, 
		gensearch.ParsedGoRoutineToSearchObject, 
		effectiveSearcher, 
		sctx,
	)
	if err != nil {
		return rpctypes.GoRoutineSearchResultData{}, err
	}

	// Extract GoIds from filtered results
	results := make([]int64, 0, len(filteredGoRoutines))
	for _, gr := range filteredGoRoutines {
		results = append(results, gr.GoId)
	}
	
	// Sort the results by goroutine ID for consistent ordering
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})

	return rpctypes.GoRoutineSearchResultData{
		SearchedCount: stats.SearchedCount,
		TotalCount:    stats.TotalCount,
		Results:       results,
		ErrorSpans:    errorSpans,
	}, nil
}

// LogSearchRequestCommand handles search requests for logs
func (*RpcServerImpl) LogSearchRequestCommand(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	manager := gensearch.GetOrCreateManager(data.WidgetId, data.AppRunId, peer.Logs)
	return manager.SearchLogs(ctx, data)
}

// LogWidgetAdminCommand handles widget administration requests
func (*RpcServerImpl) LogWidgetAdminCommand(ctx context.Context, data rpctypes.LogWidgetAdminData) error {
	manager := gensearch.GetManager(data.WidgetId)
	if manager == nil {
		return nil
	}

	// Store the RPC source with proper synchronization
	manager.SetRpcSource(ctx)

	// Drop takes precedence over KeepAlive
	if data.Drop {
		gensearch.DropManager(data.WidgetId)
	} else if data.KeepAlive {
		manager.UpdateLastUsed()
	}
	return nil
}

// LogUpdateMarkedLinesCommand handles updating marked lines for a widget
func (*RpcServerImpl) LogUpdateMarkedLinesCommand(ctx context.Context, data rpctypes.MarkedLinesData) error {
	markManager := gensearch.GetMarkManager(data.WidgetId)
	if markManager == nil {
		return fmt.Errorf("widget not found: %s", data.WidgetId)
	}

	// If Clear flag is set, clear all marked lines
	if data.Clear {
		markManager.ClearMarks()
		return nil
	}

	// Convert string keys to int64 keys
	markedLines := make(map[int64]bool)
	for lineNumStr, isMarked := range data.MarkedLines {
		lineNum, err := strconv.ParseInt(lineNumStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid line number: %s", lineNumStr)
		}
		markedLines[lineNum] = isMarked
	}

	// Update the marked lines
	markManager.UpdateMarkedLines(markedLines)
	return nil
}

// LogGetMarkedLinesCommand retrieves all marked log lines for a widget
func (*RpcServerImpl) LogGetMarkedLinesCommand(ctx context.Context, data rpctypes.MarkedLinesRequestData) (rpctypes.MarkedLinesResultData, error) {
	manager := gensearch.GetManager(data.WidgetId)
	if manager == nil {
		return rpctypes.MarkedLinesResultData{}, fmt.Errorf("widget not found: %s", data.WidgetId)
	}

	// Get marked log lines from the search manager
	markedLines, err := manager.GetMarkedLogLines()
	if err != nil {
		return rpctypes.MarkedLinesResultData{}, err
	}

	return rpctypes.MarkedLinesResultData{Lines: markedLines}, nil
}

// UpdateBrowserTabUrlCommand updates the URL for a browser tab
func (*RpcServerImpl) UpdateBrowserTabUrlCommand(ctx context.Context, data rpctypes.BrowserTabUrlData) error {
	rpcSource := rpc.GetRpcSourceFromContext(ctx)
	if rpcSource == "" {
		return fmt.Errorf("no rpc source set")
	}

	return browsertabs.UpdateBrowserTabUrl(rpcSource, data)
}
