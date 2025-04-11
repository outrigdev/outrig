// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"sync"

	"github.com/outrigdev/outrig/pkg/panichandler"
)

type WshRpcProxy struct {
	Lock         *sync.Mutex
	ToRemoteCh   chan []byte
	FromRemoteCh chan []byte
}

func MakeRpcProxy() *WshRpcProxy {
	return &WshRpcProxy{
		Lock:         &sync.Mutex{},
		ToRemoteCh:   make(chan []byte, DefaultInputChSize),
		FromRemoteCh: make(chan []byte, DefaultOutputChSize),
	}
}

func (p *WshRpcProxy) SendRpcMessage(msg []byte) {
	defer func() {
		panichandler.PanicHandler("WshRpcProxy.SendRpcMessage", recover())
	}()
	p.ToRemoteCh <- msg
}

func (p *WshRpcProxy) RecvRpcMessage() ([]byte, bool) {
	msgBytes, more := <-p.FromRemoteCh
	return msgBytes, more
}
