package ds

import (
	"net"
)

type Config struct {
	// DomainSocketPath is the path to the Unix domain socket. If "" => use default.
	// If "-" => disable domain socket.
	DomainSocketPath string

	// ServerAddr is the TCP address (host:port). If "" => use default.
	// If "-" => disable TCP.
	ServerAddr string

	WrapStdout bool
	WrapStderr bool
}

// ClientType represents our active connection client
type ClientType struct {
	Conn       net.Conn
	ClientAddr string
}

type InitInfoType struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	StartTime  int64    `json:"starttime"`
	Pid        int      `json:"pid"`
	User       string   `json:"user,omitempty"`
	Hostname   string   `json:"hostname,omitempty"`
}

type LogLine struct {
	LineNum int64  `json:"linenum"`
	Ts      int64  `json:"ts"`
	Msg     string `json:"msg"`
	Source  string `json:"source,omitempty"`
}
