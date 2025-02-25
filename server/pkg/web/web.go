// Copyright 2025, Outrig Inc.

package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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
func RunWebServer(listener net.Listener) {
	gr := mux.NewRouter()
	gr.HandleFunc("/health", WebFnWrap(WebFnOpts{AllowCaching: false, JsonErrors: true}, handleHealth))
	
	// Add more endpoints here as needed
	
	handler := http.TimeoutHandler(gr, HttpTimeoutDuration, "Timeout")
	
	// In development mode, enable CORS
	isDev := os.Getenv("OUTRIG_DEV") == "1"
	if isDev {
		handler = handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(handler)
	}
	
	server := &http.Server{
		ReadTimeout:    HttpReadTimeout,
		WriteTimeout:   HttpWriteTimeout,
		MaxHeaderBytes: HttpMaxHeaderBytes,
		Handler:        handler,
	}
	err := server.Serve(listener)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
	}
}
