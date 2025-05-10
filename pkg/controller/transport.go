// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
)

const TransportPeerBufferSize = 100
const WriteDeadline = 10 * time.Second // this is very high, just helps to clear out hung connections, not for real flow control

// transportPeer wraps a comm.ConnWrap with a buffered channel for packet sending
type transportPeer struct {
	Conn   *comm.ConnWrap
	SendCh chan string
}

// Transport handles connection management and packet sending functionality
type Transport struct {
	lock                    sync.Mutex
	connMap                 map[string]*transportPeer // map of connections by peer name
	config                  *ds.Config
	TransportPacketsSent    int64
	TransportDroppedPackets int64
}

// MakeTransport creates a new Transport instance
func MakeTransport(config *ds.Config) *Transport {
	return &Transport{
		connMap: make(map[string]*transportPeer),
		config:  config,
	}
}

// makeTransportPeer creates a new TransportPeer instance
func makeTransportPeer(conn *comm.ConnWrap) *transportPeer {
	return &transportPeer{
		Conn:   conn,
		SendCh: make(chan string, TransportPeerBufferSize),
	}
}

// IsConnected returns true if there are any active connections
func (t *Transport) HasConnections() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return len(t.connMap) > 0
}

// startPeerLoop starts a goroutine to process packets for a TransportPeer
func (t *Transport) startPeerLoop(peer *transportPeer) {
	go func() {
		for jsonStr := range peer.SendCh {
			peer.Conn.Conn.SetWriteDeadline(time.Now().Add(WriteDeadline))
			err := peer.Conn.WriteLine(jsonStr)
			if err != nil {
				t.closeConn(peer, err)
				return
			}
		}
	}()
}

// AddConn adds a connection to the connection map
func (t *Transport) AddConn(conn *comm.ConnWrap) {
	if conn == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	existingPeer := t.connMap[conn.PeerName]
	if existingPeer != nil {
		t.closeConn_nolock(existingPeer, nil)
	}

	peer := makeTransportPeer(conn)
	t.connMap[conn.PeerName] = peer
	t.startPeerLoop(peer)
}

// closeConn_nolock closes a connection and removes it from the connection map
// Caller must hold the lock
func (t *Transport) closeConn_nolock(peer *transportPeer, err error) {
	if peer == nil || peer.Conn == nil {
		return
	}
	_, found := t.connMap[peer.Conn.PeerName]
	if !found {
		// if not found, then we already closed everything, avoids double close errors
		return
	}

	if !t.config.Quiet {
		if err != nil {
			fmt.Printf("[outrig] disconnecting from %s: %v\n", peer.Conn.PeerName, err)
		} else {
			fmt.Printf("[outrig] disconnecting from %s\n", peer.Conn.PeerName)
		}
	}
	// Close the channel to stop the goroutine
	close(peer.SendCh)

	// Close the connection
	peer.Conn.Close()
	delete(t.connMap, peer.Conn.PeerName)
}

func (t *Transport) closeConn(peer *transportPeer, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.closeConn_nolock(peer, err)
}

// CloseAllConns closes all connections
func (t *Transport) CloseAllConns() {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, peer := range t.connMap {
		t.closeConn_nolock(peer, nil)
	}
}

// SendPacketInternal sends a packet to all available connections
// This is an internal method that doesn't check if Outrig is enabled
func (t *Transport) sendPacketInternal(pk *ds.PacketType) (bool, error) {
	barr, err := json.Marshal(pk)
	if err != nil {
		return false, err
	}
	jsonStr := string(barr)

	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.connMap) == 0 {
		return false, nil
	}

	sentToAny := false
	for _, peer := range t.connMap {
		// Try to send to channel non-blocking
		select {
		case peer.SendCh <- jsonStr:
			sentToAny = true
			atomic.AddInt64(&t.TransportPacketsSent, 1)
		default:
			// Channel is full, increment dropped packets counter
			atomic.AddInt64(&t.TransportDroppedPackets, 1)
		}
	}
	return sentToAny, nil
}

// SendPacket sends a packet if Outrig is enabled
func (t *Transport) SendPacket(pk *ds.PacketType, force bool) (bool, error) {
	if !force && !global.OutrigEnabled.Load() {
		return false, nil
	}
	return t.sendPacketInternal(pk)
}

// Note: This implementation uses the printf and isStdoutATerminal functions
// from controller.go to avoid redeclaration issues
