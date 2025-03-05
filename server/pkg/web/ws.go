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
	"github.com/outrigdev/outrig/pkg/utilds"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const wsReadWaitTimeout = 15 * time.Second
const wsWriteWaitTimeout = 10 * time.Second
const wsPingPeriodTickTime = 10 * time.Second
const wsInitialPingTime = 1 * time.Second

var ConnMap = utilds.MakeSyncMap[*WebSocketModel]()

type WSEventType struct {
	Type string `json:"type"`
	Ts   int64  `json:"ts"`
	Data any    `json:"data,omitempty"`
}

type WebSocketModel struct {
	ConnId   string
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

func processMessage(event WSEventType, outputCh chan WSEventType) {
	// Process incoming messages here
	if event.Type == "" {
		return
	}
	log.Printf("[websocket] processing message: %v\n", event.Type)
	// TODO process message
}

func ReadLoop(conn *websocket.Conn, outputCh chan WSEventType, closeCh chan any, connId string) {
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
		if event.Type == "pong" {
			// nothing
			continue
		}
		if event.Type == "ping" {
			now := time.Now()
			pongMessage := WSEventType{Type: "pong", Ts: now.UnixMilli()}
			outputCh <- pongMessage
			continue
		}
		go processMessage(event, outputCh)
	}
}

func WritePing(conn *websocket.Conn) error {
	now := time.Now()
	pingMessage := map[string]interface{}{"type": "ping", "ts": now.UnixMilli()}
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

	connId := uuid.New().String()
	outputCh := make(chan WSEventType, 100)
	closeCh := make(chan any)

	log.Printf("[websocket] new connection: connid:%s\n", connId)

	wsModel := &WebSocketModel{
		ConnId:   connId,
		Conn:     conn,
		OutputCh: outputCh,
	}

	ConnMap.Set(connId, wsModel)
	defer func() {
		ConnMap.Delete(connId)
		time.Sleep(1 * time.Second)
		close(outputCh)
	}()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		// read loop
		defer wg.Done()
		ReadLoop(conn, outputCh, closeCh, connId)
	}()

	go func() {
		// write loop
		defer wg.Done()
		WriteLoop(conn, outputCh, closeCh, connId)
	}()

	wg.Wait()
	return nil
}
