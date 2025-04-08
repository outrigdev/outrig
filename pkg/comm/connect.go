// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package comm

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/outrigdev/outrig/pkg/utilfn"
)

// Connect attempts to connect to either a domain socket or TCP server,
// performs the handshake with the specified mode, submode, and appRunId,
// and returns a new ConnWrap if successful.
//
// If domainSocketPath is not empty, it will first try to connect to the domain socket.
// If that fails and serverAddr is not empty, it will fall back to TCP.
// If both are empty or "-", it will return nil and an error.
//
// The function returns a new ConnWrap and error. If the connection is successful,
// the error will be nil. If the connection fails, the error will contain the reason.
func Connect(mode string, submode string, appRunId string, domainSocketPath string, serverAddr string) (*ConnWrap, error) {
	// Skip domain socket if path is empty or "-"
	if domainSocketPath != "" && domainSocketPath != "-" {
		dsPath := utilfn.ExpandHomeDir(domainSocketPath)
		if _, errStat := os.Stat(dsPath); errStat == nil {
			conn, err := net.DialTimeout("unix", dsPath, 2*time.Second)
			if err == nil {
				connWrap := MakeConnWrap(conn, dsPath)

				// Perform the handshake
				err := connWrap.ClientHandshake(mode, submode, appRunId)
				if err != nil {
					connWrap.Close()
					return nil, fmt.Errorf("handshake failed with %s: %w", connWrap.PeerName, err)
				} else {
					return connWrap, nil
				}
			}
		}
	}

	// Fall back to TCP if domain socket failed and TCP is not disabled
	if serverAddr != "" && serverAddr != "-" {
		conn, err := net.DialTimeout("tcp", serverAddr, 2*time.Second)
		if err == nil {
			connWrap := MakeConnWrap(conn, serverAddr)

			// Perform the handshake
			err := connWrap.ClientHandshake(mode, submode, appRunId)
			if err != nil {
				connWrap.Close()
				return nil, fmt.Errorf("handshake failed with %s: %w", connWrap.PeerName, err)
			} else {
				return connWrap, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to connect to domain socket or TCP server")
}
