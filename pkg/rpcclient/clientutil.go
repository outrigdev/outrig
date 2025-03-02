// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package wshclient

import (
	"errors"

	"github.com/outrigdev/outrig/pkg/panichandler"
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

func SendRpcRequestCallHelper[T any](w *rpc.RpcClient, command string, data interface{}, opts *rpc.RpcOpts) (T, error) {
	if opts == nil {
		opts = &rpc.RpcOpts{}
	}
	var respData T
	if w == nil {
		return respData, errors.New("nil RpcClient passed to rpcclient")
	}
	if opts.NoResponse {
		err := w.SendCommand(command, data, opts)
		if err != nil {
			return respData, err
		}
		return respData, nil
	}
	resp, err := w.SendRpcRequest(command, data, opts)
	if err != nil {
		return respData, err
	}
	err = utilfn.ReUnmarshal(&respData, resp)
	if err != nil {
		return respData, err
	}
	return respData, nil
}

func RtnStreamErr[T any](ch chan rpc.RespUnion[T], err error) {
	go func() {
		defer func() {
			panichandler.PanicHandler("wshclientutil:rtnErr", recover())
		}()
		ch <- rpc.RespUnion[T]{Error: err}
		close(ch)
	}()
}

func SendRpcRequestResponseStreamHelper[T any](w *rpc.RpcClient, command string, data interface{}, opts *rpc.RpcOpts) chan rpc.RespUnion[T] {
	if opts == nil {
		opts = &rpc.RpcOpts{}
	}
	respChan := make(chan rpc.RespUnion[T], 32)
	if w == nil {
		RtnStreamErr(respChan, errors.New("nil wshrpc passed to wshclient"))
		return respChan
	}
	reqHandler, err := w.SendComplexRequest(command, data, opts)
	if err != nil {
		RtnStreamErr(respChan, err)
		return respChan
	}
	opts.StreamCancelFn = func() {
		// TODO coordinate the cancel with the for loop below
		reqHandler.SendCancel()
	}
	go func() {
		defer func() {
			panichandler.PanicHandler("sendRpcRequestResponseStreamHelper", recover())
		}()
		defer close(respChan)
		for {
			if reqHandler.ResponseDone() {
				break
			}
			resp, err := reqHandler.NextResponse()
			if err != nil {
				respChan <- rpc.RespUnion[T]{Error: err}
				break
			}
			var respData T
			err = utilfn.ReUnmarshal(&respData, resp)
			if err != nil {
				respChan <- rpc.RespUnion[T]{Error: err}
				break
			}
			respChan <- rpc.RespUnion[T]{Response: respData}
		}
	}()
	return respChan
}
