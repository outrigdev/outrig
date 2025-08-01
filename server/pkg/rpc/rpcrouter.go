// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/panichandler"
	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

const (
	DefaultRoute    = "outrigsrv"
	BareClientRoute = "outrigsrv-client"
	UpstreamRoute   = "upstream"
	SysRoute        = "sys" // this route doesn't exist, just a placeholder for system messages
	ElectronRoute   = "electron"

	RoutePrefix_Conn       = "conn:"
	RoutePrefix_Controller = "controller:"
	RoutePrefix_Proc       = "proc:"
	RoutePrefix_Tab        = "tab:"
	RoutePrefix_FeBlock    = "feblock:"
)

// this works like a network switch

// TODO maybe move the wps integration here instead of in wshserver

type routeInfo struct {
	RpcId         string
	SourceRouteId string
	DestRouteId   string
}

type msgAndRoute struct {
	msgBytes    []byte
	fromRouteId string
}

type WshRouter struct {
	Lock             *sync.Mutex
	RouteMap         map[string]AbstractRpcClient // routeid => client
	UpstreamClient   AbstractRpcClient            // upstream client (if we are not the terminal router)
	AnnouncedRoutes  map[string]string            // routeid => local routeid
	RpcMap           map[string]*routeInfo        // rpcid => routeinfo
	SimpleRequestMap map[string]chan *RpcMessage  // simple reqid => response channel
	InputCh          chan msgAndRoute
}

func MakeConnectionRouteId(connId string) string {
	return "conn:" + connId
}

func MakeControllerRouteId(blockId string) string {
	return "controller:" + blockId
}

func MakeProcRouteId(procId string) string {
	return "proc:" + procId
}

func MakeTabRouteId(tabId string) string {
	return "tab:" + tabId
}

func MakeFeBlockRouteId(blockId string) string {
	return "feblock:" + blockId
}

var (
	defaultRouter     *WshRouter
	defaultRouterOnce sync.Once
)

// GetDefaultRouter returns the DefaultRouter, initializing it if needed
func GetDefaultRouter() *WshRouter {
	defaultRouterOnce.Do(func() {
		defaultRouter = NewWshRouter()
	})
	return defaultRouter
}

func init() {
	// Register a watch function that returns a sorted list of RouteMap keys
	outrig.NewWatch("rpcroutes").PollFunc(func() []string {
		return GetDefaultRouter().GetRouteKeys()
	})
}

// GetRouteKeys returns a sorted list of all route keys in the router
func (router *WshRouter) GetRouteKeys() []string {
	router.Lock.Lock()
	defer router.Lock.Unlock()

	keys := make([]string, 0, len(router.RouteMap))
	for key := range router.RouteMap {
		keys = append(keys, key)
	}

	// Sort the keys for consistent ordering
	sort.Strings(keys)
	return keys
}

func NewWshRouter() *WshRouter {
	rtn := &WshRouter{
		Lock:             &sync.Mutex{},
		RouteMap:         make(map[string]AbstractRpcClient),
		AnnouncedRoutes:  make(map[string]string),
		RpcMap:           make(map[string]*routeInfo),
		SimpleRequestMap: make(map[string]chan *RpcMessage),
		InputCh:          make(chan msgAndRoute, DefaultInputChSize),
	}
	go func() {
		outrig.SetGoRoutineName("router.run")
		rtn.runServer()
	}()
	return rtn
}

func (router *WshRouter) SendEvent(routeId string, event EventType) {
	defer func() {
		panichandler.PanicHandler("WshRouter.SendEvent", recover())
	}()
	rpc := router.GetRpc(routeId)
	if rpc == nil {
		return
	}
	msg := RpcMessage{
		Command: rpctypes.Command_EventRecv,
		Route:   routeId,
		Data:    event,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		// nothing to do
		return
	}
	rpc.SendRpcMessage(msgBytes)
}

func noRouteErr(routeId string) error {
	if routeId == "" {
		return errors.New("no default route")
	}
	return fmt.Errorf("no route for %q", routeId)
}

func (router *WshRouter) handleNoRoute(msg RpcMessage) {
	nrErr := noRouteErr(msg.Route)
	if msg.ReqId == "" {
		if msg.Command == rpctypes.Command_Message {
			// to prevent infinite loops
			return
		}
		// no response needed, but send message back to source
		respMsg := RpcMessage{Command: rpctypes.Command_Message, Route: msg.Source, Data: rpctypes.CommandMessageData{Message: nrErr.Error()}}
		respBytes, _ := json.Marshal(respMsg)
		router.InputCh <- msgAndRoute{msgBytes: respBytes, fromRouteId: SysRoute}
		return
	}
	// send error response
	response := RpcMessage{
		ResId: msg.ReqId,
		Error: nrErr.Error(),
	}
	respBytes, _ := json.Marshal(response)
	router.sendRoutedMessage(respBytes, msg.Source, msg.Command)
}

