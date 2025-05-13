// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package ds

import (
	"net"
	"reflect"
	"strconv"
)

// Transport packet types
const (
	PacketTypeLog          = "log"
	PacketTypeMultiLog     = "multilog"
	PacketTypeAppInfo      = "appinfo"
	PacketTypeGoroutine    = "goroutine"
	PacketTypeAppDone      = "appdone"
	PacketTypeWatch        = "watch"
	PacketTypeRuntimeStats = "runtimestats"
)

// Environment variables
const (
	DomainSocketEnvName = "OUTRIG_DOMAINSOCKET"
	DisabledEnvName     = "OUTRIG_DISABLED"
	NoTelemetryEnvName  = "OUTRIG_NOTELEMETRY"
)

const (
	// Preserve lower 5 bits for reflect.Kind (0-31)
	KindMask = 0x1F // 00000000_00011111

	// Shift existing flags up by 5 bits
	WatchFlag_Push     = 1 << 5  // 00000000_00100000
	WatchFlag_Counter  = 1 << 6  // 00000000_01000000
	WatchFlag_Atomic   = 1 << 7  // 00000000_10000000
	WatchFlag_Sync     = 1 << 8  // 00000001_00000000
	WatchFlag_Func     = 1 << 9  // 00000010_00000000
	WatchFlag_Hook     = 1 << 10 // 00000100_00000000
	WatchFlag_Settable = 1 << 11 // 00001000_00000000
	WatchFlag_JSON     = 1 << 12 // 00010000_00000000
	WatchFlag_GoFmt    = 1 << 13 // 00100000_00000000
)

type PacketType struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type LogProcessorConfig struct {
	// Enabled indicates whether the log processor is enabled
	Enabled    bool
	WrapStdout bool
	WrapStderr bool
	// OutrigPath is the full path to the outrig executable (including the executable name)
	// If empty, the system will look for "outrig" in the PATH
	OutrigPath string
	// AdditionalArgs are additional arguments to pass to the outrig command
	// These are inserted before the "capturelogs" argument
	AdditionalArgs []string
}

type WatchConfig struct {
	// Enabled indicates whether the watch collector is enabled
	Enabled bool
}

type GoRoutineConfig struct {
	// Enabled indicates whether the goroutine collector is enabled
	Enabled bool
}

type RuntimeStatsConfig struct {
	// Enabled indicates whether the runtime stats collector is enabled
	Enabled bool
}

