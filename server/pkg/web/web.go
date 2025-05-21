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
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
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

// TrayAppRunInfo contains minimal app run information for the tray app
type TrayAppRunInfo struct {
	AppRunId  string `json:"apprunid"`
	AppName   string `json:"appname"`
	IsRunning bool   `json:"isrunning"`
	StartTime int64  `json:"starttime"`
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

// Status endpoint for checking server status
func handleStatus(w http.ResponseWriter, r *http.Request) {
	// Get all app run infos (passing 0 to get all regardless of modification time)
	appRunInfos := apppeer.GetAllAppRunPeerInfos(0)
	
	// Check if there are any active connections
	hasConnections := false
	trayAppRuns := []TrayAppRunInfo{}
	
	for _, info := range appRunInfos {
		// If any app is running, we have active connections
		if info.IsRunning {
			hasConnections = true
		}
		
		// Add app run info to the array
		trayAppRuns = append(trayAppRuns, TrayAppRunInfo{
			AppRunId:  info.AppRunId,
			AppName:   info.AppName,
			IsRunning: info.IsRunning,
			StartTime: info.StartTime,
		})
	}
	
	WriteJsonSuccess(w, map[string]interface{}{
		"status":         "ok",
		"time":           time.Now().UnixMilli(),
		"hasconnections": hasConnections,
		"appruns":        trayAppRuns,
		"version":        serverbase.OutrigServerVersion,
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
	return rtn, nil
}

// blocking
func runWebServerInternal(ctx context.Context, listener net.Listener) {
	gr := mux.NewRouter()

	// WebSocket endpoint - this will be handled separately to avoid the timeout handler
	gr.HandleFunc("/ws", HandleWs)

	apiRouter := gr.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/health", WebFnWrap(WebFnOpts{AllowCaching: false, JsonErrors: true}, handleHealth))
	apiRouter.HandleFunc("/status", WebFnWrap(WebFnOpts{AllowCaching: false, JsonErrors: true}, handleStatus))

	// Add more API endpoints here as needed

	fileSystem := GetFileSystem()

	// Handle SPA routing - serve static files or fall back to index.html
	gr.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeIndexOrFile(w, r, fileSystem)
	})

	// Create a special handler that bypasses the timeout handler for WebSocket requests
	// This is necessary because the timeout handler doesn't implement http.Hijacker
	// which is required for WebSocket connections
	specialHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			// WebSocket requests bypass the timeout handler
			gr.ServeHTTP(w, r)
		} else {
			// All other requests go through the timeout handler
			timeoutHandler := http.TimeoutHandler(gr, HttpTimeoutDuration, "Timeout")
			timeoutHandler.ServeHTTP(w, r)
		}
	})

	// Final handler with CORS if in dev mode
	var handler http.Handler = specialHandler
	if serverbase.IsDev() {
		handler = handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(specialHandler)
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
		outrig.SetGoRoutineName("WebServer")
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

// RunWebServer initializes and runs the HTTP server (which also handles WebSockets)
// If overridePort is non-zero, it will be used instead of the default port
// Returns the port on which the server is running
func RunWebServer(ctx context.Context, overridePort int) (int, error) {
	webServerPort := serverbase.GetWebServerPort()
	if overridePort > 0 {
		webServerPort = overridePort
	}

	httpListener, err := MakeTCPListener("http", "127.0.0.1:"+strconv.Itoa(webServerPort))
	if err != nil {
		return 0, fmt.Errorf("failed to create HTTP listener: %w", err)
	}
	log.Printf("Outrig server running on http://%s\n", httpListener.Addr().String())
	go func() {
		outrig.SetGoRoutineName("WebServer.Waiter")
		runWebServerInternal(ctx, httpListener)
	}()
	return webServerPort, nil
}
