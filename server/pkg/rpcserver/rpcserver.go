// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package rpcserver

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/outrigdev/outrig/server/pkg/apppeer"
	"github.com/outrigdev/outrig/server/pkg/browsertabs"
	"github.com/outrigdev/outrig/server/pkg/democontroller"
	"github.com/outrigdev/outrig/server/pkg/gensearch"
	"github.com/outrigdev/outrig/server/pkg/rpc"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/tevent"
	"github.com/outrigdev/outrig/server/pkg/updatecheck"
)

const (
	MaxGoRoutineSearchResults = 1000 // Maximum number of goroutines to return from a search
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

// GetAppRunWatchesByIdsCommand returns specific watches by their IDs for a specific app run
func (*RpcServerImpl) GetAppRunWatchesByIdsCommand(ctx context.Context, data rpctypes.AppRunWatchesByIdsRequest) (rpctypes.AppRunWatchesData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunWatchesData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get watches by IDs from the WatchesPeer
	watches := peer.Watches.GetWatchesByIds(data.WatchIds)

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

	// Get goroutine counts
	numGoRoutines, numActiveGoRoutines, numOutrigGoRoutines := peer.GoRoutines.GetGoRoutineCounts()

	// Initialize result with goroutine counts
	result := rpctypes.AppRunRuntimeStatsData{
		AppRunId:            peer.AppRunId,
		AppName:             peer.AppInfo.AppName,
		NumTotalGoRoutines:  numGoRoutines,
		NumActiveGoRoutines: numActiveGoRoutines,
		NumOutrigGoRoutines: numOutrigGoRoutines,
		Stats:               peer.RuntimeStats.GetRuntimeStats(data.Since),
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

	// Limit the number of results to MaxGoRoutineSearchResults
	if len(results) > MaxGoRoutineSearchResults {
		results = results[:MaxGoRoutineSearchResults]
	}

	// Calculate total non-outrig goroutines count
	totalNonOutrig := 0
	for _, gr := range allGoRoutines {
		if !slices.Contains(gr.Tags, "outrig") {
			totalNonOutrig++
		}
	}

	return rpctypes.GoRoutineSearchResultData{
		SearchedCount:  stats.SearchedCount,
		TotalCount:     stats.TotalCount,
		TotalNonOutrig: totalNonOutrig,
		Results:        results,
		ErrorSpans:     errorSpans,
	}, nil
}

// combinedWatchSampleToSearchObject converts a CombinedWatchSample to a WatchSearchObject
func combinedWatchSampleToSearchObject(combined rpctypes.CombinedWatchSample) gensearch.SearchObject {
	// Extract data from both the declaration and sample
	decl := combined.Decl
	sample := combined.Sample

	// Use the WatchNum from the CombinedWatchSample
	watchNum := combined.WatchNum

	// Get the value from the sample
	val := sample.Val

	return &gensearch.WatchSearchObject{
		WatchNum: watchNum,
		Name:     sample.Name,
		Val:      val,
		Tags:     decl.Tags,
		Type:     sample.Type,
	}
}

// WatchSearchRequestCommand handles search requests for watches
func (*RpcServerImpl) WatchSearchRequestCommand(ctx context.Context, data rpctypes.WatchSearchRequestData) (rpctypes.WatchSearchResultData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId, false)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.WatchSearchResultData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get all watches
	allWatches := peer.Watches.GetAllWatches()
	totalCount := len(allWatches)

	// Create user searcher based on search term and get any error spans
	userSearcher, errorSpans, err := gensearch.GetSearcherWithErrors(data.SearchTerm)
	if err != nil {
		return rpctypes.WatchSearchResultData{}, fmt.Errorf("invalid search term: %w", err)
	}

	// Determine which searcher to use (user or system)
	var effectiveSearcher gensearch.Searcher = userSearcher

	// If system query is provided, use it as the effective searcher
	if data.SystemQuery != "" {
		// System queries are generated by the system, so they shouldn't have syntax errors
		systemSearcher, err := gensearch.GetSearcher(data.SystemQuery)
		if err != nil {
			return rpctypes.WatchSearchResultData{}, fmt.Errorf("invalid system query: %w", err)
		}
		effectiveSearcher = systemSearcher
	}

	// Create search context with user searcher for #userquery references
	sctx := &gensearch.SearchContext{
		UserQuery: userSearcher,
	}

	// Perform the search
	filteredWatches, stats, err := gensearch.PerformSearch(
		allWatches,
		totalCount,
		combinedWatchSampleToSearchObject,
		effectiveSearcher,
		sctx,
	)
	if err != nil {
		return rpctypes.WatchSearchResultData{}, err
	}

	// Extract WatchNums from filtered results
	results := make([]int64, 0, len(filteredWatches))
	for _, watch := range filteredWatches {
		results = append(results, watch.WatchNum)
	}

	// Sort the results by watch ID for consistent ordering
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})

	return rpctypes.WatchSearchResultData{
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

// SendTEventFeCommand sends a telemetry event from the frontend
func (*RpcServerImpl) SendTEventFeCommand(ctx context.Context, data rpctypes.TEventFeData) error {
	// Create a TEvent from the frontend data
	props := tevent.TEventProps{
		FrontendClickType:      data.Props.FrontendClickType,
		FrontendTab:            data.Props.FrontendTab,
		FrontendSearchFeatures: data.Props.FrontendSearchFeatures,
		FrontendSearchLatency:  data.Props.FrontendSearchLatency,
		FrontendSearchItems:    data.Props.FrontendSearchItems,
	}

	event := tevent.MakeTEvent(data.Event, props)
	tevent.WriteTEvent(*event)
	return nil
}

// UpdateCheckCommand returns information about available updates
func (*RpcServerImpl) UpdateCheckCommand(ctx context.Context) (rpctypes.UpdateCheckData, error) {
	newerVersion := updatecheck.GetUpdatedVersion()
	fromTrayApp := updatecheck.GetFromTrayApp()

	return rpctypes.UpdateCheckData{
		NewerVersion: newerVersion,
		FromTrayApp:  fromTrayApp,
	}, nil
}

// TriggerTrayUpdateCommand triggers an update check in the tray app
func (*RpcServerImpl) TriggerTrayUpdateCommand(ctx context.Context) error {
	return updatecheck.TriggerTrayAppUpdateCheck()
}

// ClearNonActiveAppRunsCommand removes all AppPeers for non-connected app runs
func (*RpcServerImpl) ClearNonActiveAppRunsCommand(ctx context.Context) error {
	return apppeer.ClearNonActiveAppRuns()
}

// LaunchDemoAppCommand launches the demo application
func (*RpcServerImpl) LaunchDemoAppCommand(ctx context.Context) error {
	return democontroller.LaunchDemoApp()
}

// KillDemoAppCommand kills the demo application
func (*RpcServerImpl) KillDemoAppCommand(ctx context.Context) error {
	return democontroller.KillDemoApp()
}

// GetDemoAppStatusCommand returns the status of the demo application
func (*RpcServerImpl) GetDemoAppStatusCommand(ctx context.Context) (string, error) {
	status, err := democontroller.GetDemoAppStatus()
	if err != nil {
		return status, err
	}
	return status, nil
}
