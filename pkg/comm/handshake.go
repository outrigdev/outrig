// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package comm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
)

// Constants for handshake protocol
const (
	OkResponse  = "OK"
	ErrorPrefix = "ERROR"
)

// Connection mode constants
const (
	ConnectionModePacket = "packet"
	ConnectionModeLog    = "log"
)

const MinClientVersion = "v0.1.10-alpha"
const MinServerVersion = "v0.1.10-alpha"

const PacketModeVersion = 1
const LogModeVersion = 1

var ClientModeVersions = map[string]int{
	ConnectionModePacket: PacketModeVersion,
	ConnectionModeLog:    LogModeVersion,
}

var ServerModeDefs = map[string]ProtocolDef{
	ConnectionModePacket: {
		Mode:             ConnectionModePacket,
		Version:          1,
		VersionsAccepted: []int{1},
	},
	ConnectionModeLog: {
		Mode:             ConnectionModeLog,
		Version:          1,
		VersionsAccepted: []int{1},
	},
}

type ProtocolDef struct {
	Mode             string `json:"mode"`
	Version          int    `json:"version"`
	VersionsAccepted []int  `json:"versionsaccepted"`
}

type ServerHandshakePacket struct {
	OutrigVersion string                 `json:"outrigversion"`
	Modes         map[string]ProtocolDef `json:"modes"`
}

// ClientHandshakePacket represents the JSON structure for client handshake
type ClientHandshakePacket struct {
	OutrigSDK   string `json:"outrigsdk"`
	Mode        string `json:"mode"`
	ModeVersion int    `json:"modeversion,omitempty"`
	Submode     string `json:"submode,omitempty"`
	AppRunID    string `json:"apprunid,omitempty"`
}

type ServerHandshakeResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Regexp for validating log source paths
var logSourceRegexp = regexp.MustCompile(`^[a-zA-Z0-9.+_/:-]+$`)

// ConnWrap wraps a net.Conn and a bufio.Reader for convenient line-based communication.
type ConnWrap struct {
	Conn     net.Conn
	Reader   *bufio.Reader
	PeerName string
}

// MakeConnWrap creates a new ConnWrap from a net.Conn.
func MakeConnWrap(conn net.Conn, peerName string) *ConnWrap {
	return &ConnWrap{
		Conn:     conn,
		Reader:   bufio.NewReader(conn),
		PeerName: peerName,
	}
}

// ReadLine reads a line from the connection.
func (cw *ConnWrap) ReadLine() (string, error) {
	return cw.Reader.ReadString('\n')
}

// WriteLine writes a line to the connection.
func (cw *ConnWrap) WriteLine(line string) error {
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	_, err := cw.Conn.Write([]byte(line))
	return err
}

// Close closes the underlying connection.
func (cw *ConnWrap) Close() error {
	return cw.Conn.Close()
}

// ClientHandshake performs the client side of the handshake protocol with the server.
// It receives a ServerHandshakePacket, validates compatibility,
// sends a ClientHandshakePacket, and processes the server's response.
func (cw *ConnWrap) ClientHandshake(modeName string, submode string, appRunId string) error {
	// Read the server handshake packet
	packetLine, err := cw.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read server handshake packet: %v", err)
	}

	packetLine = strings.TrimSpace(packetLine)

	// Parse the JSON packet
	var serverPacket ServerHandshakePacket
	if err := json.Unmarshal([]byte(packetLine), &serverPacket); err != nil {
		return fmt.Errorf("invalid server handshake packet format: %v", err)
	}

	// Validate the server version using semver
	serverVersion, err := semver.NewVersion(strings.TrimPrefix(serverPacket.OutrigVersion, "v"))
	if err != nil {
		return fmt.Errorf("invalid server version format: %s", serverPacket.OutrigVersion)
	}

	minVersion, _ := semver.NewVersion(strings.TrimPrefix(MinServerVersion, "v"))
	if serverVersion.LessThan(minVersion) {
		return fmt.Errorf("server version %s is less than minimum required version %s",
			serverPacket.OutrigVersion, MinServerVersion)
	}

	// Check if the requested mode is supported by the server
	protocolDef, modeSupported := serverPacket.Modes[modeName]
	if !modeSupported {
		return fmt.Errorf("server does not support mode: %s", modeName)
	}

	// Get the client's version for this mode
	clientModeVersion, hasModeVersion := ClientModeVersions[modeName]
	if !hasModeVersion {
		return fmt.Errorf("client does not support mode: %s", modeName)
	}

	// Check if the server accepts this mode version
	versionAccepted := false
	for _, v := range protocolDef.VersionsAccepted {
		if v == clientModeVersion {
			versionAccepted = true
			break
		}
	}

	if !versionAccepted {
		return fmt.Errorf("server does not accept mode version %d for mode %s. Supported versions: %v",
			clientModeVersion, modeName, protocolDef.VersionsAccepted)
	}

	// Create the client handshake packet
	clientPacket := ClientHandshakePacket{
		OutrigSDK:   base.OutrigSDKVersion,
		Mode:        modeName,
		ModeVersion: clientModeVersion,
		Submode:     submode,
		AppRunID:    appRunId,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(clientPacket)
	if err != nil {
		return fmt.Errorf("failed to marshal client handshake packet: %v", err)
	}

	// Send the JSON packet
	if err := cw.WriteLine(string(jsonData)); err != nil {
		return fmt.Errorf("failed to send client handshake packet: %v", err)
	}

	// Read the response
	respLine, err := cw.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read server handshake response: %v", err)
	}

	respLine = strings.TrimSpace(respLine)

	// Parse the response
	var response ServerHandshakeResponse
	if err := json.Unmarshal([]byte(respLine), &response); err != nil {
		// Try to handle legacy format (just "OK" or "ERROR: message")
		if respLine == OkResponse {
			return nil
		}
		return fmt.Errorf("received error response from server: %s", respLine)
	}

	if !response.Success {
		return fmt.Errorf("handshake failed: %s", response.Error)
	}

	return nil
}

