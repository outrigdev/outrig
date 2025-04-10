// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// WSInfo contains information about a WebSocket connection
type WSInfo struct {
	ConnId  string `json:"connid"`
	RouteId string `json:"routeid"`
}

func init() {
	// Register a watch function that returns a map of connection ID to WSInfo
	outrig.WatchFunc("websockets", func() map[string]WSInfo {
		return GetAllWSInfo()
	}, nil)
}

// GetAllWSInfo returns a map of connection ID to WSInfo for all WebSocket connections
func GetAllWSInfo() map[string]WSInfo {
	keys := ConnMap.Keys()
	result := make(map[string]WSInfo, len(keys))

	for _, key := range keys {
		wsModel := ConnMap.Get(key)
		if wsModel != nil {
			info := WSInfo{
				ConnId:  wsModel.ConnId,
				RouteId: wsModel.RouteId,
			}
			result[key] = info
		}
	}

	return result
}

const wsReadWaitTimeout = 15 * time.Second
const wsWriteWaitTimeout = 10 * time.Second
const wsPingPeriodTickTime = 10 * time.Second
const wsInitialPingTime = 1 * time.Second

const EventType_Rpc = "rpc"
const EventType_Ping = "ping"
const EventType_Pong = "pong"

var ConnMap = utilds.MakeSyncMap[string, *WebSocketModel]()

type WSEventType struct {
	Type string `json:"type"`
	Ts   int64  `json:"ts"`
	Data any    `json:"data,omitempty"`
}

type WebSocketModel struct {
	ConnId   string
	RouteId  string
	Conn     *websocket.Conn
	OutputCh chan WSEventType
}

func RunWebSocketServer(ctx context.Context, listener net.Listener) {
	gr := mux.NewRouter()
	gr.HandleFunc("/ws", HandleWs)
	server := &http.Server{
		ReadTimeout:    HttpReadTimeout,
		WriteTimeout:   HttpWriteTimeout,
		MaxHeaderBytes: HttpMaxHeaderBytes,
		Handler:        gr,
	}
	server.SetKeepAlivesEnabled(false)
	
	// Create a channel to signal when the server is done
	serverDone := make(chan struct{})
	
	// Start the server in a goroutine
	go func() {
		log.Printf("[websocket] running websocket server on %s\n", listener.Addr())
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("[websocket] error trying to run websocket server: %v\n", err)
		}
		close(serverDone)
	}()
	
	// Wait for context cancellation or server to finish
	select {
	case <-ctx.Done():
		log.Printf("Shutting down WebSocket server...\n")
		// Create a shutdown context with timeout (using 100ms since these are local connections)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer shutdownCancel()
		
		// Attempt graceful shutdown
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("WebSocket server shutdown error: %v\n", err)
		}
		log.Printf("WebSocket server shutdown complete\n")
	case <-serverDone:
		// Server stopped on its own
	}
}

var WebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:   4 * 1024,
	WriteBufferSize:  32 * 1024,
	HandshakeTimeout: 1 * time.Second,
	CheckOrigin:      func(r *http.Request) bool { return true },
}

func HandleWs(w http.ResponseWriter, r *http.Request) {
	err := HandleWsInternal(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func processMessage(event WSEventType, rpcCh chan []byte) {
	// Process incoming messages here
	if event.Type == "" {
		return
	}
	if event.Type == EventType_Rpc {
		rpcMsg := event.Data
		msgBytes, err := json.Marshal(rpcMsg)
		if err != nil {
			log.Printf("[websocket] error marshalling rpc message: %v\n", err)
			return
		}
		rpcCh <- msgBytes
		return
	}
	log.Printf("[websocket] invalid message type: %s\n", event.Type)
}

func ReadLoop(conn *websocket.Conn, outputCh chan WSEventType, closeCh chan any, connId string, rpcCh chan []byte) {
	readWait := wsReadWaitTimeout
	conn.SetReadLimit(64 * 1024)
	conn.SetReadDeadline(time.Now().Add(readWait))
	defer close(closeCh)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[websocket] ReadPump error (%s): %v\n", connId, err)
			break
		}
		var event WSEventType
		err = json.Unmarshal(message, &event)
		if err != nil {
			log.Printf("[websocket] error unmarshalling json: %v\n", err)
			break
		}
		conn.SetReadDeadline(time.Now().Add(readWait))
		if event.Type == EventType_Pong {
			// nothing
			continue
		}
		if event.Type == EventType_Ping {
			now := time.Now()
			pongMessage := WSEventType{Type: EventType_Pong, Ts: now.UnixMilli()}
			outputCh <- pongMessage
			continue
		}
		go processMessage(event, rpcCh)
	}
}

