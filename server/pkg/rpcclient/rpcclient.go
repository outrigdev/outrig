// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Generated Code. DO NOT EDIT.

package rpcclient

import (
	"github.com/outrigdev/outrig/server/pkg/rpc"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

// command "clearnonactiveappruns", rpctypes.ClearNonActiveAppRunsCommand
func ClearNonActiveAppRunsCommand(w *rpc.RpcClient, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "clearnonactiveappruns", nil, opts)
	return err
}

// command "eventpublish", rpctypes.EventPublishCommand
func EventPublishCommand(w *rpc.RpcClient, data rpctypes.EventType, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "eventpublish", data, opts)
	return err
}

// command "eventreadhistory", rpctypes.EventReadHistoryCommand
func EventReadHistoryCommand(w *rpc.RpcClient, data rpctypes.EventReadHistoryData, opts *rpc.RpcOpts) ([]*rpctypes.EventType, error) {
	resp, err := SendRpcRequestCallHelper[[]*rpctypes.EventType](w, "eventreadhistory", data, opts)
	return resp, err
}

// command "eventsub", rpctypes.EventSubCommand
func EventSubCommand(w *rpc.RpcClient, data rpctypes.SubscriptionRequest, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "eventsub", data, opts)
	return err
}

// command "eventunsub", rpctypes.EventUnsubCommand
func EventUnsubCommand(w *rpc.RpcClient, data string, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "eventunsub", data, opts)
	return err
}

// command "eventunsuball", rpctypes.EventUnsubAllCommand
func EventUnsubAllCommand(w *rpc.RpcClient, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "eventunsuball", nil, opts)
	return err
}

// command "getapprungoroutinesbyids", rpctypes.GetAppRunGoRoutinesByIdsCommand
func GetAppRunGoRoutinesByIdsCommand(w *rpc.RpcClient, data rpctypes.AppRunGoRoutinesByIdsRequest, opts *rpc.RpcOpts) (rpctypes.AppRunGoRoutinesData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.AppRunGoRoutinesData](w, "getapprungoroutinesbyids", data, opts)
	return resp, err
}

// command "getapprunruntimestats", rpctypes.GetAppRunRuntimeStatsCommand
func GetAppRunRuntimeStatsCommand(w *rpc.RpcClient, data rpctypes.AppRunRequest, opts *rpc.RpcOpts) (rpctypes.AppRunRuntimeStatsData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.AppRunRuntimeStatsData](w, "getapprunruntimestats", data, opts)
	return resp, err
}

// command "getappruns", rpctypes.GetAppRunsCommand
func GetAppRunsCommand(w *rpc.RpcClient, data rpctypes.AppRunUpdatesRequest, opts *rpc.RpcOpts) (rpctypes.AppRunsData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.AppRunsData](w, "getappruns", data, opts)
	return resp, err
}

// command "getapprunwatchesbyids", rpctypes.GetAppRunWatchesByIdsCommand
func GetAppRunWatchesByIdsCommand(w *rpc.RpcClient, data rpctypes.AppRunWatchesByIdsRequest, opts *rpc.RpcOpts) (rpctypes.AppRunWatchesData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.AppRunWatchesData](w, "getapprunwatchesbyids", data, opts)
	return resp, err
}

// command "getdemoappstatus", rpctypes.GetDemoAppStatusCommand
func GetDemoAppStatusCommand(w *rpc.RpcClient, opts *rpc.RpcOpts) (string, error) {
	resp, err := SendRpcRequestCallHelper[string](w, "getdemoappstatus", nil, opts)
	return resp, err
}

// command "goroutinesearchrequest", rpctypes.GoRoutineSearchRequestCommand
func GoRoutineSearchRequestCommand(w *rpc.RpcClient, data rpctypes.GoRoutineSearchRequestData, opts *rpc.RpcOpts) (rpctypes.GoRoutineSearchResultData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.GoRoutineSearchResultData](w, "goroutinesearchrequest", data, opts)
	return resp, err
}

// command "killdemoapp", rpctypes.KillDemoAppCommand
func KillDemoAppCommand(w *rpc.RpcClient, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "killdemoapp", nil, opts)
	return err
}

// command "launchdemoapp", rpctypes.LaunchDemoAppCommand
func LaunchDemoAppCommand(w *rpc.RpcClient, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "launchdemoapp", nil, opts)
	return err
}

// command "loggetmarkedlines", rpctypes.LogGetMarkedLinesCommand
func LogGetMarkedLinesCommand(w *rpc.RpcClient, data rpctypes.MarkedLinesRequestData, opts *rpc.RpcOpts) (rpctypes.MarkedLinesResultData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.MarkedLinesResultData](w, "loggetmarkedlines", data, opts)
	return resp, err
}

// command "logsearchrequest", rpctypes.LogSearchRequestCommand
func LogSearchRequestCommand(w *rpc.RpcClient, data rpctypes.SearchRequestData, opts *rpc.RpcOpts) (rpctypes.SearchResultData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.SearchResultData](w, "logsearchrequest", data, opts)
	return resp, err
}

// command "logstreamupdate", rpctypes.LogStreamUpdateCommand
func LogStreamUpdateCommand(w *rpc.RpcClient, data rpctypes.StreamUpdateData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "logstreamupdate", data, opts)
	return err
}

// command "logupdatemarkedlines", rpctypes.LogUpdateMarkedLinesCommand
func LogUpdateMarkedLinesCommand(w *rpc.RpcClient, data rpctypes.MarkedLinesData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "logupdatemarkedlines", data, opts)
	return err
}

// command "logwidgetadmin", rpctypes.LogWidgetAdminCommand
func LogWidgetAdminCommand(w *rpc.RpcClient, data rpctypes.LogWidgetAdminData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "logwidgetadmin", data, opts)
	return err
}

// command "message", rpctypes.MessageCommand
func MessageCommand(w *rpc.RpcClient, data rpctypes.CommandMessageData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "message", data, opts)
	return err
}

// command "sendteventfe", rpctypes.SendTEventFeCommand
func SendTEventFeCommand(w *rpc.RpcClient, data rpctypes.TEventFeData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "sendteventfe", data, opts)
	return err
}

// command "triggertrayupdate", rpctypes.TriggerTrayUpdateCommand
func TriggerTrayUpdateCommand(w *rpc.RpcClient, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "triggertrayupdate", nil, opts)
	return err
}

// command "updatebrowsertaburl", rpctypes.UpdateBrowserTabUrlCommand
func UpdateBrowserTabUrlCommand(w *rpc.RpcClient, data rpctypes.BrowserTabUrlData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "updatebrowsertaburl", data, opts)
	return err
}

// command "updatecheck", rpctypes.UpdateCheckCommand
func UpdateCheckCommand(w *rpc.RpcClient, opts *rpc.RpcOpts) (rpctypes.UpdateCheckData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.UpdateCheckData](w, "updatecheck", nil, opts)
	return resp, err
}

// command "updatestatus", rpctypes.UpdateStatusCommand
func UpdateStatusCommand(w *rpc.RpcClient, data rpctypes.StatusUpdateData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "updatestatus", data, opts)
	return err
}

// command "watchsearchrequest", rpctypes.WatchSearchRequestCommand
func WatchSearchRequestCommand(w *rpc.RpcClient, data rpctypes.WatchSearchRequestData, opts *rpc.RpcOpts) (rpctypes.WatchSearchResultData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.WatchSearchResultData](w, "watchsearchrequest", data, opts)
	return resp, err
}


