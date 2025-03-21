package rpcserver

import (
	"context"
	"fmt"
	"strconv"

	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
	"github.com/outrigdev/outrig/server/pkg/browsertabs"
	"github.com/outrigdev/outrig/server/pkg/logsearch"
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
	return apppeer.GetAppRunGoRoutinesCommand(ctx, data)
}

// GetAppRunWatchesCommand returns watches for a specific app run
func (*RpcServerImpl) GetAppRunWatchesCommand(ctx context.Context, data rpctypes.AppRunRequest) (rpctypes.AppRunWatchesData, error) {
	return apppeer.GetAppRunWatches(ctx, data)
}

// GetAppRunRuntimeStatsCommand returns runtime stats for a specific app run
func (*RpcServerImpl) GetAppRunRuntimeStatsCommand(ctx context.Context, data rpctypes.AppRunRequest) (rpctypes.AppRunRuntimeStatsData, error) {
	return apppeer.GetAppRunRuntimeStats(ctx, data)
}

// LogSearchRequestCommand handles search requests for logs
func (*RpcServerImpl) LogSearchRequestCommand(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	manager := logsearch.GetOrCreateManager(data.WidgetId, data.AppRunId)
	return manager.SearchRequest(ctx, data)
}

// LogWidgetAdminCommand handles widget administration requests
func (*RpcServerImpl) LogWidgetAdminCommand(ctx context.Context, data rpctypes.LogWidgetAdminData) error {
	manager := logsearch.GetManager(data.WidgetId)
	if manager == nil {
		return nil
	}

	// Store the RPC source with proper synchronization
	manager.SetRpcSource(ctx)

	// Drop takes precedence over KeepAlive
	if data.Drop {
		logsearch.DropManager(data.WidgetId)
	} else if data.KeepAlive {
		manager.UpdateLastUsed()
	}
	return nil
}

// LogUpdateMarkedLinesCommand handles updating marked lines for a widget
func (*RpcServerImpl) LogUpdateMarkedLinesCommand(ctx context.Context, data rpctypes.MarkedLinesData) error {
	manager := logsearch.GetManager(data.WidgetId)
	if manager == nil {
		return fmt.Errorf("widget not found: %s", data.WidgetId)
	}

	// If Clear flag is set, clear all marked lines
	if data.Clear {
		manager.ClearMarkedLines()
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

	// Merge the converted marked lines
	manager.MergeMarkedLines(markedLines)
	return nil
}

// LogGetMarkedLinesCommand retrieves all marked log lines for a widget
func (*RpcServerImpl) LogGetMarkedLinesCommand(ctx context.Context, data rpctypes.MarkedLinesRequestData) (rpctypes.MarkedLinesResultData, error) {
	manager := logsearch.GetManager(data.WidgetId)
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
