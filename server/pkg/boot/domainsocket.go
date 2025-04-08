// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package boot

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/comm"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
)

// PacketUnmarshalHelper is the envelope for incoming JSON packets.
type PacketUnmarshalHelper struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// handleLogMode handles a connection in log mode
// The source is set to the part after "log:" in the mode line
// Unlike other modes, lines are not trimmed
func handleLogMode(connWrap *comm.ConnWrap, appRunId string, source string) {
	peer := apppeer.GetAppRunPeer(appRunId, true)
	if peer == nil {
		log.Printf("Error: No AppRunPeer found for log mode app run ID: %s\n", appRunId)
		return
	}
	log.Printf("Received log connection for app run ID: %s with source: %s\n", appRunId, source)

	defer peer.Release()

	// Use the ConnWrap to read lines
	for {
		line, err := connWrap.ReadLine()
		if err != nil {
			fmt.Printf("error reading from log connection: %v\n", err)
			break
		}
		line = strings.TrimSuffix(line, "\n")

		// Create a log line packet (note: we don't trim the line)
		logLine := &ds.LogLine{
			LineNum: 0, // LineNum will be set by AppRunPeer.HandlePacket
			Ts:      time.Now().UnixMilli(),
			Msg:     line,
			Source:  source,
		}

		// Marshal the log line to JSON
		logData, err := json.Marshal(logLine)
		if err != nil {
			log.Printf("Error marshaling log line: %v\n", err)
			continue
		}

		// Handle the packet
		if err := peer.HandlePacket(ds.PacketTypeLog, logData); err != nil {
			log.Printf("Error handling log line: %v\n", err)
		}
	}
}

// handlePacketMode handles a connection in packet mode
func handlePacketMode(connWrap *comm.ConnWrap, appRunId string) {
	// Get the AppRunPeer for this connection
	peer := apppeer.GetAppRunPeer(appRunId, true)
	if peer == nil {
		log.Printf("Error: No AppRunPeer found for app run ID: %s\n", appRunId)
		return
	}
	log.Printf("Using AppRunPeer for app run ID: %s\n", appRunId)

	defer peer.Release()

	// Use the ConnWrap to read lines
	for {
		line, err := connWrap.ReadLine()
		if err != nil {
			fmt.Printf("error reading from packet connection: %v\n", err)
			break
		}

		line = strings.TrimSpace(line)

		// Normal packet handling
		var pkt PacketUnmarshalHelper
		if err := json.Unmarshal([]byte(line), &pkt); err != nil {
			fmt.Printf("failed to unmarshal packet: %v\n", err)
			continue
		}

		// Route the packet to the AppRunPeer
		if err := peer.HandlePacket(pkt.Type, pkt.Data); err != nil {
			fmt.Printf("error handling packet: %v\n", err)
		}
	}
}

// handleDomainSocketConn reads the mode line from the connection and dispatches to the appropriate handler.
func handleDomainSocketConn(conn net.Conn) {
	defer conn.Close()

	// Create a ConnWrap for the connection
	connWrap := comm.MakeConnWrap(conn, "domain-socket-client")

	// Perform the handshake
	mode, submode, appRunId, err := connWrap.ServerHandshake()
	if err != nil {
		log.Printf("Handshake failed: %v\n", err)
		return
	}

	log.Printf("Connection mode: %s, submode: %s, app run ID: %s\n", mode, submode, appRunId)

	// Dispatch to the appropriate handler based on the mode
	switch mode {
	case base.ConnectionModePacket:
		handlePacketMode(connWrap, appRunId)
	case base.ConnectionModeLog:
		handleLogMode(connWrap, appRunId, submode)
	default:
		log.Printf("Unhandled connection mode: %s\n", mode)
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
