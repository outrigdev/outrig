// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package comm

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/outrigdev/outrig/pkg/base"
)

// Constants for handshake protocol
const (
	ModePrefix  = "MODE:"
	OkResponse  = "OK"
	ErrorPrefix = "ERROR"
)

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

// ClientHandshake performs the client side of the mode-based handshake protocol with the server.
// It sends a mode line in the format "MODE:[mode]:[submode] [apprunid]\n" and
// expects an "OK\n" response from the server.
// If the server responds with an error, it returns that error.
// The submode is optional and can be empty.
func (cw *ConnWrap) ClientHandshake(modeName string, submode string, appRunId string) error {
	// Construct the full mode string (mode:submode)
	fullMode := modeName
	if submode != "" {
		fullMode = fmt.Sprintf("%s:%s", modeName, submode)
	}

	// Send the mode line to identify the connection type
	modeLine := fmt.Sprintf("%s%s %s", ModePrefix, fullMode, appRunId)
	if err := cw.WriteLine(modeLine); err != nil {
		return fmt.Errorf("failed to send mode line: %v", err)
	}

	// Read the response line
	resp, err := cw.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %v", err)
	}

	if !strings.HasPrefix(resp, OkResponse) {
		return fmt.Errorf("received error response from server: %s", strings.TrimSpace(resp))
	}

	return nil
}

// ServerHandshake performs the server side of the mode-based handshake protocol.
// It reads a mode line in the format "MODE:[mode]:[submode] [apprunid]\n", validates it,
// and sends an "OK\n" response if valid or an error message if invalid.
// It returns the mode, submode, and appRunId if successful, or an error if the handshake fails.
// The submode is optional and can be empty.
func (cw *ConnWrap) ServerHandshake() (string, string, string, error) {
	// Read the mode line
	modeLine, err := cw.ReadLine()
	if err != nil {
		errMsg := fmt.Sprintf("%s failed to read mode line: %v", ErrorPrefix, err)
		cw.WriteLine(errMsg)
		return "", "", "", fmt.Errorf("failed to read mode line: %v", err)
	}

	modeLine = strings.TrimSpace(modeLine)

	// Parse the mode line format: "MODE:[mode]:[submode] [apprunid]"
	if !strings.HasPrefix(modeLine, ModePrefix) {
		errMsg := fmt.Sprintf("%s invalid mode line format", ErrorPrefix)
		cw.WriteLine(errMsg)
		return "", "", "", fmt.Errorf("invalid mode line format: %s", modeLine)
	}

	// Extract the part after MODE: prefix
	modeAndAppId := strings.TrimPrefix(modeLine, ModePrefix)

	// Split into mode part and appRunId
	parts := strings.SplitN(modeAndAppId, " ", 2)
	modePart := parts[0]
	appRunId := ""
	if len(parts) > 1 {
		appRunId = parts[1]
	}

	// Parse mode and submode
	// Format can be either "mode" or "mode:submode"
	var mode, submode string
	if strings.Contains(modePart, ":") {
		modeParts := strings.SplitN(modePart, ":", 2)
		mode = modeParts[0]
		submode = modeParts[1]

		// Validate submode format if present
		if submode != "" && !logSourceRegexp.MatchString(submode) {
			errMsg := fmt.Sprintf("%s invalid submode format: %s", ErrorPrefix, submode)
			cw.WriteLine(errMsg)
			return "", "", "", fmt.Errorf("invalid submode format: %s", submode)
		}
	} else {
		mode = modePart
		submode = ""
	}

	// Validate the mode
	validMode := mode == base.ConnectionModePacket ||
		mode == base.ConnectionModeLog

	if !validMode {
		errMsg := fmt.Sprintf("%s unknown connection mode: %s", ErrorPrefix, mode)
		cw.WriteLine(errMsg)
		return "", "", "", fmt.Errorf("unknown connection mode: %s", mode)
	}

	// Validate the appRunId as a UUID if provided
	if appRunId != "" {
		_, err := uuid.Parse(appRunId)
		if err != nil {
			errMsg := fmt.Sprintf("%s invalid app run ID (not a valid UUID): %s", ErrorPrefix, appRunId)
			cw.WriteLine(errMsg)
			return "", "", "", fmt.Errorf("invalid app run ID: %s", appRunId)
		}
	}

	// Send OK response
	cw.WriteLine(OkResponse)
	return mode, submode, appRunId, nil
}
