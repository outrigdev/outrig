// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package comm

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
)

// Connection mode constants
const (
	ConnectionModePacket = "packet"
	ConnectionModeLog    = "log"
)

const MinClientVersion = "v0.2.0"
const MinServerVersion = "v0.2.0"

type ServerHandshakePacket struct {
	OutrigVersion string `json:"outrigversion"`
}

// ClientHandshakePacket represents the JSON structure for client handshake
type ClientHandshakePacket struct {
	OutrigSDK string `json:"outrigsdk"`
	Mode      string `json:"mode"`
	Submode   string `json:"submode,omitempty"`
	AppRunID  string `json:"apprunid,omitempty"`
}

type ServerHandshakeResponse struct {
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	ServerHttpPort int    `json:"serverhttpport,omitempty"`
}

// Regexp for validating log source paths
var logSourceRegexp = regexp.MustCompile(`^[a-zA-Z0-9.+_/:-]+$`)

// stripPrereleaseInfo returns a new version without prerelease information
func stripPrereleaseInfo(v *semver.Version) *semver.Version {
	if v == nil {
		return nil
	}
	// Create a new version with just the major, minor, and patch components
	cleanVersion, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch()))
	return cleanVersion
}

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
func (cw *ConnWrap) ClientHandshake(modeName string, submode string, appRunId string) (*ServerHandshakeResponse, error) {
	// Read the server handshake packet
	packetLine, err := cw.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read server handshake packet: %v", err)
	}

	packetLine = strings.TrimSpace(packetLine)

	// Parse the JSON packet
	var serverPacket ServerHandshakePacket
	if err := json.Unmarshal([]byte(packetLine), &serverPacket); err != nil {
		return nil, fmt.Errorf("invalid server handshake packet format: %v", err)
	}

	// Validate the server version using semver
	serverVersion, err := semver.NewVersion(serverPacket.OutrigVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid server version format: %s", serverPacket.OutrigVersion)
	}

	minVersion, _ := semver.NewVersion(MinServerVersion)

	// Strip prerelease info before comparing versions
	cleanServerVersion := stripPrereleaseInfo(serverVersion)
	cleanMinVersion := stripPrereleaseInfo(minVersion)

	if cleanServerVersion.LessThan(cleanMinVersion) {
		return nil, fmt.Errorf("server version %s is less than minimum required version %s",
			serverPacket.OutrigVersion, MinServerVersion)
	}

	// Create the client handshake packet
	clientPacket := ClientHandshakePacket{
		OutrigSDK: base.OutrigSDKVersion,
		Mode:      modeName,
		Submode:   submode,
		AppRunID:  appRunId,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(clientPacket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal client handshake packet: %v", err)
	}

	// Send the JSON packet
	if err := cw.WriteLine(string(jsonData)); err != nil {
		return nil, fmt.Errorf("failed to send client handshake packet: %v", err)
	}

	// Read the response
	respLine, err := cw.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read server handshake response: %v", err)
	}

	respLine = strings.TrimSpace(respLine)

	// Parse the response
	var response ServerHandshakeResponse
	if err := json.Unmarshal([]byte(respLine), &response); err != nil {
		return nil, fmt.Errorf("invalid server handshake response format: %v", err)
	}

	if !response.Success {
		return &response, fmt.Errorf("handshake failed: %s", response.Error)
	}

	return &response, nil
}

// Helper function to send error response
func sendErrorResponse(cw *ConnWrap, err error) error {
	response := ServerHandshakeResponse{
		Success: false,
		Error:   err.Error(),
	}
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal error response: %v", err)
	}
	return cw.WriteLine(string(jsonData))
}

// Helper function to send success response
func sendSuccessResponse(cw *ConnWrap, webServerPort int) error {
	response := ServerHandshakeResponse{
		Success:        true,
		ServerHttpPort: webServerPort,
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
func (cw *ConnWrap) ServerHandshake(webServerPort int) (*ClientHandshakePacket, error) {
	// Create and send the server handshake packet
	serverPacket := ServerHandshakePacket{
		OutrigVersion: base.OutrigSDKVersion,
	}

	jsonData, err := json.Marshal(serverPacket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server handshake packet: %v", err)
	}

	if err := cw.WriteLine(string(jsonData)); err != nil {
		return nil, fmt.Errorf("failed to send server handshake packet: %v", err)
	}

	// Read the client handshake packet
	packetLine, err := cw.ReadLine()
	if errors.Is(err, io.EOF) {
		return nil, io.EOF
	}
	if err != nil {
		readErr := fmt.Errorf("failed to read client handshake packet: %v", err)
		sendErrorResponse(cw, readErr)
		return nil, readErr
	}

	packetLine = strings.TrimSpace(packetLine)

	// Parse the JSON packet
	var packet ClientHandshakePacket
	if err := json.Unmarshal([]byte(packetLine), &packet); err != nil {
		formatErr := fmt.Errorf("invalid client handshake packet format: %v", err)
		sendErrorResponse(cw, formatErr)
		return nil, formatErr
	}

	// Validate the outrigsdk field is present
	if packet.OutrigSDK == "" {
		missingFieldErr := fmt.Errorf("missing outrigsdk field")
		sendErrorResponse(cw, missingFieldErr)
		return nil, missingFieldErr
	}

	// Validate the client SDK version using semver
	clientVersion, err := semver.NewVersion(packet.OutrigSDK)
	if err != nil {
		versionFormatErr := fmt.Errorf("invalid client SDK version format: %s", packet.OutrigSDK)
		sendErrorResponse(cw, versionFormatErr)
		return nil, versionFormatErr
	}

	minVersion, _ := semver.NewVersion(MinClientVersion)

	// Strip prerelease info before comparing versions
	cleanClientVersion := stripPrereleaseInfo(clientVersion)
	cleanMinVersion := stripPrereleaseInfo(minVersion)

	if cleanClientVersion.LessThan(cleanMinVersion) {
		versionErr := fmt.Errorf("client SDK version %s is less than minimum required version %s",
			packet.OutrigSDK, MinClientVersion)
		sendErrorResponse(cw, versionErr)
		return nil, versionErr
	}

	// Validate the mode
	if packet.Mode != ConnectionModePacket && packet.Mode != ConnectionModeLog {
		modeErr := fmt.Errorf("unknown connection mode: %s", packet.Mode)
		sendErrorResponse(cw, modeErr)
		return nil, modeErr
	}

	// Validate submode format if present
	if packet.Submode != "" && !logSourceRegexp.MatchString(packet.Submode) {
		submodeErr := fmt.Errorf("invalid submode format: %s", packet.Submode)
		sendErrorResponse(cw, submodeErr)
		return nil, submodeErr
	}

	// Validate the appRunId as a UUID if provided
	if packet.AppRunID != "" {
		_, err := uuid.Parse(packet.AppRunID)
		if err != nil {
			uuidErr := fmt.Errorf("invalid app run ID (not a valid UUID): %s", packet.AppRunID)
			sendErrorResponse(cw, uuidErr)
			return nil, uuidErr
		}
	}

	// Send success response
	if err := sendSuccessResponse(cw, webServerPort); err != nil {
		return nil, fmt.Errorf("failed to send success response: %v", err)
	}

	return &packet, nil
}
