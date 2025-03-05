package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

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
	log.Printf("HTTP server listening on http://%s\n", httpListener.Addr().String())

	// Create TCP listener for WebSocket server
	wsListener, err := web.MakeTCPListener("websocket", "127.0.0.1:"+strconv.Itoa(WebSocketPort))
	if err != nil {
		return fmt.Errorf("failed to create WebSocket listener: %w", err)
	}
	log.Printf("WebSocket server listening on ws://%s\n", wsListener.Addr().String())

	// Run HTTP server
	go web.RunWebServer(httpListener)

	// Run WebSocket server
	go web.RunWebSocketServer(wsListener)

	return nil
}

// startViteServer starts the Vite development server as a subprocess
// and pipes its stdout/stderr to the Go server's stdout/stderr.
// It returns a function that can be called to stop the Vite server.
func startViteServer(ctx context.Context) (*exec.Cmd, error) {
	log.Println("Starting Vite development server...")

	// Create the command to run task dev:vite
	cmd := exec.CommandContext(ctx, "task", "dev:vite")

	// Get pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Vite server: %w", err)
	}

	// Copy stdout to our stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println("[vite]", scanner.Text())
		}
	}()

	// Copy stderr to our stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, "[vite]", scanner.Text())
		}
	}()

	log.Println("Vite development server started")
	return cmd, nil
}

func main() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to track subprocess shutdown
	var wg sync.WaitGroup

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Handle signals in a goroutine
	go func() {
		sig := <-signalChan
		log.Printf("Received signal: %v\n", sig)
		cancel() // Cancel the context to stop all processes

		// Give processes a moment to clean up
		signal.Stop(signalChan)
	}()

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

	// If we're in development mode, start the Vite server
	if os.Getenv("OUTRIG_DEV") == "1" {
		viteCmd, err := startViteServer(ctx)
		if err != nil {
			log.Printf("Error starting Vite server: %v\n", err)
			return
		}

		// Add to WaitGroup before starting the goroutine
		wg.Add(1)

		// Wait for the Vite server to exit when the context is canceled
		go func() {
			defer wg.Done() // Mark this goroutine as done when it completes

			<-ctx.Done()
			log.Println("Shutting down Vite server...")

			// The context cancellation should already signal the process to stop,
			// but we can also explicitly wait for it to finish
			if err := viteCmd.Wait(); err != nil {
				// Don't report error if it's due to the context being canceled
				if ctx.Err() != context.Canceled {
					log.Printf("Vite server exited with error: %v\n", err)
				}
			}

			log.Println("Vite server shutdown complete")
		}()
	}

	// Wait for context cancellation (from signal handler)
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Wait for all subprocesses to finish shutting down
	log.Println("Waiting for all processes to complete...")
	wg.Wait()
	log.Println("All processes shutdown complete")
}
