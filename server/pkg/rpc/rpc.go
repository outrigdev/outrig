// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/panichandler"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

const DefaultTimeoutMs = 5000
const RespChSize = 32
const DefaultMessageChSize = 32
const CtxDoneChSize = 10
const DefaultInputChSize = 32
const DefaultOutputChSize = 32

var blockingExpMap = MakeExpMap[bool]()

const (
	RpcType_Call             = "call"             // single response (regular rpc)
	RpcType_ResponseStream   = "responsestream"   // stream of responses (streaming rpc)
	RpcType_StreamingRequest = "streamingrequest" // streaming request
	RpcType_Complex          = "complex"          // streaming request/response
)

// returns true if handler is complete, false for an async handler
type CommandHandlerFnType = func(*RpcResponseHandler) bool

type RpcServerImpl interface {
	RpcServerImpl()
}

type AbstractRpcClient interface {
	SendRpcMessage(msg []byte)
	RecvRpcMessage() ([]byte, bool) // blocking
}

type RpcClient struct {
	Lock               *sync.Mutex
	InputCh            chan []byte
	OutputCh           chan []byte
	CtxDoneCh          chan string // for context cancellation, value is ResId
	AuthToken          string
	RpcMap             map[string]*rpcData
	ServerImpl         RpcServerImpl
	EventListener      *EventListener
	ResponseHandlerMap map[string]*RpcResponseHandler // reqId => handler
	Debug              bool
	DebugName          string
	ServerDone         bool
}

type RpcOpts struct {
	Timeout        int64  `json:"timeout,omitempty"`
	NoResponse     bool   `json:"noresponse,omitempty"`
	Route          string `json:"route,omitempty"`
	StreamCancelFn func() `json:"-"` // this is an *output* parameter, set by the handler
}

type rpcContextKey struct{}
type rpcRespHandlerContextKey struct{}

func withRpcClientContext(ctx context.Context, rpcClient *RpcClient) context.Context {
	return context.WithValue(ctx, rpcContextKey{}, rpcClient)
}

func withRespHandler(ctx context.Context, handler *RpcResponseHandler) context.Context {
	return context.WithValue(ctx, rpcRespHandlerContextKey{}, handler)
}

func GetRpcClientFromContext(ctx context.Context) *RpcClient {
	rtn := ctx.Value(rpcContextKey{})
	if rtn == nil {
		return nil
	}
	return rtn.(*RpcClient)
}

func GetRpcSourceFromContext(ctx context.Context) string {
	rtn := ctx.Value(rpcRespHandlerContextKey{})
	if rtn == nil {
		return ""
	}
	return rtn.(*RpcResponseHandler).GetSource()
}

func GetIsCanceledFromContext(ctx context.Context) bool {
	rtn := ctx.Value(rpcRespHandlerContextKey{})
	if rtn == nil {
		return false
	}
	return rtn.(*RpcResponseHandler).IsCanceled()
}

func GetRpcResponseHandlerFromContext(ctx context.Context) *RpcResponseHandler {
	rtn := ctx.Value(rpcRespHandlerContextKey{})
	if rtn == nil {
		return nil
	}
	return rtn.(*RpcResponseHandler)
}

func (w *RpcClient) SendRpcMessage(msg []byte) {
	w.InputCh <- msg
}

func (w *RpcClient) RecvRpcMessage() ([]byte, bool) {
	msg, more := <-w.OutputCh
	return msg, more
}

type RpcMessage struct {
	Command   string `json:"command,omitempty"`
	ReqId     string `json:"reqid,omitempty"`
	ResId     string `json:"resid,omitempty"`
	Timeout   int64  `json:"timeout,omitempty"`
	Route     string `json:"route,omitempty"`     // to route/forward requests to alternate servers
	AuthToken string `json:"authtoken,omitempty"` // needed for routing unauthenticated requests (RpcMultiProxy)
	Source    string `json:"source,omitempty"`    // source route id
	Cont      bool   `json:"cont,omitempty"`      // flag if additional requests/responses are forthcoming
	Cancel    bool   `json:"cancel,omitempty"`    // used to cancel a streaming request or response (sent from the side that is not streaming)
	Error     string `json:"error,omitempty"`
	DataType  string `json:"datatype,omitempty"`
	Data      any    `json:"data,omitempty"`
}

func (r *RpcMessage) IsRpcRequest() bool {
	return r.Command != "" || r.ReqId != ""
}

