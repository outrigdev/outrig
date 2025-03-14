// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Generated Code. DO NOT EDIT.

package wshclient

import (
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
)

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

// command "getapprungoroutines", rpctypes.GetAppRunGoroutinesCommand
func GetAppRunGoroutinesCommand(w *rpc.RpcClient, data rpctypes.AppRunRequest, opts *rpc.RpcOpts) (rpctypes.AppRunGoroutinesData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.AppRunGoroutinesData](w, "getapprungoroutines", data, opts)
	return resp, err
}

// command "getapprunlogs", rpctypes.GetAppRunLogsCommand
func GetAppRunLogsCommand(w *rpc.RpcClient, data rpctypes.AppRunRequest, opts *rpc.RpcOpts) (rpctypes.AppRunLogsData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.AppRunLogsData](w, "getapprunlogs", data, opts)
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

// command "getapprunwatches", rpctypes.GetAppRunWatchesCommand
func GetAppRunWatchesCommand(w *rpc.RpcClient, data rpctypes.AppRunRequest, opts *rpc.RpcOpts) (rpctypes.AppRunWatchesData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.AppRunWatchesData](w, "getapprunwatches", data, opts)
	return resp, err
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

// command "updatebrowsertaburl", rpctypes.UpdateBrowserTabUrlCommand
func UpdateBrowserTabUrlCommand(w *rpc.RpcClient, data rpctypes.BrowserTabUrlData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "updatebrowsertaburl", data, opts)
	return err
}

// command "updatestatus", rpctypes.UpdateStatusCommand
func UpdateStatusCommand(w *rpc.RpcClient, data rpctypes.StatusUpdateData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "updatestatus", data, opts)
	return err
}


