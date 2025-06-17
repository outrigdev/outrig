// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package comm

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const ConnectDialTimeout = 500 * time.Millisecond

type ConnectAddr struct {
	ConnType string
	Network  string
	DialAddr string
}

func (ca ConnectAddr) IsTcp() bool {
	return ca.Network == "tcp"
}

func connectAddrsToStrings(addrs []ConnectAddr) []string {
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		result[i] = addr.DialAddr
	}
	return result
}

var (
	dockerProbeOnce  sync.Once
	dockerHostExists bool
)

// probeDockerHost performs a one-time DNS lookup for host.docker.internal and returns true if it exists
func probeDockerHost() bool {
	dockerProbeOnce.Do(func() {
		_, err := net.LookupHost("host.docker.internal")
		dockerHostExists = (err == nil)
	})
	return dockerHostExists
}

// MakeConnectAddrs builds the list of connection addresses based on config and environment variables
func MakeConnectAddrs(cfg *config.Config) []ConnectAddr {
	// Check for domain socket override from environment variable
	domainSocketPath := cfg.DomainSocketPath
	if envPath := os.Getenv(config.DomainSocketEnvName); envPath != "" {
		domainSocketPath = envPath
	}

	// Check for TCP address override from environment variable
	tcpAddr := cfg.TcpAddr
	if envAddr := os.Getenv(config.TcpAddrEnvName); envAddr != "" {
		tcpAddr = envAddr
	}

	var connectAddrs []ConnectAddr
	if domainSocketPath != "" && domainSocketPath != "-" {
		dialAddr := utilfn.ExpandHomeDir(domainSocketPath)
		connectAddrs = append(connectAddrs, ConnectAddr{
			ConnType: "domain socket",
			Network:  "unix",
			DialAddr: dialAddr,
		})
	}
	if tcpAddr != "" && tcpAddr != "-" {
		connectAddrs = append(connectAddrs, ConnectAddr{
			ConnType: "TCP server",
			Network:  "tcp",
			DialAddr: tcpAddr,
		})
	}

	// Check for disable docker probe override from environment variable
	disableDockerProbe := cfg.DisableDockerProbe
	if envDisable := os.Getenv(config.DisableDockerProbeEnvName); envDisable != "" {
		disableDockerProbe = true
	}

	// Add Docker host probe if enabled and running in Docker environment
	if !disableDockerProbe && utilfn.InDockerEnv() {
		if probeDockerHost() {
			port := config.GetMonitorPort()
			dockerAddr := "host.docker.internal:" + strconv.Itoa(port)
			connectAddrs = append(connectAddrs, ConnectAddr{
				ConnType: "TCP server",
				Network:  "tcp",
				DialAddr: dockerAddr,
			})
		}
	}

	return connectAddrs
}

// Connect attempts to connect to addresses based on the provided config,
// performs the handshake with the specified mode, submode, and appRunId,
// and returns a new ConnWrap if successful.
//
// It tries each address in order using the structured ConnectAddr information.
// When no valid addresses are available, the function returns (nil, nil, nil) without error.
//
// The function returns (ConnWrap, PermanentError, TransientError)
func Connect(mode string, submode string, appRunId string, cfg *config.Config) (*ConnWrap, error, error) {
	connectAddrs := MakeConnectAddrs(cfg)
	var triedConnect bool
	var attemptedAddrs []string

	// log.Printf("Connecting to Outrig server with mode: %s, submode: %s, appRunId: %s, connectAddrs: %v\n", mode, submode, appRunId, connectAddrsToStrings(connectAddrs))
	for _, connectAddr := range connectAddrs {
		if connectAddr.DialAddr == "" || connectAddr.DialAddr == "-" {
			continue
		}

		triedConnect = true
		attemptedAddrs = append(attemptedAddrs, connectAddr.DialAddr)

		// For domain sockets, check if the file exists
		if connectAddr.Network == "unix" {
			if _, errStat := os.Stat(connectAddr.DialAddr); errStat != nil {
				continue
			}
		}
		conn, err := net.DialTimeout(connectAddr.Network, connectAddr.DialAddr, ConnectDialTimeout)
		if err != nil {
			continue
		}
		connWrap := MakeConnWrap(conn, connectAddr.DialAddr)
		sresp, err := connWrap.ClientHandshake(mode, submode, appRunId, connectAddr.IsTcp())
		if err != nil {
			connWrap.Close()
			return nil, fmt.Errorf("handshake failed with %s: %w", connWrap.PeerName, err), nil
		}
		connWrap.ServerResponse = sresp
		return connWrap, nil, nil
	}
	if !triedConnect {
		return nil, nil, nil
	}
	errMsg := fmt.Sprintf("failed to connect to outrig addresses: %v", attemptedAddrs)
	return nil, nil, errors.New(errMsg)
}
