// Copyright 2025, Outrig Inc.

package web

import (
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
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const wsReadWaitTimeout = 15 * time.Second
const wsWriteWaitTimeout = 10 * time.Second
const wsPingPeriodTickTime = 10 * time.Second
const wsInitialPingTime = 1 * time.Second

const EventType_Rpc = "rpc"
const EventType_Ping = "ping"
const EventType_Pong = "pong"

var ConnMap = utilds.MakeSyncMap[*WebSocketModel]()

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

func RunWebSocketServer(listener net.Listener) {
	gr := mux.NewRouter()
	gr.HandleFunc("/ws", HandleWs)
	server := &http.Server{
		ReadTimeout:    HttpReadTimeout,
		WriteTimeout:   HttpWriteTimeout,
		MaxHeaderBytes: HttpMaxHeaderBytes,
		Handler:        gr,
	}
	server.SetKeepAlivesEnabled(false)
	log.Printf("[websocket] running websocket server on %s\n", listener.Addr())
	err := server.Serve(listener)
	if err != nil {
		log.Printf("[websocket] error trying to run websocket server: %v\n", err)
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
	defer utilfn.GoDrainChan(outputCh)
	initialPing := true
	for {
		select {
		case msg := <-outputCh:
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

	wproxy := rpc.MakeRpcProxy()
	rpc.DefaultRouter.RegisterRoute(routeId, wproxy, true)
	defer rpc.DefaultRouter.UnregisterRoute(routeId)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		// read loop
		defer wg.Done()
		ReadLoop(conn, outputCh, closeCh, connId, wproxy.FromRemoteCh)
	}()

	go func() {
		// write loop
		defer wg.Done()
		WriteLoop(conn, outputCh, closeCh, connId)
	}()

	wg.Wait()
	return nil
}