// Helper function to send error response
func sendErrorResponse(cw *ConnWrap, errMsg string) error {
	response := ServerHandshakeResponse{
		Success: false,
		Error:   errMsg,
	}
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal error response: %v", err)
	}
	return cw.WriteLine(string(jsonData))
}

// Helper function to send success response
func sendSuccessResponse(cw *ConnWrap) error {
	response := ServerHandshakeResponse{
		Success: true,
	}
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal success response: %v", err)
	}
	return cw.WriteLine(string(jsonData))
}

// ServerHandshake performs the server side of the handshake protocol.
// It sends a ServerHandshakePacket, reads a ClientHandshakePacket,
// validates it, and sends a response.
func (cw *ConnWrap) ServerHandshake() (string, string, string, error) {
	// Create and send the server handshake packet
	serverPacket := ServerHandshakePacket{
		OutrigVersion: base.OutrigSDKVersion,
		Modes:         ServerModeDefs,
	}

	jsonData, err := json.Marshal(serverPacket)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal server handshake packet: %v", err)
	}

	if err := cw.WriteLine(string(jsonData)); err != nil {
		return "", "", "", fmt.Errorf("failed to send server handshake packet: %v", err)
	}

	// Read the client handshake packet
	packetLine, err := cw.ReadLine()
	if err != nil {
		errMsg := fmt.Sprintf("%s failed to read client handshake packet: %v", ErrorPrefix, err)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("failed to read client handshake packet: %v", err)
	}

	packetLine = strings.TrimSpace(packetLine)

	// Parse the JSON packet
	var packet ClientHandshakePacket
	if err := json.Unmarshal([]byte(packetLine), &packet); err != nil {
		errMsg := fmt.Sprintf("%s invalid client handshake packet format: %v", ErrorPrefix, err)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("invalid client handshake packet format: %v", err)
	}

	// Validate the outrigsdk field is present
	if packet.OutrigSDK == "" {
		errMsg := fmt.Sprintf("%s missing outrigsdk field", ErrorPrefix)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("missing outrigsdk field")
	}

	// Validate the client SDK version using semver
	clientVersion, err := semver.NewVersion(strings.TrimPrefix(packet.OutrigSDK, "v"))
	if err != nil {
		errMsg := fmt.Sprintf("%s invalid client SDK version format: %s", ErrorPrefix, packet.OutrigSDK)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("invalid client SDK version format: %s", packet.OutrigSDK)
	}

	minVersion, _ := semver.NewVersion(strings.TrimPrefix(MinClientVersion, "v"))
	if clientVersion.LessThan(minVersion) {
		errMsg := fmt.Sprintf("%s client SDK version %s is less than minimum required version %s",
			ErrorPrefix, packet.OutrigSDK, MinClientVersion)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("client SDK version %s is less than minimum required version %s",
			packet.OutrigSDK, MinClientVersion)
	}

	// Validate the mode
	protocolDef, validMode := ServerModeDefs[packet.Mode]
	if !validMode {
		errMsg := fmt.Sprintf("%s unknown connection mode: %s", ErrorPrefix, packet.Mode)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("unknown connection mode: %s", packet.Mode)
	}

	// Validate the mode version
	validVersion := false
	for _, v := range protocolDef.VersionsAccepted {
		if v == packet.ModeVersion {
			validVersion = true
			break
		}
	}

	if !validVersion {
		errMsg := fmt.Sprintf("%s unsupported mode version %d for mode %s. Supported versions: %v",
			ErrorPrefix, packet.ModeVersion, packet.Mode, protocolDef.VersionsAccepted)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("unsupported mode version %d for mode %s. Supported versions: %v",
			packet.ModeVersion, packet.Mode, protocolDef.VersionsAccepted)
	}

	// Validate submode format if present
	if packet.Submode != "" && !logSourceRegexp.MatchString(packet.Submode) {
		errMsg := fmt.Sprintf("%s invalid submode format: %s", ErrorPrefix, packet.Submode)
		sendErrorResponse(cw, errMsg)
		return "", "", "", fmt.Errorf("invalid submode format: %s", packet.Submode)
	}

	// Validate the appRunId as a UUID if provided
	if packet.AppRunID != "" {
		_, err := uuid.Parse(packet.AppRunID)
		if err != nil {
			errMsg := fmt.Sprintf("%s invalid app run ID (not a valid UUID): %s", ErrorPrefix, packet.AppRunID)
			sendErrorResponse(cw, errMsg)
			return "", "", "", fmt.Errorf("invalid app run ID: %s", packet.AppRunID)
		}
	}

	// Send success response
	if err := sendSuccessResponse(cw); err != nil {
		return "", "", "", fmt.Errorf("failed to send success response: %v", err)
	}

	return packet.Mode, packet.Submode, packet.AppRunID, nil
}
