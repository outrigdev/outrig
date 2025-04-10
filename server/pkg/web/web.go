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
	"os"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
)

// Header constants
const (
	CacheControlHeaderKey     = "Cache-Control"
	CacheControlHeaderNoCache = "no-cache"

	ContentTypeHeaderKey = "Content-Type"
	ContentTypeJson      = "application/json"
	ContentTypeBinary    = "application/octet-stream"

	ContentLengthHeaderKey = "Content-Length"
	LastModifiedHeaderKey  = "Last-Modified"
)

const HttpReadTimeout = 5 * time.Second
const HttpWriteTimeout = 21 * time.Second
const HttpMaxHeaderBytes = 60000
const HttpTimeoutDuration = 21 * time.Second

type WebFnType = func(http.ResponseWriter, *http.Request)

type WebFnOpts struct {
	AllowCaching bool
	JsonErrors   bool
}

func WriteJsonError(w http.ResponseWriter, errVal error) {
	w.Header().Set(ContentTypeHeaderKey, ContentTypeJson)
	w.WriteHeader(http.StatusOK)
	errMap := make(map[string]interface{})
	errMap["error"] = errVal.Error()
	barr, _ := json.Marshal(errMap)
	w.Write(barr)
}

func WriteJsonSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set(ContentTypeHeaderKey, ContentTypeJson)
	rtnMap := make(map[string]interface{})
	rtnMap["success"] = true
	if data != nil {
		rtnMap["data"] = data
	}
	barr, err := json.Marshal(rtnMap)
	if err != nil {
		WriteJsonError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(barr)
}

// Simple health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	WriteJsonSuccess(w, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UnixMilli(),
	})
}

func WebFnWrap(opts WebFnOpts, fn WebFnType) WebFnType {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[web] panic in handler: %v\n", r)
				if opts.JsonErrors {
					WriteJsonError(w, fmt.Errorf("internal server error"))
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}()
		if !opts.AllowCaching {
			w.Header().Set(CacheControlHeaderKey, CacheControlHeaderNoCache)
		}
		fn(w, r)
	}
}

func MakeTCPListener(serviceName string, addr string) (net.Listener, error) {
	if addr == "" {
		addr = "127.0.0.1:0" // Use any available port
	}
	rtn, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error creating listener at %v: %v", addr, err)
	}
	log.Printf("Server [%s] listening on %s\n", serviceName, rtn.Addr())
	return rtn, nil
}

func MakeUnixListener(socketPath string) (net.Listener, error) {
	os.Remove(socketPath) // ignore error
	rtn, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("error creating listener at %v: %v", socketPath, err)
	}
	os.Chmod(socketPath, 0700)
	log.Printf("Server [unix-domain] listening on %s\n", socketPath)
	return rtn, nil
}

// blocking
func RunWebServer(ctx context.Context, listener net.Listener) {
	gr := mux.NewRouter()

	apiRouter := gr.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/health", WebFnWrap(WebFnOpts{AllowCaching: false, JsonErrors: true}, handleHealth))

	// Add more API endpoints here as needed

	fileSystem := GetFileSystem()

	// Handle SPA routing - serve static files or fall back to index.html
	gr.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeIndexOrFile(w, r, fileSystem)
	})

	handler := http.TimeoutHandler(gr, HttpTimeoutDuration, "Timeout")

	if serverbase.IsDev() {
		handler = handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(handler)
	}

	server := &http.Server{
		ReadTimeout:    HttpReadTimeout,
		WriteTimeout:   HttpWriteTimeout,
		MaxHeaderBytes: HttpMaxHeaderBytes,
		Handler:        handler,
	}

	// Create a channel to signal when the server is done
	serverDone := make(chan struct{})

	// Start the server in a goroutine
	go func() {
		log.Printf("HTTP server running on http://%s\n", listener.Addr())
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v\n", err)
		}
		close(serverDone)
	}()

	// Wait for context cancellation or server to finish
	select {
	case <-ctx.Done():
		log.Printf("Shutting down HTTP server...\n")
		// Create a shutdown context with timeout (using 100ms since these are local connections)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer shutdownCancel()
		
		// Attempt graceful shutdown
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v\n", err)
		}
		log.Printf("HTTP server shutdown complete\n")
	case <-serverDone:
		// Server stopped on its own
	}
}

// RunAllWebServers initializes and runs the HTTP and WebSocket servers
func RunAllWebServers(ctx context.Context) error {
	webServerPort := serverbase.GetWebServerPort()
	webSocketPort := serverbase.GetWebSocketPort()

	httpListener, err := MakeTCPListener("http", "127.0.0.1:"+strconv.Itoa(webServerPort))
	if err != nil {
		return fmt.Errorf("failed to create HTTP listener: %w", err)
	}
	log.Printf("HTTP server listening on http://%s\n", httpListener.Addr().String())

	wsListener, err := MakeTCPListener("websocket", "127.0.0.1:"+strconv.Itoa(webSocketPort))
	if err != nil {
		return fmt.Errorf("failed to create WebSocket listener: %w", err)
	}
	log.Printf("WebSocket server listening on ws://%s\n", wsListener.Addr().String())

	go RunWebServer(ctx, httpListener)
	go RunWebSocketServer(ctx, wsListener)

	return nil
}
