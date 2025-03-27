package ds

import (
	"net"
)

// Transport packet types
const (
	PacketTypeLog          = "log"
	PacketTypeAppInfo      = "appinfo"
	PacketTypeGoroutine    = "goroutine"
	PacketTypeAppDone      = "appdone"
	PacketTypeWatch        = "watch"
	PacketTypeRuntimeStats = "runtimestats"
)

type PacketType struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type LogProcessorConfig struct {
	WrapStdout bool
	WrapStderr bool
	// OutrigPath is the full path to the outrig executable (including the executable name)
	// If empty, the system will look for "outrig" in the PATH
	OutrigPath string
	// AdditionalArgs are additional arguments to pass to the outrig command
	// These are inserted before the "capturelogs" argument
	AdditionalArgs []string
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

	// Dev indicates whether the client is in development mode
	Dev bool

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

type ViewWindow struct {
	Start int `json:"start"`
	Size  int `json:"size"`
}

func (vw ViewWindow) End() int {
	return vw.Start + vw.Size
}

// BuildInfoData represents a simplified version of runtime/debug.BuildInfo
type BuildInfoData struct {
	GoVersion string            `json:"goversion"`
	Path      string            `json:"path"`
	Version   string            `json:"version,omitempty"`
	Settings  map[string]string `json:"settings,omitempty"`
}

type AppInfo struct {
	AppRunId   string         `json:"apprunid"`
	AppName    string         `json:"appname"`
	ModuleName string         `json:"modulename"`
	Executable string         `json:"executable"`
	Args       []string       `json:"args"`
	Env        []string       `json:"env"`
	StartTime  int64          `json:"starttime"`
	Pid        int            `json:"pid"`
	User       string         `json:"user,omitempty"`
	Hostname   string         `json:"hostname,omitempty"`
	BuildInfo  *BuildInfoData `json:"buildinfo,omitempty"`
}

type GoroutineInfo struct {
	Ts     int64            `json:"ts"`
	Count  int              `json:"count"`
	Stacks []GoRoutineStack `json:"stacks"`
}

type GoRoutineStack struct {
	GoId       int64    `json:"goid"`
	State      string   `json:"state"`
	StackTrace string   `json:"stacktrace"`
	Name       string   `json:"name,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type WatchInfo struct {
	Ts      int64         `json:"ts"`
	Watches []WatchSample `json:"watches"`
}

type MemoryStatsInfo struct {
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"totalalloc"`
	Sys          uint64 `json:"sys"`
	HeapAlloc    uint64 `json:"heapalloc"`
	HeapSys      uint64 `json:"heapsys"`
	HeapIdle     uint64 `json:"heapidle"`
	HeapInuse    uint64 `json:"heapinuse"`
	StackInuse   uint64 `json:"stackinuse"`
	StackSys     uint64 `json:"stacksys"`
	MSpanInuse   uint64 `json:"mspaninuse"`
	MSpanSys     uint64 `json:"mspansys"`
	MCacheInuse  uint64 `json:"mcacheinuse"`
	MCacheSys    uint64 `json:"mcachesys"`
	GCSys        uint64 `json:"gcsys"`
	OtherSys     uint64 `json:"othersys"`
	NextGC       uint64 `json:"nextgc"`
	LastGC       uint64 `json:"lastgc"`
	PauseTotalNs uint64 `json:"pausetotalns"`
	NumGC        uint32 `json:"numgc"`
}

type RuntimeStatsInfo struct {
	Ts             int64           `json:"ts"`
	CPUUsage       float64         `json:"cpuusage"`
	GoRoutineCount int             `json:"goroutinecount"`
	GoMaxProcs     int             `json:"gomaxprocs"`
	NumCPU         int             `json:"numcpu"`
	GOOS           string          `json:"goos"`
	GOARCH         string          `json:"goarch"`
	GoVersion      string          `json:"goversion"`
	Pid            int             `json:"pid"`
	Cwd            string          `json:"cwd"`
	MemStats       MemoryStatsInfo `json:"memstats"`
}

type WatchSample struct {
	Name     string   `json:"name"`
	Tags     []string `json:"tags,omitempty"`
	Ts       int64    `json:"ts"`
	Flags    int      `json:"flags,omitempty"`
	Value    string   `json:"value,omitempty"`
	Type     string   `json:"type"`
	Error    string   `json:"error,omitempty"`
	Addr     []string `json:"addr,omitempty"`
	Cap      int      `json:"cap,omitempty"`
	Len      int      `json:"len,omitempty"`
	WaitTime int64    `json:"waittime,omitempty"`
}

type Controller interface {
	Enable()
	Disable(disconnect bool)

	Connect() bool
	Disconnect()

	// Configuration
	GetConfig() Config
	GetAppRunId() string

	// Transport
	SendPacket(pk *PacketType) (bool, error)

	Shutdown()
}
