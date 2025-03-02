// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Generated Code. DO NOT EDIT.

package wshclient

import (
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
)

// command "droprequest", rpctypes.DropRequestCommand
func DropRequestCommand(w *rpc.RpcClient, data rpctypes.DropRequestData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "droprequest", data, opts)
	return err
}

// command "message", rpctypes.MessageCommand
func MessageCommand(w *rpc.RpcClient, data rpctypes.CommandMessageData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "message", data, opts)
	return err
}

// command "searchrequest", rpctypes.SearchRequestCommand
func SearchRequestCommand(w *rpc.RpcClient, data rpctypes.SearchRequestData, opts *rpc.RpcOpts) (rpctypes.SearchResultData, error) {
	resp, err := SendRpcRequestCallHelper[rpctypes.SearchResultData](w, "searchrequest", data, opts)
	return resp, err
}

// command "streamupdate", rpctypes.StreamUpdateCommand
func StreamUpdateCommand(w *rpc.RpcClient, data rpctypes.StreamUpdateData, opts *rpc.RpcOpts) error {
	_, err := SendRpcRequestCallHelper[any](w, "streamupdate", data, opts)
	return err
}