type Config struct {
	Quiet bool // If true, suppresses init, connect, and disconnect messages

	// DomainSocketPath is the path to the Unix domain socket. If "" => use default.
	// If "-" => disable domain socket.
	DomainSocketPath string

	// ModuleName is the name of the Go module. If not specified, it will be determined
	// from the go.mod file.
	ModuleName string

	// Dev indicates whether the client is in development mode
	Dev bool

	// If true, try to synchronously connect to the server on Init
	ConnectOnInit bool

	// Collector configurations
	LogProcessorConfig LogProcessorConfig
	WatchConfig        WatchConfig
	GoRoutineConfig    GoRoutineConfig
	RuntimeStatsConfig RuntimeStatsConfig
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

// MultiLogLines represents a collection of log lines to be processed together
type MultiLogLines struct {
	LogLines []LogLine `json:"loglines"`
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
	AppRunId         string         `json:"apprunid"`
	AppName          string         `json:"appname"`
	ModuleName       string         `json:"modulename"`
	Executable       string         `json:"executable"`
	Args             []string       `json:"args"`
	Env              []string       `json:"env"`
	StartTime        int64          `json:"starttime"`
	Pid              int            `json:"pid"`
	User             string         `json:"user,omitempty"`
	Hostname         string         `json:"hostname,omitempty"`
	BuildInfo        *BuildInfoData `json:"buildinfo,omitempty"`
	OutrigSDKVersion string         `json:"outrigsdkversion,omitempty"`
}

type GoroutineInfo struct {
	Ts     int64            `json:"ts"`
	Count  int              `json:"count"`
	Delta  bool             `json:"delta,omitempty"`
	Stacks []GoRoutineStack `json:"stacks"`
}

type GoRoutineStack struct {
	GoId       int64    `json:"goid"`
	State      string   `json:"state,omitempty"`
	Name       string   `json:"name,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	StackTrace string   `json:"stacktrace,omitempty"` // does not include the goroutine header (goid + state)
}

type WatchInfo struct {
	Ts      int64         `json:"ts"`
	Delta   bool          `json:"delta,omitempty"`
	Watches []WatchSample `json:"watches"`
}

type MemoryStatsInfo struct {
	Alloc            uint64 `json:"alloc"`
	TotalAlloc       uint64 `json:"totalalloc"`
	Sys              uint64 `json:"sys"`
	HeapAlloc        uint64 `json:"heapalloc"`
	HeapSys          uint64 `json:"heapsys"`
	HeapIdle         uint64 `json:"heapidle"`
	HeapInuse        uint64 `json:"heapinuse"`
	StackInuse       uint64 `json:"stackinuse"`
	StackSys         uint64 `json:"stacksys"`
	MSpanInuse       uint64 `json:"mspaninuse"`
	MSpanSys         uint64 `json:"mspansys"`
	MCacheInuse      uint64 `json:"mcacheinuse"`
	MCacheSys        uint64 `json:"mcachesys"`
	GCSys            uint64 `json:"gcsys"`
	OtherSys         uint64 `json:"othersys"`
	NextGC           uint64 `json:"nextgc"`
	LastGC           uint64 `json:"lastgc"`
	PauseTotalNs     uint64 `json:"pausetotalns"`
	NumGC            uint32 `json:"numgc"`
	TotalHeapObj     uint64 `json:"totalheapobj"`
	TotalHeapObjFree uint64 `json:"totalheapobjfree"`
}

type RuntimeStatsInfo struct {
	Ts             int64           `json:"ts"`
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
	WatchNum int64    `json:"watchnum,omitempty"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags,omitempty"`
	Ts       int64    `json:"ts"`
	Flags    int      `json:"flags,omitempty"`
	StrVal   string   `json:"strval,omitempty"`
	GoFmtVal string   `json:"gofmtval,omitempty"`
	JsonVal  string   `json:"jsonval,omitempty"`
	Type     string   `json:"type,omitempty"`
	Error    string   `json:"error,omitempty"`
	Addr     []string `json:"addr,omitempty"`
	Cap      int      `json:"cap,omitempty"`
	Len      int      `json:"len,omitempty"`
	WaitTime int64    `json:"waittime,omitempty"`
}

// GetKind extracts the reflect.Kind from the flags
func (w *WatchSample) GetKind() uint {
	return uint(w.Flags & KindMask)
}

// SetKind sets the reflect.Kind in the flags
func (w *WatchSample) SetKind(kind uint) {
	// Clear the current kind bits
	w.Flags &= ^KindMask
	// Set the new kind bits
	w.Flags |= int(kind) & KindMask
}

func (w *WatchSample) IsPush() bool {
	return (w.Flags & WatchFlag_Push) != 0
}

// IsNumeric checks if the value is numeric based on its Kind
func (w *WatchSample) IsNumeric() bool {
	kind := reflect.Kind(w.GetKind())
	switch kind {
	case reflect.Bool:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return true
	default:
		return false
	}
}

// GetNumericVal returns a float64 representation of the value
func (w *WatchSample) GetNumericVal() float64 {
	if !w.IsNumeric() {
		return 0
	}

	kind := reflect.Kind(w.GetKind())
	switch kind {
	case reflect.Bool:
		if w.StrVal == "true" {
			return 1
		}
		return 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(w.StrVal, 64)
		if err != nil {
			return 0
		}
		return val
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return float64(w.Len)
	default:
		return 0
	}
}

// for internal use (import cycles)
type Controller interface {
	// Configuration
	GetConfig() Config
	GetAppRunId() string

	// Transport
	SendPacket(pk *PacketType) (bool, error)

	ILog(format string, args ...any)
}

type AppRunContext struct {
	IsDev    bool
	AppRunId string
}