func (r *RpcMessage) Validate() error {
	if r.ReqId != "" && r.ResId != "" {
		return fmt.Errorf("request packets may not have both reqid and resid set")
	}
	if r.Cancel {
		if r.Command != "" {
			return fmt.Errorf("cancel packets may not have command set")
		}
		if r.ReqId == "" && r.ResId == "" {
			return fmt.Errorf("cancel packets must have reqid or resid set")
		}
		if r.Data != nil {
			return fmt.Errorf("cancel packets may not have data set")
		}
		return nil
	}
	if r.Command != "" {
		if r.ResId != "" {
			return fmt.Errorf("command packets may not have resid set")
		}
		if r.Error != "" {
			return fmt.Errorf("command packets may not have error set")
		}
		if r.DataType != "" {
			return fmt.Errorf("command packets may not have datatype set")
		}
		return nil
	}
	if r.ReqId != "" {
		if r.ResId == "" {
			return fmt.Errorf("request packets must have resid set")
		}
		if r.Timeout != 0 {
			return fmt.Errorf("non-command request packets may not have timeout set")
		}
		return nil
	}
	if r.ResId != "" {
		if r.Command != "" {
			return fmt.Errorf("response packets may not have command set")
		}
		if r.ReqId == "" {
			return fmt.Errorf("response packets must have reqid set")
		}
		if r.Timeout != 0 {
			return fmt.Errorf("response packets may not have timeout set")
		}
		return nil
	}
	return fmt.Errorf("invalid packet: must have command, reqid, or resid set")
}

type rpcData struct {
	Command string
	Route   string
	ResCh   chan *RpcMessage
	Handler *RpcRequestHandler
}

func validateServerImpl(serverImpl RpcServerImpl) {
	if serverImpl == nil {
		return
	}
	serverType := reflect.TypeOf(serverImpl)
	if serverType.Kind() != reflect.Pointer && serverType.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("serverImpl must be a pointer to struct, got %v", serverType))
	}
}

// closes outputCh when inputCh is closed/done
func MakeRpcClient(inputCh chan []byte, outputCh chan []byte, serverImpl RpcServerImpl, debugName string) *RpcClient {
	if inputCh == nil {
		inputCh = make(chan []byte, DefaultInputChSize)
	}
	if outputCh == nil {
		outputCh = make(chan []byte, DefaultOutputChSize)
	}
	validateServerImpl(serverImpl)
	rtn := &RpcClient{
		Lock:               &sync.Mutex{},
		DebugName:          debugName,
		InputCh:            inputCh,
		OutputCh:           outputCh,
		CtxDoneCh:          make(chan string, CtxDoneChSize),
		RpcMap:             make(map[string]*rpcData),
		ServerImpl:         serverImpl,
		EventListener:      MakeEventListener(),
		ResponseHandlerMap: make(map[string]*RpcResponseHandler),
	}
	go func() {
		outrig.SetGoRoutineName("rpc." + debugName)
		rtn.runServer()
	}()
	return rtn
}

func (w *RpcClient) SetAuthToken(token string) {
	w.AuthToken = token
}

func (w *RpcClient) GetAuthToken() string {
	return w.AuthToken
}

func (w *RpcClient) registerResponseHandler(reqId string, handler *RpcResponseHandler) {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	w.ResponseHandlerMap[reqId] = handler
}

func (w *RpcClient) unregisterResponseHandler(reqId string) {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	delete(w.ResponseHandlerMap, reqId)
}

func (w *RpcClient) cancelRequest(reqId string) {
	if reqId == "" {
		return
	}
	w.Lock.Lock()
	defer w.Lock.Unlock()
	handler := w.ResponseHandlerMap[reqId]
	if handler != nil {
		handler.canceled.Store(true)
	}

}

