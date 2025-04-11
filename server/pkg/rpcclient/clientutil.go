// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package rpcclient

import (
	"errors"
	"sync"

	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/panichandler"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/rpc"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

var (
	bareClient     *rpc.RpcClient
	bareClientOnce sync.Once
)

// GetBareClient returns the BareClient, initializing it if needed
func GetBareClient() *rpc.RpcClient {
	bareClientOnce.Do(func() {
		bareClient = rpc.MakeRpcClient(nil, nil, nil, "outrigsrv-client")
		rpc.GetDefaultRouter().RegisterRoute(rpc.BareClientRoute, bareClient, true)
	})
	return bareClient
}

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

func RtnStreamErr[T any](ch chan rpctypes.RespUnion[T], err error) {
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig RtnStreamErr")
		defer func() {
			panichandler.PanicHandler("wshclientutil:rtnErr", recover())
		}()
		ch <- rpctypes.RespUnion[T]{Error: err}
		close(ch)
	}()
}

func SendRpcRequestResponseStreamHelper[T any](w *rpc.RpcClient, command string, data interface{}, opts *rpc.RpcOpts) chan rpctypes.RespUnion[T] {
	if opts == nil {
		opts = &rpc.RpcOpts{}
	}
	respChan := make(chan rpctypes.RespUnion[T], 32)
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
		ioutrig.I.SetGoRoutineName("#outrig SendRpcRequestResponseStreamHelper")
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
				respChan <- rpctypes.RespUnion[T]{Error: err}
				break
			}
			var respData T
			err = utilfn.ReUnmarshal(&respData, resp)
			if err != nil {
				respChan <- rpctypes.RespUnion[T]{Error: err}
				break
			}
			respChan <- rpctypes.RespUnion[T]{Response: respData}
		}
	}()
	return respChan
}
