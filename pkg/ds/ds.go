package ds

import (
	"net"
)

// Transport packet types
const (
	PacketTypeLog     = "log"
	PacketTypeAppInfo = "appinfo"
)

type PacketType struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Config struct {
	// DomainSocketPath is the path to the Unix domain socket. If "" => use default.
	// If "-" => disable domain socket.
	DomainSocketPath string

	// ServerAddr is the TCP address (host:port). If "" => use default.
	// If "-" => disable TCP.
	ServerAddr string

	// AppName is the name of the application. If not specified, it will be determined
	// from the executable name.
	AppName string

	// ModuleName is the name of the Go module. If not specified, it will be determined
	// from the go.mod file.
	ModuleName string

	WrapStdout bool
	WrapStderr bool
}

// ClientType represents our active connection client
type ClientType struct {
	Conn       net.Conn
	ClientAddr string
}

type LogLine struct {
	LineNum int64  `json:"linenum"`
	Ts      int64  `json:"ts"`
	Msg     string `json:"msg"`
	Source  string `json:"source,omitempty"`
}

type AppInfo struct {
	AppRunId   string   `json:"apprunid"`
	AppName    string   `json:"appname"`
	ModuleName string   `json:"modulename"`
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	StartTime  int64    `json:"starttime"`
	Pid        int      `json:"pid"`
	User       string   `json:"user,omitempty"`
	Hostname   string   `json:"hostname,omitempty"`
}

type Controller interface {
	Enable()
	Disable(disconnect bool)

	Connect() bool
	Disconnect()

	// Configuration
	GetConfig() *Config

	// Transport
	SendPacket(pk *PacketType) (bool, error)

	Shutdown()
}
