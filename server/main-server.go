package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig/server/pkg/web"
)

const WebServerPort = 5005
const WebSocketPort = 5006

// Packet is the envelope for incoming JSON packets.
type Packet struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// LogLine represents a log message.
type LogLine struct {
	LineNum int64  `json:"linenum"`
	Ts      int64  `json:"ts"`
	Msg     string `json:"msg"`
	Source  string `json:"source,omitempty"`
}

// handleConn reads packets from the connection and prints log packets.
func handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		var pkt Packet
		if err := json.Unmarshal([]byte(line), &pkt); err != nil {
			fmt.Printf("failed to unmarshal packet: %v\n", err)
			continue
		}
		if pkt.Type == "log" {
			var logLine LogLine
			if err := json.Unmarshal(pkt.Data, &logLine); err != nil {
				fmt.Printf("failed to unmarshal log line: %v\n", err)
				continue
			}
			// POC: just print the log line.
			optNewLine := ""
			if !strings.HasSuffix(logLine.Msg, "\n") {
				optNewLine = "\n"
			}
			fmt.Printf("logline: %s %d %s%s", logLine.Source, logLine.LineNum, logLine.Msg, optNewLine)
		} else {
			fmt.Printf("unknown packet type: %s\n", pkt.Type)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("error reading from connection: %v\n", err)
	}
}

func runDomainSocketServer() error {
	outrigPath := utilfn.ExpandHomeDir(base.OutrigHome)
	if err := os.MkdirAll(outrigPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outrigPath, err)
	}

	// Determine the full path for the socket, remove
	socketPath := utilfn.ExpandHomeDir(base.DefaultDomainSocketName)
	_ = os.Remove(socketPath)

	// Listen on the Unix domain socket.
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", socketPath, err)
	}
	fmt.Printf("Server listening on %s\n", socketPath)

	// Accept connections in a loop.
	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Printf("failed to accept connection: %v\n", err)
				continue
			}
			log.Printf("accepted domain socket connection\n")
			go handleConn(conn)
		}
	}()
	return nil
}

func runWebServers() error {
	// Create TCP listener for HTTP server
	httpListener, err := web.MakeTCPListener("http", "127.0.0.1:"+strconv.Itoa(WebServerPort))
	if err != nil {
		return fmt.Errorf("failed to create HTTP listener: %w", err)
	}
	log.Printf("HTTP server listening on %s\n", httpListener.Addr().String())

	// Create TCP listener for WebSocket server
	wsListener, err := web.MakeTCPListener("websocket", "127.0.0.1:"+strconv.Itoa(WebSocketPort))
	if err != nil {
		return fmt.Errorf("failed to create WebSocket listener: %w", err)
	}
	log.Printf("WebSocket server listening on %s\n", wsListener.Addr().String())

	// Run HTTP server
	go web.RunWebServer(httpListener)

	// Run WebSocket server
	go web.RunWebSocketServer(wsListener)

	return nil
}

func main() {
	err := serverbase.EnsureHomeDir()
	if err != nil {
		log.Printf("error cannot create outrig home directory (%s): %v\n", base.OutrigHome, err)
		return
	}
	lock, err := serverbase.AcquireOutrigServerLock()
	if err != nil {
		log.Printf("error acquiring outrig lock (another instance of Outrig Server is likely running): %v\n", err)
		return
	}
	defer lock.Close() // the defer statement will keep the lock alive

	// Run domain socket server
	err = runDomainSocketServer()
	if err != nil {
		log.Printf("Error starting domain socket server: %v\n", err)
		return
	}

	// Run web servers (HTTP and WebSocket)
	err = runWebServers()
	if err != nil {
		log.Printf("Error starting web servers: %v\n", err)
		return
	}

	log.Println("All servers started successfully")
	select {} // Wait forever
}