func (router *WshRouter) registerRouteInfo(rpcId string, sourceRouteId string, destRouteId string) {
	if rpcId == "" {
		return
	}
	router.Lock.Lock()
	defer router.Lock.Unlock()
	router.RpcMap[rpcId] = &routeInfo{RpcId: rpcId, SourceRouteId: sourceRouteId, DestRouteId: destRouteId}
}

func (router *WshRouter) unregisterRouteInfo(rpcId string) {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	delete(router.RpcMap, rpcId)
}

func (router *WshRouter) getRouteInfo(rpcId string) *routeInfo {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	return router.RpcMap[rpcId]
}

func (router *WshRouter) handleAnnounceMessage(msg RpcMessage, input msgAndRoute) {
	// if we have an upstream, send it there
	// if we don't (we are the terminal router), then add it to our announced route map
	upstream := router.GetUpstreamClient()
	if upstream != nil {
		upstream.SendRpcMessage(input.msgBytes)
		return
	}
	if msg.Source == input.fromRouteId {
		// not necessary to save the id mapping
		return
	}
	router.Lock.Lock()
	defer router.Lock.Unlock()
	router.AnnouncedRoutes[msg.Source] = input.fromRouteId
}

func (router *WshRouter) handleUnannounceMessage(msg RpcMessage) {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	delete(router.AnnouncedRoutes, msg.Source)
}

func (router *WshRouter) getAnnouncedRoute(routeId string) string {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	return router.AnnouncedRoutes[routeId]
}

// returns true if message was sent, false if failed
func (router *WshRouter) sendRoutedMessage(msgBytes []byte, routeId string, command string) bool {
	rpc := router.GetRpc(routeId)
	if rpc != nil {
		rpc.SendRpcMessage(msgBytes)
		return true
	}
	upstream := router.GetUpstreamClient()
	if upstream != nil {
		upstream.SendRpcMessage(msgBytes)
		return true
	} else {
		// we are the upstream, so consult our announced routes map
		localRouteId := router.getAnnouncedRoute(routeId)
		rpc := router.GetRpc(localRouteId)
		if rpc == nil {
			if command != "" {
				log.Printf("[router] no rpc for route id %q (command:%s)\n", routeId, command)
			} else {
				log.Printf("[router] no rpc for route id %q\n", routeId)
			}
			return false
		}
		rpc.SendRpcMessage(msgBytes)
		return true
	}
}

func (router *WshRouter) runServer() {
	for input := range router.InputCh {
		msgBytes := input.msgBytes
		var msg RpcMessage
		err := json.Unmarshal(msgBytes, &msg)
		if err != nil {
			fmt.Printf("error unmarshalling message: %v\n", err)
			continue
		}
		routeId := msg.Route
		if msg.Command == rpctypes.Command_RouteAnnounce {
			router.handleAnnounceMessage(msg, input)
			continue
		}
		if msg.Command == rpctypes.Command_RouteUnannounce {
			router.handleUnannounceMessage(msg)
			continue
		}
		if msg.Command != "" {
			// new comand, setup new rpc
			ok := router.sendRoutedMessage(msgBytes, routeId, msg.Command)
			if !ok {
				router.handleNoRoute(msg)
				continue
			}
			router.registerRouteInfo(msg.ReqId, msg.Source, routeId)
			continue
		}
		// look at reqid or resid to route correctly
		if msg.ReqId != "" {
			routeInfo := router.getRouteInfo(msg.ReqId)
			if routeInfo == nil {
				// no route info, nothing to do
				continue
			}
			// no need to check the return value here (noop if failed)
			router.sendRoutedMessage(msgBytes, routeInfo.DestRouteId, msg.Command)
			continue
		} else if msg.ResId != "" {
			ok := router.trySimpleResponse(&msg)
			if ok {
				continue
			}
			routeInfo := router.getRouteInfo(msg.ResId)
			if routeInfo == nil {
				// no route info, nothing to do
				continue
			}
			router.sendRoutedMessage(msgBytes, routeInfo.SourceRouteId, msg.Command)
			if !msg.Cont {
				router.unregisterRouteInfo(msg.ResId)
			}
			continue
		} else {
			// this is a bad message (no command, reqid, or resid)
			continue
		}
	}
}

func (router *WshRouter) WaitForRegister(ctx context.Context, routeId string) error {
	for {
		if router.GetRpc(routeId) != nil {
			return nil
		}
		if router.getAnnouncedRoute(routeId) != "" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Millisecond):
			continue
		}
	}
}

