package rpcserver

import (
	"context"
	"fmt"

	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
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

		// Determine if the app is still running (this is a simple heuristic)
		// In a real implementation, you might want to check if the process is still alive
		isRunning := true

		// Create AppRunInfo
		appRun := rpctypes.AppRunInfo{
			AppRunId:  peer.AppRunId,
			AppName:   peer.AppInfo.AppName,
			StartTime: peer.AppInfo.StartTime,
			IsRunning: isRunning,
			NumLogs:   peer.Logs.Size(),
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
