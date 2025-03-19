package comm

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
)

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
// It sends a mode line in the format "MODE:[mode] [apprunid]\n" and
// expects an "OK\n" response from the server.
// If the server responds with an error, it returns that error.
func (cw *ConnWrap) ClientHandshake(modeName string, appRunId string) error {
	// Send the mode line to identify the connection type
	modeLine := fmt.Sprintf("MODE:%s %s", modeName, appRunId)
	if err := cw.WriteLine(modeLine); err != nil {
		return fmt.Errorf("failed to send mode line: %v", err)
	}
	
	// Read the response line
	resp, err := cw.ReadLine()
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %v", err)
	}
	
	if !strings.HasPrefix(resp, "OK") {
		return fmt.Errorf("received error response from server: %s", strings.TrimSpace(resp))
	}
	
	return nil
}

// ServerHandshake performs the server side of the mode-based handshake protocol.
// It reads a mode line in the format "MODE:[mode] [apprunid]\n", validates it,
// and sends an "OK\n" response if valid or an error message if invalid.
// It returns the mode and appRunId if successful, or an error if the handshake fails.
func (cw *ConnWrap) ServerHandshake() (string, string, error) {
	// Read the mode line
	modeLine, err := cw.ReadLine()
	if err != nil {
		errMsg := fmt.Sprintf("ERROR failed to read mode line: %v", err)
		cw.WriteLine(errMsg)
		return "", "", fmt.Errorf("failed to read mode line: %v", err)
	}
	
	modeLine = strings.TrimSpace(modeLine)
	
	// Parse the mode line format: "MODE:[mode] [apprunid]"
	if !strings.HasPrefix(modeLine, "MODE:") {
		errMsg := "ERROR invalid mode line format"
		cw.WriteLine(errMsg)
		return "", "", fmt.Errorf("invalid mode line format: %s", modeLine)
	}
	
	parts := strings.SplitN(strings.TrimPrefix(modeLine, "MODE:"), " ", 2)
	mode := parts[0]
	appRunId := ""
	if len(parts) > 1 {
		appRunId = parts[1]
	}
	
	// Validate the mode
	validMode := mode == "crashoutput" || mode == "packet" // Using string literals for now, will use constants later
	if !validMode {
		errMsg := fmt.Sprintf("ERROR unknown connection mode: %s", mode)
		cw.WriteLine(errMsg)
		return "", "", fmt.Errorf("unknown connection mode: %s", mode)
	}
	
	// Validate the appRunId as a UUID if provided
	if appRunId != "" {
		_, err := uuid.Parse(appRunId)
		if err != nil {
			errMsg := fmt.Sprintf("ERROR invalid app run ID (not a valid UUID): %s", appRunId)
			cw.WriteLine(errMsg)
			return "", "", fmt.Errorf("invalid app run ID: %s", appRunId)
		}
	}
	
	// Send OK response
	cw.WriteLine("OK")
	return mode, appRunId, nil
}
