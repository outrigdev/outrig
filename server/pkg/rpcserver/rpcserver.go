package rpcserver

import (
	"context"
	"fmt"

	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
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

// GetAppRunsCommand returns a list of all app runs
func (*RpcServerImpl) GetAppRunsCommand(ctx context.Context) (rpctypes.AppRunsData, error) {
	// Get all app run peers from the apppeer package
	appRunPeers := apppeer.GetAllAppRunPeers()

	// Convert to AppRunInfo slice
	appRuns := make([]rpctypes.AppRunInfo, 0, len(appRunPeers))
	for _, peer := range appRunPeers {
		// Skip peers with no AppInfo
		if peer.AppInfo == nil {
			continue
		}

		// Determine if the app is still running based on its status
		isRunning := peer.Status == apppeer.AppStatusRunning

		// Get the number of active and total goroutines
		numActiveGoRoutines := len(peer.ActiveGoRoutines)
		numTotalGoRoutines := len(peer.GoRoutines.Keys())

		// Create AppRunInfo
		appRun := rpctypes.AppRunInfo{
			AppRunId:            peer.AppRunId,
			AppName:             peer.AppInfo.AppName,
			StartTime:           peer.AppInfo.StartTime,
			IsRunning:           isRunning,
			Status:              peer.Status,
			NumLogs:             peer.Logs.Size(),
			NumActiveGoRoutines: numActiveGoRoutines,
			NumTotalGoRoutines:  numTotalGoRoutines,
		}

		appRuns = append(appRuns, appRun)
	}

	return rpctypes.AppRunsData{
		AppRuns: appRuns,
	}, nil
}

// GetAppRunLogsCommand returns logs for a specific app run
func (*RpcServerImpl) GetAppRunLogsCommand(ctx context.Context, data rpctypes.AppRunRequest) (rpctypes.AppRunLogsData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunLogsData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get logs from the circular buffer
	logLines := peer.Logs.GetAll()

	return rpctypes.AppRunLogsData{
		AppRunId: peer.AppRunId,
		AppName:  peer.AppInfo.AppName,
		Logs:     logLines,
	}, nil
}

// GetAppRunGoroutinesCommand returns goroutines for a specific app run
func (*RpcServerImpl) GetAppRunGoroutinesCommand(ctx context.Context, data rpctypes.AppRunRequest) (rpctypes.AppRunGoroutinesData, error) {
	// Get the app run peer
	peer := apppeer.GetAppRunPeer(data.AppRunId)
	if peer == nil || peer.AppInfo == nil {
		return rpctypes.AppRunGoroutinesData{}, fmt.Errorf("app run not found: %s", data.AppRunId)
	}

	// Get all goroutine keys
	goroutineKeys := peer.GoRoutines.Keys()
	goroutines := make([]rpctypes.GoroutineData, 0, len(goroutineKeys))

	// For each goroutine, get the most recent stack trace
	for _, key := range goroutineKeys {
		goroutine, exists := peer.GoRoutines.GetEx(key)
		if !exists || goroutine.StackTraces.IsEmpty() {
			continue
		}

		// Get the most recent stack trace
		stackTraces := goroutine.StackTraces.GetAll()
		if len(stackTraces) == 0 {
			continue
		}

		// Use the most recent stack trace
		latestStack := stackTraces[len(stackTraces)-1]

		goroutines = append(goroutines, rpctypes.GoroutineData{
			GoId:       latestStack.GoId,
			State:      latestStack.State,
			StackTrace: latestStack.StackTrace,
		})
	}

	return rpctypes.AppRunGoroutinesData{
		AppRunId:   peer.AppRunId,
		AppName:    peer.AppInfo.AppName,
		Goroutines: goroutines,
	}, nil
}

// LogSearchRequestCommand handles search requests for logs
func (*RpcServerImpl) LogSearchRequestCommand(ctx context.Context, data rpctypes.SearchRequestData) (rpctypes.SearchResultData, error) {
	// Get or create a search manager for this widget
	manager := logsearch.GetOrCreateManager(data.WidgetId, data.AppRunId)

	// Delegate the search request to the manager
	return manager.SearchRequest(ctx, data)
}

// LogWidgetAdminCommand handles widget administration requests
func (*RpcServerImpl) LogWidgetAdminCommand(ctx context.Context, data rpctypes.LogWidgetAdminData) error {
	manager := logsearch.GetManager(data.WidgetId)
	if manager == nil {
		return nil
	}
	// Drop takes precedence over KeepAlive
	if data.Drop {
		logsearch.DropManager(data.WidgetId)
	} else if data.KeepAlive {
		manager.UpdateLastUsed()
	}
	return nil
}