// this will also consume the output channel of the abstract client
func (router *WshRouter) RegisterRoute(routeId string, rpc AbstractRpcClient, shouldAnnounce bool) {
	if routeId == SysRoute || routeId == UpstreamRoute {
		// cannot register sys route
		log.Printf("error: WshRouter cannot register %s route\n", routeId)
		return
	}
	// log.Printf("[router] registering wsh route %q\n", routeId)
	router.Lock.Lock()
	defer router.Lock.Unlock()
	alreadyExists := router.RouteMap[routeId] != nil
	if alreadyExists {
		log.Printf("[router] warning: route %q already exists (replacing)\n", routeId)
	}
	router.RouteMap[routeId] = rpc
	go func() {
		outrig.SetGoRoutineName("router.routeup")
		defer func() {
			panichandler.PanicHandler("RpcRouter:registerRoute:routeup", recover())
		}()
		Broker.Publish(EventType{Event: rpctypes.Event_RouteUp, Scopes: []string{routeId}})
	}()
	go func() {
		outrig.SetGoRoutineName("router.recv")
		defer func() {
			panichandler.PanicHandler("WshRouter:registerRoute:recvloop", recover())
		}()
		// announce
		if shouldAnnounce && !alreadyExists && router.GetUpstreamClient() != nil {
			announceMsg := RpcMessage{Command: rpctypes.Command_RouteAnnounce, Source: routeId}
			announceBytes, _ := json.Marshal(announceMsg)
			router.GetUpstreamClient().SendRpcMessage(announceBytes)
		}
		for {
			msgBytes, ok := rpc.RecvRpcMessage()
			if !ok {
				break
			}
			var rpcMsg RpcMessage
			err := json.Unmarshal(msgBytes, &rpcMsg)
			if err != nil {
				continue
			}
			if rpcMsg.Command != "" {
				if rpcMsg.Source == "" {
					rpcMsg.Source = routeId
				}
				if rpcMsg.Route == "" {
					rpcMsg.Route = DefaultRoute
				}
				msgBytes, err = json.Marshal(rpcMsg)
				if err != nil {
					continue
				}
			}
			router.InputCh <- msgAndRoute{msgBytes: msgBytes, fromRouteId: routeId}
		}
	}()
}

func (router *WshRouter) UnregisterRoute(routeId string) {
	// log.Printf("[router] unregistering wsh route %q\n", routeId)
	router.Lock.Lock()
	defer router.Lock.Unlock()
	delete(router.RouteMap, routeId)
	// clear out announced routes
	for routeId, localRouteId := range router.AnnouncedRoutes {
		if localRouteId == routeId {
			delete(router.AnnouncedRoutes, routeId)
		}
	}
	go func() {
		outrig.SetGoRoutineName("router.routedown")
		defer func() {
			panichandler.PanicHandler("RpcRouter:unregisterRoute:routedown", recover())
		}()
		Broker.UnsubscribeAll(routeId)
		Broker.Publish(EventType{Event: rpctypes.Event_RouteDown, Scopes: []string{routeId}})
	}()
}

// this may return nil (returns default only for empty routeId)
func (router *WshRouter) GetRpc(routeId string) AbstractRpcClient {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	return router.RouteMap[routeId]
}

func (router *WshRouter) SetUpstreamClient(rpc AbstractRpcClient) {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	router.UpstreamClient = rpc
}

func (router *WshRouter) GetUpstreamClient() AbstractRpcClient {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	return router.UpstreamClient
}

func (router *WshRouter) InjectMessage(msgBytes []byte, fromRouteId string) {
	router.InputCh <- msgAndRoute{msgBytes: msgBytes, fromRouteId: fromRouteId}
}

func (router *WshRouter) registerSimpleRequest(reqId string) chan *RpcMessage {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	rtn := make(chan *RpcMessage, 1)
	router.SimpleRequestMap[reqId] = rtn
	return rtn
}

func (router *WshRouter) trySimpleResponse(msg *RpcMessage) bool {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	respCh := router.SimpleRequestMap[msg.ResId]
	if respCh == nil {
		return false
	}
	respCh <- msg
	delete(router.SimpleRequestMap, msg.ResId)
	return true
}

func (router *WshRouter) clearSimpleRequest(reqId string) {
	router.Lock.Lock()
	defer router.Lock.Unlock()
	delete(router.SimpleRequestMap, reqId)
}

func (router *WshRouter) RunSimpleRawCommand(ctx context.Context, msg RpcMessage, fromRouteId string) (*RpcMessage, error) {
	if msg.Command == "" {
		return nil, errors.New("no command")
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var respCh chan *RpcMessage
	if msg.ReqId != "" {
		respCh = router.registerSimpleRequest(msg.ReqId)
	}
	router.InjectMessage(msgBytes, fromRouteId)
	if respCh == nil {
		return nil, nil
	}
	select {
	case <-ctx.Done():
		router.clearSimpleRequest(msg.ReqId)
		return nil, ctx.Err()
	case resp := <-respCh:
		if resp.Error != "" {
			return nil, errors.New(resp.Error)
		}
		return resp, nil
	}
}
