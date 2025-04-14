// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package comm

import (
	"errors"
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
// The function returns (ConnWrap, PermanentError, TransientError)
func Connect(mode string, submode string, appRunId string, domainSocketPath string, serverAddr string) (*ConnWrap, error, error) {
	// Skip domain socket if path is empty or "-"
	var triedConnect bool

	if domainSocketPath != "" && domainSocketPath != "-" {
		triedConnect = true
		dsPath := utilfn.ExpandHomeDir(domainSocketPath)
		if _, errStat := os.Stat(dsPath); errStat == nil {
			conn, err := net.DialTimeout("unix", dsPath, 2*time.Second)
			if err == nil {
				connWrap := MakeConnWrap(conn, dsPath)

				// Perform the handshake
				err := connWrap.ClientHandshake(mode, submode, appRunId)
				if err != nil {
					connWrap.Close()
					return nil, fmt.Errorf("handshake failed with %s: %w", connWrap.PeerName, err), nil
				} else {
					return connWrap, nil, nil
				}
			}
		}
	}

	// Fall back to TCP if domain socket failed and TCP is not disabled
	if serverAddr != "" && serverAddr != "-" {
		triedConnect = true
		conn, err := net.DialTimeout("tcp", serverAddr, 2*time.Second)
		if err == nil {
			connWrap := MakeConnWrap(conn, serverAddr)

			// Perform the handshake
			err := connWrap.ClientHandshake(mode, submode, appRunId)
			if err != nil {
				connWrap.Close()
				return nil, fmt.Errorf("handshake failed with %s: %w", connWrap.PeerName, err), nil
			} else {
				return connWrap, nil, nil
			}
		}
	}
	// If both connection methods are disabled or not provided, return nil without error
	if !triedConnect {
		return nil, nil, nil
	}

	// Construct appropriate error message based on what connection methods were attempted
	errMsg := "failed to connect to outrig "
	if domainSocketPath != "" && domainSocketPath != "-" {
		errMsg += fmt.Sprintf("domain socket %s", domainSocketPath)
		if serverAddr != "" && serverAddr != "-" {
			errMsg += fmt.Sprintf(" and TCP server %s", serverAddr)
		}
	} else if serverAddr != "" && serverAddr != "-" {
		errMsg += fmt.Sprintf("TCP server %s", serverAddr)
	}

	return nil, nil, errors.New(errMsg)
}
