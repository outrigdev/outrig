package ds

import (
	"net"
)

// Transport packet types
const (
	PacketTypeLog       = "log"
	PacketTypeAppInfo   = "appinfo"
	PacketTypeGoroutine = "goroutine"
)

type PacketType struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type LogProcessorConfig struct {
	WrapStdout bool
	WrapStderr bool
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

	StartAsync bool

	LogProcessorConfig *LogProcessorConfig
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

type GoroutineInfo struct {
	Timestamp int64            `json:"timestamp"`
	Count     int              `json:"count"`
	Stacks    []GoroutineStack `json:"stacks"`
}

type GoroutineStack struct {
	ID         int64  `json:"id"`
	State      string `json:"state"`
	StackTrace string `json:"stacktrace"`
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