func (w *RpcClient) handleRequest(req *RpcMessage) {
	// events first
	if req.Command == rpctypes.Command_EventRecv {
		if req.Data == nil {
			// invalid
			return
		}
		var waveEvent EventType
		err := utilfn.ReUnmarshal(&waveEvent, req.Data)
		if err != nil {
			// invalid
			return
		}
		w.EventListener.RecvEvent(&waveEvent)
		return
	}

	var respHandler *RpcResponseHandler
	timeoutMs := req.Timeout
	if timeoutMs <= 0 {
		timeoutMs = DefaultTimeoutMs
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	ctx = withRpcClientContext(ctx, w)
	respHandler = &RpcResponseHandler{
		w:               w,
		ctx:             ctx,
		reqId:           req.ReqId,
		command:         req.Command,
		commandData:     req.Data,
		source:          req.Source,
		done:            &atomic.Bool{},
		canceled:        &atomic.Bool{},
		contextCancelFn: &atomic.Pointer[context.CancelFunc]{},
	}
	respHandler.contextCancelFn.Store(&cancelFn)
	respHandler.ctx = withRespHandler(ctx, respHandler)
	w.registerResponseHandler(req.ReqId, respHandler)
	isAsync := false
	defer func() {
		panicErr := panichandler.PanicHandler("handleRequest", recover())
		if panicErr != nil {
			respHandler.SendResponseError(panicErr)
		}
		if isAsync {
			go func() {
				outrig.SetGoRoutineName("rpc.fin/" + w.DebugName)
				defer func() {
					panichandler.PanicHandler("handleRequest:finalize", recover())
				}()
				<-ctx.Done()
				respHandler.Finalize()
			}()
		} else {
			cancelFn()
			respHandler.Finalize()
		}
	}()
	handlerFn := serverImplAdapter(w.ServerImpl)
	isAsync = !handlerFn(respHandler)
}

func (w *RpcClient) runServer() {
	defer func() {
		panichandler.PanicHandler("rpc.runServer", recover())
		close(w.OutputCh)
		w.setServerDone()
	}()
outer:
	for {
		var msgBytes []byte
		var inputChMore bool
		var resIdTimeout string

		select {
		case msgBytes, inputChMore = <-w.InputCh:
			if !inputChMore {
				break outer
			}
			if w.Debug {
				log.Printf("[%s] received message: %s\n", w.DebugName, string(msgBytes))
			}
		case resIdTimeout = <-w.CtxDoneCh:
			if w.Debug {
				log.Printf("[%s] received request timeout: %s\n", w.DebugName, resIdTimeout)
			}
			w.unregisterRpc(resIdTimeout, fmt.Errorf("EC-TIME: timeout waiting for response"))
			continue
		}

		var msg RpcMessage
		err := json.Unmarshal(msgBytes, &msg)
		if err != nil {
			log.Printf("[%s] rpcclient received bad message: %v\n", w.DebugName, err)
			continue
		}
		if msg.Cancel {
			if msg.ReqId != "" {
				w.cancelRequest(msg.ReqId)
			}
			continue
		}
		if msg.IsRpcRequest() {
			go func() {
				outrig.SetGoRoutineName("rpc.req/" + w.DebugName + "/" + msg.Command)
				defer func() {
					panichandler.PanicHandler("handleRequest:goroutine", recover())
				}()
				w.handleRequest(&msg)
			}()
		} else {
			w.sendRespWithBlockMessage(msg)
			if !msg.Cont {
				w.unregisterRpc(msg.ResId, nil)
			}
		}
	}
}

func (w *RpcClient) getResponseCh(resId string) (chan *RpcMessage, *rpcData) {
	if resId == "" {
		return nil, nil
	}
	w.Lock.Lock()
	defer w.Lock.Unlock()
	rd := w.RpcMap[resId]
	if rd == nil {
		return nil, nil
	}
	return rd.ResCh, rd
}

func (w *RpcClient) SetServerImpl(serverImpl RpcServerImpl) {
	validateServerImpl(serverImpl)
	w.Lock.Lock()
	defer w.Lock.Unlock()
	w.ServerImpl = serverImpl
}

func (w *RpcClient) registerRpc(handler *RpcRequestHandler, command string, route string, reqId string) chan *RpcMessage {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	rpcCh := make(chan *RpcMessage, RespChSize)
	w.RpcMap[reqId] = &rpcData{
		Handler: handler,
		Command: command,
		Route:   route,
		ResCh:   rpcCh,
	}
	go func() {
		outrig.SetGoRoutineName("rpc.timeout")
		defer func() {
			panichandler.PanicHandler("registerRpc:timeout", recover())
		}()
		<-handler.ctx.Done()
		w.retrySendTimeout(reqId)
	}()
	return rpcCh
}

func (w *RpcClient) unregisterRpc(reqId string, err error) {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	rd := w.RpcMap[reqId]
	if rd == nil {
		return
	}
	if err != nil {
		errResp := &RpcMessage{
			ResId: reqId,
			Error: err.Error(),
		}
		// non-blocking send since we're about to close anyway
		// likely the channel isn't being actively read
		// this also prevents us from blocking the main loop (and holding the lock)
		select {
		case rd.ResCh <- errResp:
		default:
		}
	}
	delete(w.RpcMap, reqId)
	close(rd.ResCh)
	rd.Handler.callContextCancelFn()
}

// no response
func (w *RpcClient) SendCommand(command string, data any, opts *RpcOpts) error {
	var optsCopy RpcOpts
	if opts != nil {
		optsCopy = *opts
	}
	optsCopy.NoResponse = true
	optsCopy.Timeout = 0
	handler, err := w.SendComplexRequest(command, data, &optsCopy)
	if err != nil {
		return err
	}
	handler.finalize()
	return nil
}

// single response
func (w *RpcClient) SendRpcRequest(command string, data any, opts *RpcOpts) (any, error) {
	var optsCopy RpcOpts
	if opts != nil {
		optsCopy = *opts
	}
	optsCopy.NoResponse = false
	handler, err := w.SendComplexRequest(command, data, &optsCopy)
	if err != nil {
		return nil, err
	}
	defer handler.finalize()
	return handler.NextResponse()
}

type RpcRequestHandler struct {
	w           *RpcClient
	ctx         context.Context
	ctxCancelFn *atomic.Pointer[context.CancelFunc]
	reqId       string
	respCh      chan *RpcMessage
	cachedResp  *RpcMessage
}

func (handler *RpcRequestHandler) Context() context.Context {
	return handler.ctx
}

func (handler *RpcRequestHandler) SendCancel() {
	defer func() {
		panichandler.PanicHandler("SendCancel", recover())
	}()
	msg := &RpcMessage{
		Cancel:    true,
		ReqId:     handler.reqId,
		AuthToken: handler.w.GetAuthToken(),
	}
	barr, _ := json.Marshal(msg) // will never fail
	handler.w.OutputCh <- barr
	handler.finalize()
}

func (handler *RpcRequestHandler) ResponseDone() bool {
	if handler.cachedResp != nil {
		return false
	}
	select {
	case msg, more := <-handler.respCh:
		if !more {
			return true
		}
		handler.cachedResp = msg
		return false
	default:
		return false
	}
}

func (handler *RpcRequestHandler) NextResponse() (any, error) {
	var resp *RpcMessage
	if handler.cachedResp != nil {
		resp = handler.cachedResp
		handler.cachedResp = nil
	} else {
		resp = <-handler.respCh
	}
	if resp == nil {
		return nil, errors.New("response channel closed")
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Data, nil
}

func (handler *RpcRequestHandler) finalize() {
	handler.callContextCancelFn()
	if handler.reqId != "" {
		handler.w.unregisterRpc(handler.reqId, nil)
	}
}

func (handler *RpcRequestHandler) callContextCancelFn() {
	cancelFnPtr := handler.ctxCancelFn.Swap(nil)
	if cancelFnPtr != nil && *cancelFnPtr != nil {
		(*cancelFnPtr)()
	}
}

type RpcResponseHandler struct {
	w               *RpcClient
	ctx             context.Context
	contextCancelFn *atomic.Pointer[context.CancelFunc]
	reqId           string
	source          string
	command         string
	commandData     any
	canceled        *atomic.Bool // canceled by requestor
	done            *atomic.Bool
}

func (handler *RpcResponseHandler) Context() context.Context {
	return handler.ctx
}

func (handler *RpcResponseHandler) GetCommand() string {
	return handler.command
}

func (handler *RpcResponseHandler) GetCommandRawData() any {
	return handler.commandData
}

func (handler *RpcResponseHandler) GetSource() string {
	return handler.source
}

func (handler *RpcResponseHandler) NeedsResponse() bool {
	return handler.reqId != ""
}

func (handler *RpcResponseHandler) SendMessage(msg string) {
	rpcMsg := &RpcMessage{
		Command: rpctypes.Command_Message,
		Data: rpctypes.CommandMessageData{
			Message: msg,
		},
		AuthToken: handler.w.GetAuthToken(),
	}
	msgBytes, _ := json.Marshal(rpcMsg) // will never fail
	handler.w.OutputCh <- msgBytes
}

func (handler *RpcResponseHandler) SendResponse(data any, done bool) error {
	defer func() {
		panichandler.PanicHandler("SendResponse", recover())
	}()
	if handler.reqId == "" {
		return nil // no response expected
	}
	if handler.done.Load() {
		return fmt.Errorf("request already done, cannot send additional response")
	}
	if done {
		defer handler.close()
	}
	msg := &RpcMessage{
		ResId:     handler.reqId,
		Data:      data,
		Cont:      !done,
		AuthToken: handler.w.GetAuthToken(),
	}
	barr, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	handler.w.OutputCh <- barr
	return nil
}

func (handler *RpcResponseHandler) SendResponseError(err error) {
	defer func() {
		panichandler.PanicHandler("SendResponseError", recover())
	}()
	if handler.reqId == "" || handler.done.Load() {
		return
	}
	defer handler.close()
	msg := &RpcMessage{
		ResId:     handler.reqId,
		Error:     err.Error(),
		AuthToken: handler.w.GetAuthToken(),
	}
	barr, _ := json.Marshal(msg) // will never fail
	handler.w.OutputCh <- barr
}

func (handler *RpcResponseHandler) IsCanceled() bool {
	return handler.canceled.Load()
}

func (handler *RpcResponseHandler) close() {
	cancelFn := handler.contextCancelFn.Load()
	if cancelFn != nil && *cancelFn != nil {
		(*cancelFn)()
		handler.contextCancelFn.Store(nil)
	}
	handler.done.Store(true)
}

// if async, caller must call finalize
func (handler *RpcResponseHandler) Finalize() {
	if handler.reqId == "" || handler.done.Load() {
		return
	}
	handler.SendResponse(nil, true)
	handler.close()
	handler.w.unregisterResponseHandler(handler.reqId)
}

func (handler *RpcResponseHandler) IsDone() bool {
	return handler.done.Load()
}

func (w *RpcClient) SendComplexRequest(command string, data any, opts *RpcOpts) (rtnHandler *RpcRequestHandler, rtnErr error) {
	if w.IsServerDone() {
		return nil, errors.New("server is no longer running, cannot send new requests")
	}
	if opts == nil {
		opts = &RpcOpts{}
	}
	timeoutMs := opts.Timeout
	if timeoutMs <= 0 {
		timeoutMs = DefaultTimeoutMs
	}
	defer func() {
		panichandler.PanicHandler("SendComplexRequest", recover())
	}()
	if command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}
	handler := &RpcRequestHandler{
		w:           w,
		ctxCancelFn: &atomic.Pointer[context.CancelFunc]{},
	}
	var cancelFn context.CancelFunc
	handler.ctx, cancelFn = context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	handler.ctxCancelFn.Store(&cancelFn)
	if !opts.NoResponse {
		handler.reqId = uuid.New().String()
	}
	req := &RpcMessage{
		Command:   command,
		ReqId:     handler.reqId,
		Data:      data,
		Timeout:   timeoutMs,
		Route:     opts.Route,
		AuthToken: w.GetAuthToken(),
	}
	barr, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	handler.respCh = w.registerRpc(handler, command, opts.Route, handler.reqId)
	w.OutputCh <- barr
	return handler, nil
}

func (w *RpcClient) IsServerDone() bool {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	return w.ServerDone
}

func (w *RpcClient) setServerDone() {
	w.Lock.Lock()
	defer w.Lock.Unlock()
	w.ServerDone = true
	close(w.CtxDoneCh)
	go func() {
		outrig.SetGoRoutineName("rpc.drain")
		utilfn.DrainChan(w.InputCh)
	}()
}

func (w *RpcClient) retrySendTimeout(resId string) {
	done := func() bool {
		w.Lock.Lock()
		defer w.Lock.Unlock()
		if w.ServerDone {
			return true
		}
		select {
		case w.CtxDoneCh <- resId:
			return true
		default:
			return false
		}
	}
	for {
		if done() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (w *RpcClient) sendRespWithBlockMessage(msg RpcMessage) {
	respCh, rd := w.getResponseCh(msg.ResId)
	if respCh == nil {
		return
	}
	select {
	case respCh <- &msg:
		// normal case, message got sent, just return!
		return
	default:
		// channel is full, we would block...
	}
	// log the fact that we're blocking
	_, noLog := blockingExpMap.Get(msg.ResId)
	if !noLog {
		log.Printf("[%s] blocking on response command:%s route:%s resid:%s\n", w.DebugName, rd.Command, rd.Route, msg.ResId)
		blockingExpMap.Set(msg.ResId, true, time.Now().Add(time.Second))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	select {
	case respCh <- &msg:
		// message got sent, just return!
		return
	case <-ctx.Done():
	}
	log.Printf("[%s] failed to clear response channel (waited 1s), will fail RPC command:%s route:%s resid:%s\n", w.DebugName, rd.Command, rd.Route, msg.ResId)
	w.unregisterRpc(msg.ResId, nil) // we don't pass an error because the channel is full, it won't work anyway...
}
