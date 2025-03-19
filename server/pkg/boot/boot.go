package boot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
	"github.com/outrigdev/outrig/server/pkg/browsertabs"
	"github.com/outrigdev/outrig/server/pkg/rpcserver"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig/server/pkg/web"
)

// PacketUnmarshalHelper is the envelope for incoming JSON packets.
type PacketUnmarshalHelper struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// handleDomainSocketConn reads packets from the connection and routes them to the appropriate AppRunPeer.
func handleDomainSocketConn(conn net.Conn) {
	var peer *apppeer.AppRunPeer
	var appRunId string

	defer func() {
		conn.Close()
		// If we have a peer, release the reference
		if peer != nil {
			peer.Release()
		}
	}()

	scanner := bufio.NewScanner(conn)
	var isCrashOutput bool
	var crashOutputAppRunId string

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this is a crash output sentinel message
		if !isCrashOutput && strings.HasPrefix(line, "CRASHOUTPUT ") {
			isCrashOutput = true
			crashOutputAppRunId = strings.TrimPrefix(line, "CRASHOUTPUT ")

			// Get the AppRunPeer for this connection
			peer = apppeer.GetAppRunPeer(crashOutputAppRunId)
			if peer == nil {
				log.Printf("Error: No AppRunPeer found for crash output app run ID: %s\n", crashOutputAppRunId)
				return
			}
			log.Printf("Received crash output connection for app run ID: %s\n", crashOutputAppRunId)
			continue
		}

		// If this is a crash output connection, handle the line as crash output
		if isCrashOutput {
			// Create a log line packet
			logLine := &ds.LogLine{
				LineNum: 0, // LineNum will be set by AppRunPeer.HandlePacket
				Ts:      time.Now().UnixMilli(),
				Msg:     line,
				Source:  "crash",
			}
			log.Printf("got #crashoutput line for apprun: %s\n", crashOutputAppRunId)

			// Marshal the log line to JSON
			logData, err := json.Marshal(logLine)
			if err != nil {
				log.Printf("Error marshaling crash output log line: %v\n", err)
				continue
			}

			// Handle the packet
			if err := peer.HandlePacket(ds.PacketTypeLog, logData); err != nil {
				log.Printf("Error handling crash output log line: %v\n", err)
			}
			continue
		}

		// Normal packet handling
		var pkt PacketUnmarshalHelper
		if err := json.Unmarshal([]byte(line), &pkt); err != nil {
			fmt.Printf("failed to unmarshal packet: %v\n", err)
			continue
		}

		// If we haven't identified the app run yet, look for an AppInfo packet
		if peer == nil {
			if pkt.Type == ds.PacketTypeAppInfo {
				var appInfo ds.AppInfo
				if err := json.Unmarshal(pkt.Data, &appInfo); err != nil {
					fmt.Printf("failed to unmarshal AppInfo: %v\n", err)
					continue
				}

				// Get the AppRunId from the AppInfo packet
				appRunId = appInfo.AppRunId
				log.Printf("Identified app run ID: %s\n", appRunId)

				// Get or create the AppRunPeer for this connection
				peer = apppeer.GetAppRunPeer(appRunId)

				// fallthrough to HandlePacket
			} else {
				// Drop packets until we get an AppInfo packet
				log.Printf("Dropping packet of type %s until AppInfo is received\n", pkt.Type)
				continue
			}
		}

		// Route the packet to the AppRunPeer
		if err := peer.HandlePacket(pkt.Type, pkt.Data); err != nil {
			fmt.Printf("error handling packet: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("error reading from connection: %v\n", err)
	}
}

func runDomainSocketServer() error {
	outrigPath := utilfn.ExpandHomeDir(serverbase.GetOutrigHome())
	if err := os.MkdirAll(outrigPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outrigPath, err)
	}

	// Determine the full path for the socket, remove
	socketPath := utilfn.ExpandHomeDir(serverbase.GetDomainSocketName())
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
			go handleDomainSocketConn(conn)
		}
	}()
	return nil
}


// RunServer initializes and runs the Outrig server
func RunServer() error {
	if serverbase.IsDev() {
		outrigConfig := outrig.DefaultConfig()
		outrig.Init(outrigConfig)
	}

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
		return fmt.Errorf("cannot create outrig home directory (%s): %w", serverbase.GetOutrigHome(), err)
	}

	err = serverbase.EnsureDataDir()
	if err != nil {
		return fmt.Errorf("cannot create outrig data directory (%s): %w", serverbase.GetOutrigDataDir(), err)
	}

	lock, err := serverbase.AcquireOutrigServerLock()
	if err != nil {
		return fmt.Errorf("error acquiring outrig lock (another instance of Outrig Server is likely running): %w", err)
	}
	defer lock.Close() // the defer statement will keep the lock alive

	outrigRpcServer := rpc.MakeRpcClient(nil, nil, &rpcserver.RpcServerImpl{}, "outrigsrv")
	rpc.DefaultRouter.RegisterRoute("outrigsrv", outrigRpcServer, true)

	// Initialize browser tabs tracking
	browsertabs.Initialize()
	log.Printf("Browser tabs tracking initialized\n")

	// Run domain socket server
	err = runDomainSocketServer()
	if err != nil {
		return fmt.Errorf("error starting domain socket server: %w", err)
	}

	// Run web servers (HTTP and WebSocket)
	err = web.RunAllWebServers()
	if err != nil {
		return fmt.Errorf("error starting web servers: %w", err)
	}

	log.Printf("All servers started successfully\n")

	// If we're in development mode, start the Vite server
	if serverbase.IsDev() {
		viteCmd, err := startViteServer(ctx)
		if err != nil {
			return fmt.Errorf("error starting Vite server: %w", err)
		}

		// Add to WaitGroup before starting the goroutine
		wg.Add(1)

		// Wait for the Vite server to exit when the context is canceled
		go func() {
			defer wg.Done() // Mark this goroutine as done when it completes

			<-ctx.Done()
			log.Printf("Shutting down Vite server...\n")

			// The context cancellation should already signal the process to stop,
			// but we can also explicitly wait for it to finish
			if err := viteCmd.Wait(); err != nil {
				// Don't report error if it's due to the context being canceled
				if ctx.Err() != context.Canceled {
					log.Printf("Vite server exited with error: %v\n", err)
				}
			}

			log.Printf("Vite server shutdown complete\n")
		}()
	}

	// Wait for context cancellation (from signal handler)
	<-ctx.Done()
	log.Printf("Shutting down server...\n")

	// Wait for all subprocesses to finish shutting down
	log.Printf("Waiting for all processes to complete...\n")
	wg.Wait()
	log.Printf("All processes shutdown complete\n")
	return nil
}