func WritePing(conn *websocket.Conn) error {
	now := time.Now()
	pingMessage := map[string]interface{}{"type": EventType_Ping, "ts": now.UnixMilli()}
	jsonVal, _ := json.Marshal(pingMessage)
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWaitTimeout)) // no error
	err := conn.WriteMessage(websocket.TextMessage, jsonVal)
	if err != nil {
		return err
	}
	return nil
}

func WriteLoop(conn *websocket.Conn, outputCh chan WSEventType, closeCh chan any, connId string) {
	ticker := time.NewTicker(wsInitialPingTime)
	defer ticker.Stop()
	defer func() {
		go func() {
			ioutrig.I.SetGoRoutineName("#outrig ws:WriteLoop:DrainChan")
			utilfn.DrainChan(outputCh)
		}()
	}()
	initialPing := true
	for {
		select {
		case msg, ok := <-outputCh:
			if !ok {
				return
			}
			barr, err := json.Marshal(msg)
			if err != nil {
				log.Printf("[websocket] cannot marshal websocket message: %v\n", err)
				// just loop again
				break
			}
			err = conn.WriteMessage(websocket.TextMessage, barr)
			if err != nil {
				conn.Close()
				log.Printf("[websocket] WritePump error (%s): %v\n", connId, err)
				return
			}

		case <-ticker.C:
			err := WritePing(conn)
			if err != nil {
				log.Printf("[websocket] WritePump error (%s): %v\n", connId, err)
				return
			}
			if initialPing {
				initialPing = false
				ticker.Reset(wsPingPeriodTickTime)
			}

		case <-closeCh:
			return
		}
	}
}

func HandleWsInternal(w http.ResponseWriter, r *http.Request) error {
	conn, err := WebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("WebSocket Upgrade Failed: %v", err)
	}
	defer conn.Close()

	routeId := r.URL.Query().Get("routeid")
	if routeId == "" {
		return fmt.Errorf("routeid not provided")
	}
	connId := uuid.New().String()
	outputCh := make(chan WSEventType, 100)
	closeCh := make(chan any)

	log.Printf("[websocket] new connection: connid:%s, routeid:%s\n", connId, routeId)
	wsModel := &WebSocketModel{
		ConnId:   connId,
		RouteId:  routeId,
		Conn:     conn,
		OutputCh: outputCh,
	}
	ConnMap.Set(connId, wsModel)
	defer func() {
		ConnMap.Delete(connId)
		time.Sleep(1 * time.Second)
		close(outputCh)
	}()

	proxy := rpc.MakeRpcProxy()
	rpc.DefaultRouter.RegisterRoute(routeId, proxy, true)
	defer rpc.DefaultRouter.UnregisterRoute(routeId)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		for msg := range proxy.ToRemoteCh {
			rawMsg := json.RawMessage(msg)
			outputCh <- WSEventType{Type: EventType_Rpc, Ts: time.Now().UnixMilli(), Data: rawMsg}
		}
	}()

	go func() {
		// read loop
		defer wg.Done()
		ReadLoop(conn, outputCh, closeCh, connId, proxy.FromRemoteCh)
	}()

	go func() {
		// write loop
		defer wg.Done()
		WriteLoop(conn, outputCh, closeCh, connId)
	}()

	wg.Wait()
	return nil
}
