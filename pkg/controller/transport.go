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
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
)

// Global counters for transport statistics
var TransportPacketsQueued int64
var TransportDroppedPackets int64

const TransportPeerBufferSize = 100
const WriteDeadline = 10 * time.Second // this is very high, just helps to clear out hung connections, not for real flow control
const LogBatchSize = 100

// packetWrap wraps a packet for sending, with special handling for multilog packets
type packetWrap struct {
	RawPacket string
	MultiLog  bool
	LogLines  *[]ds.LogLine
}

// transportPeer wraps a comm.ConnWrap with a buffered channel for packet sending
type transportPeer struct {
	Conn         *comm.ConnWrap
	SendCh       chan packetWrap
	multiLogLock sync.Mutex
	logLines     *[]ds.LogLine
}

// Transport handles connection management and packet sending functionality
type Transport struct {
	lock    sync.Mutex
	connMap map[string]*transportPeer // map of connections by peer name
	config  *config.Config
}

// MakeTransport creates a new Transport instance
func MakeTransport(cfg *config.Config) *Transport {
	return &Transport{
		connMap: make(map[string]*transportPeer),
		config:  cfg,
	}
}

// makeTransportPeer creates a new TransportPeer instance
func makeTransportPeer(conn *comm.ConnWrap) *transportPeer {
	return &transportPeer{
		Conn:     conn,
		SendCh:   make(chan packetWrap, TransportPeerBufferSize),
		logLines: nil,
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
		ioutrig.I.SetGoRoutineName("#outrig TransportPeerLoop")
		for packet := range peer.SendCh {
			peer.Conn.Conn.SetWriteDeadline(time.Now().Add(WriteDeadline))

			var jsonStr string
			var err error

			if packet.MultiLog {
				// For multilog packets, marshal the packet just before sending
				jsonStr, err = peer.marshalMultiLogPacket(packet.LogLines)
				if err != nil {
					// If there's an error marshaling, skip this packet
					continue
				}
			} else {
				// For regular packets, just use the pre-marshaled JSON
				jsonStr = packet.RawPacket
			}

			err = peer.Conn.WriteLine(jsonStr)
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

// marshalMultiLogPacket marshals a multilog packet to JSON
func (p *transportPeer) marshalMultiLogPacket(logLines *[]ds.LogLine) (string, error) {
	p.multiLogLock.Lock()
	defer p.multiLogLock.Unlock()

	// Create the multilog packet
	multiLogPacket := &ds.PacketType{
		Type: ds.PacketTypeMultiLog,
		Data: &ds.MultiLogLines{
			LogLines: *logLines,
		},
	}

	// Marshal the packet
	barr, err := json.Marshal(multiLogPacket)
	if err != nil {
		return "", err
	}

	// If this is our current logLines, clear it
	if logLines == p.logLines {
		p.logLines = nil
	}

	return string(barr), nil
}

// addLogLine adds a log line from a packet to the peer's multilog packet
// Returns true if the log line was successfully added
func (p *transportPeer) addLogLine(pk *ds.PacketType) bool {
	var logData ds.LogLine

	// Handle both ds.LogLine and *ds.LogLine cases
	if data, ok := pk.Data.(ds.LogLine); ok {
		// Case 1: pk.Data is a ds.LogLine value
		logData = data
	} else if ptrData, ok := pk.Data.(*ds.LogLine); ok {
		// Case 2: pk.Data is a *ds.LogLine pointer
		if ptrData == nil {
			return false
		}
		logData = *ptrData
	} else {
		// Neither a LogLine nor a *LogLine
		return false
	}

	p.multiLogLock.Lock()
	defer p.multiLogLock.Unlock()

	// If we don't have a multilog packet in the queue yet or past buffer size, create one
	if p.logLines == nil || len(*p.logLines) >= LogBatchSize {
		// Create a new log lines slice
		logLines := make([]ds.LogLine, 0, LogBatchSize)
		logLines = append(logLines, logData)

		// Create the packet wrap
		packet := packetWrap{
			MultiLog: true,
			LogLines: &logLines,
		}

		// Store the log lines pointer
		p.logLines = &logLines

		// Send the packet to the channel
		sent := sendNonBlock(p.SendCh, packet)
		if !sent {
			// Channel is full, clear the log lines
			p.logLines = nil
		}
		return sent
	}

	// We already have a multilog packet in the queue, just append the log line
	*p.logLines = append(*p.logLines, logData)
	return true
}

// SendPacketInternal sends a packet to all available connections
// This is an internal method that doesn't check if Outrig is enabled
func (t *Transport) sendPacketInternal(pk *ds.PacketType) (bool, error) {
	isLogPacket := pk.Type == ds.PacketTypeLog

	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.connMap) == 0 {
		return false, nil
	}

	sentToAny := false
	for _, peer := range t.connMap {
		if isLogPacket {
			// For log packets, just add to the peer's multilog packet
			if peer.addLogLine(pk) {
				sentToAny = true
			}
		} else {
			// For non-log packets, marshal and send directly
			barr, err := json.Marshal(pk)
			if err != nil {
				continue
			}

			packet := packetWrap{
				RawPacket: string(barr),
				MultiLog:  false,
				LogLines:  nil,
			}

			if sendNonBlock(peer.SendCh, packet) {
				sentToAny = true
			}
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

// sendNonBlock attempts to send a packet to a channel without blocking
// Returns true if the send was successful, false if the channel is full
// Also updates the global packet counters
func sendNonBlock(ch chan packetWrap, packet packetWrap) bool {
	select {
	case ch <- packet:
		atomic.AddInt64(&TransportPacketsQueued, 1)
		return true
	default:
		atomic.AddInt64(&TransportDroppedPackets, 1)
		return false
	}
}
