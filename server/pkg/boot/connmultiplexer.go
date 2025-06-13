// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package boot

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig"
)

const (
	// PeekBufferSize is the number of bytes to peek for protocol identification
	PeekBufferSize = 8
)

// bufferedConn wraps a net.Conn with a bufio.Reader to handle peeked bytes
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

// Read implements net.Conn.Read, reading from the buffered reader first
func (bc *bufferedConn) Read(b []byte) (int, error) {
	return bc.reader.Read(b)
}

// channelListener implements net.Listener using a channel for connections
type channelListener struct {
	connChan chan net.Conn
	addr     net.Addr
	closed   bool
	mu       sync.Mutex
}

// Accept implements net.Listener.Accept
func (cl *channelListener) Accept() (net.Conn, error) {
	conn, ok := <-cl.connChan
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return conn, nil
}

// Close implements net.Listener.Close
func (cl *channelListener) Close() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if !cl.closed {
		cl.closed = true
		close(cl.connChan)
	}
	return nil
}

// Addr implements net.Listener.Addr
func (cl *channelListener) Addr() net.Addr {
	return cl.addr
}

// addConn adds a connection to the channel listener
func (cl *channelListener) addConn(conn net.Conn) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.closed {
		return fmt.Errorf("listener is closed")
	}

	select {
	case cl.connChan <- conn:
		return nil
	default:
		return fmt.Errorf("connection channel is full")
	}
}

// NewChannelListener creates a new channel-based listener
func NewChannelListener(addr net.Addr) *channelListener {
	return &channelListener{
		connChan: make(chan net.Conn, 10),
		addr:     addr,
	}
}

// MultiplexerListeners holds the two listeners returned by MakeMultiplexerListener
type MultiplexerListeners struct {
	HTTPListener net.Listener
	TCPListener  net.Listener
}

// MakeMultiplexerListener creates a multiplexer that listens on a single address
// and returns separate listeners for HTTP and custom TCP protocols
func MakeMultiplexerListener(ctx context.Context, addr string) (*MultiplexerListeners, error) {
	// Create the main TCP listener
	mainListener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create main listener: %w", err)
	}

	// Create channel listeners for both HTTP and TCP connections
	httpListener := NewChannelListener(mainListener.Addr())
	tcpListener := NewChannelListener(mainListener.Addr())

	// Start the multiplexer goroutine
	outrig.Go("multiplexer.acceptmonitor").Run(func() {
		var shutdown atomic.Bool

		defer func() {
			mainListener.Close()
			httpListener.Close()
			tcpListener.Close()
		}()

		// Create a channel to signal when Accept() returns
		acceptDone := make(chan struct{})

		// Start a goroutine to accept connections
		outrig.Go("multiplexer.acceptloop").Run(func() {
			for {
				conn, err := mainListener.Accept()
				if err != nil {
					// Only log errors if we haven't initiated shutdown
					if !shutdown.Load() {
						log.Printf("failed to accept connection: %v\n", err)
					}
					close(acceptDone)
					return
				}

				// Handle the connection in a separate goroutine
				go handleMultiplexedConnection(conn, httpListener, tcpListener)
			}
		})

		// Wait for either context cancellation or accept to finish
		select {
		case <-ctx.Done():
			shutdown.Store(true)
			mainListener.Close() // This will cause Accept() to return with an error
		case <-acceptDone:
			// Accept returned on its own (likely due to an error)
		}
	})

	return &MultiplexerListeners{
		HTTPListener: httpListener,
		TCPListener:  tcpListener,
	}, nil
}

// handleMultiplexedConnection identifies the protocol and routes the connection
func handleMultiplexedConnection(conn net.Conn, httpListener *channelListener, tcpListener *channelListener) {
	// Set a read deadline for protocol identification
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Create a buffered reader to peek at the first few bytes
	reader := bufio.NewReader(conn)

	// Peek at the first few bytes to identify the protocol
	peekedBytes, err := reader.Peek(PeekBufferSize)
	if err != nil && err != io.EOF {
		conn.Close()
		return
	}

	// Clear the read deadline after peeking
	conn.SetReadDeadline(time.Time{})

	// Create a buffered connection that preserves the peeked bytes
	bufferedConn := &bufferedConn{
		Conn:   conn,
		reader: reader,
	}

	// Identify the protocol based on the peeked bytes
	if isHTTPProtocol(peekedBytes) {
		// Route to HTTP listener
		if err := httpListener.addConn(bufferedConn); err != nil {
			conn.Close()
		}
	} else {
		// Route to custom TCP listener
		if err := tcpListener.addConn(bufferedConn); err != nil {
			conn.Close()
		}
	}
}

// isHTTPProtocol checks if the peeked bytes indicate an HTTP request
func isHTTPProtocol(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Convert to string for easier matching
	str := string(data)

	// Check for common HTTP methods
	httpMethods := []string{
		"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH ", "TRACE ", "CONNECT ",
	}

	for _, method := range httpMethods {
		if strings.HasPrefix(str, method) {
			return true
		}
	}

	return false
}
