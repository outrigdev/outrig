// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package boot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/server/pkg/browsertabs"
	"github.com/outrigdev/outrig/server/pkg/rpc"
	"github.com/outrigdev/outrig/server/pkg/rpcserver"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig/server/pkg/tevent"
	"github.com/outrigdev/outrig/server/pkg/web"
)

// CLIConfig holds configuration options passed from the command line
type CLIConfig struct {
	// Port overrides the default web server port if non-zero
	Port int
}

// RunServer initializes and runs the Outrig server
func RunServer(config CLIConfig) error {
	if serverbase.IsDev() {
		outrigConfig := outrig.DefaultConfig()
		outrigConfig.LogProcessorConfig.OutrigPath = "bin/outrig"
		outrig.Init(outrigConfig)
	}

	// Create a context that we can cancelFn
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	// Create a WaitGroup to track subprocess shutdown
	var wg sync.WaitGroup

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Handle signals in a goroutine
	go func() {
		sig := <-signalChan
		log.Printf("Received signal: %v - Graceful shutdown initiated\n", sig)

		// Perform graceful shutdown
		gracefulShutdown(cancelFn, &wg)

		// Give processes a moment to clean up
		signal.Stop(signalChan)
	}()

	if serverbase.IsDev() {
		log.Printf("Starting Outrig server (dev mode)\n")
	} else {
		log.Printf("Starting Outrig server %s (%s)...\n", serverbase.OutrigServerVersion, serverbase.OutrigCommit)
	}

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

	// Ensure we have a unique server ID
	outrigId, isFirstRun, err := serverbase.EnsureOutrigId()
	if err != nil {
		return fmt.Errorf("error ensuring outrig ID: %w", err)
	}
	// Set the global variables
	serverbase.OutrigId = outrigId
	serverbase.OutrigFirstRun = isFirstRun

	// Send telemetry events
	if isFirstRun {
		// If this is the first run, send an install event
		tevent.SendInstallEvent()
	}
	// Always send a startup event
	tevent.SendStartupEvent()

	// Flush events after startup (asynchronously)
	tevent.UploadEventsAsync()

	outrigRpcServer := rpc.MakeRpcClient(nil, nil, &rpcserver.RpcServerImpl{}, "outrigsrv")
	rpc.GetDefaultRouter().RegisterRoute("outrigsrv", outrigRpcServer, true)
	rpc.InitBroker()

	// Initialize browser tabs tracking
	browsertabs.Initialize()
	log.Printf("Browser tabs tracking initialized\n")

	// Run domain socket server
	err = runDomainSocketServer(ctx)
	if err != nil {
		return fmt.Errorf("error starting domain socket server: %w", err)
	}

	// Run web servers (HTTP and WebSocket)
	err = web.RunWebServer(ctx, config.Port)
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

// gracefulShutdown performs a graceful shutdown of the server
// It sends a shutdown event, flushes telemetry events, and sets a timeout
// after which it will force exit if the server hasn't already shut down
func gracefulShutdown(cancel context.CancelFunc, wg *sync.WaitGroup) {
	// Send shutdown event
	tevent.SendShutdownEvent()

	// Add to WaitGroup before starting the goroutine
	wg.Add(1)

	// Upload telemetry events in a goroutine
	go func() {
		defer wg.Done()

		err := tevent.UploadEvents()
		if err != nil {
			log.Printf("Failed to upload telemetry during shutdown: %v", err)
		}
	}()

	// Cancel the context to stop all processes
	cancel()

	// Set a timeout for shutdown
	go func() {
		// Wait for 5 seconds then force exit if we haven't already
		time.Sleep(5 * time.Second)
		log.Printf("Shutdown timeout reached, forcing exit")
		os.Exit(1)
	}()
}
