// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package boot

import (
	"context"
	"log"
	"net"
	"sync/atomic"

	"github.com/outrigdev/outrig"
)

// runTCPAcceptLoop handles incoming TCP connections from the multiplexer
func runTCPAcceptLoop(ctx context.Context, tcpListener net.Listener, webServerPort int) error {
	go func() {
		outrig.SetGoRoutineName("boot.TCPAcceptLoop.Waiter")
		var shutdown atomic.Bool

		defer func() {
			tcpListener.Close()
			log.Printf("TCP accept loop shutdown complete\n")
		}()

		// Create a channel to signal when Accept() returns
		acceptDone := make(chan struct{})

		// Start a goroutine to accept connections
		go func() {
			outrig.SetGoRoutineName("TCPAcceptLoop")
			for {
				conn, err := tcpListener.Accept()
				if err != nil {
					// Only log errors if we haven't initiated shutdown
					if !shutdown.Load() {
						log.Printf("failed to accept TCP connection: %v\n", err)
					}
					close(acceptDone)
					return
				}
				go func() {
					outrig.SetGoRoutineName("TCPAcceptLoop.HandleConn")
					handleDomainSocketConn(conn, webServerPort)
				}()
			}
		}()

		// Wait for either context cancellation or accept to finish
		select {
		case <-ctx.Done():
			log.Printf("Shutting down TCP accept loop...\n")
			shutdown.Store(true)
			tcpListener.Close() // This will cause Accept() to return with an error
		case <-acceptDone:
			// Accept returned on its own (likely due to an error)
		}
	}()
	return nil
}