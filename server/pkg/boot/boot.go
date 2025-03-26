package boot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/server/pkg/browsertabs"
	"github.com/outrigdev/outrig/server/pkg/rpcserver"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"github.com/outrigdev/outrig/server/pkg/web"
)

// RunServer initializes and runs the Outrig server
func RunServer() error {
	if serverbase.IsDev() {
		outrigConfig := outrig.DefaultConfig()
		outrigConfig.LogProcessorConfig.OutrigPath = "bin/outrig"
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
