// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Package ds provides data structures, types, and constants used over the wire between the SDK and the server
package ds

import (
	"net"
	"sync"

	"github.com/outrigdev/outrig/pkg/config"
)

// Transport packet types
const (
	PacketTypeLog             = "log"
	PacketTypeMultiLog        = "multilog"
	PacketTypeAppInfo         = "appinfo"
	PacketTypeGoroutine       = "goroutine"
	PacketTypeAppDone         = "appdone"
	PacketTypeWatch           = "watch"
	PacketTypeRuntimeStats    = "runtimestats"
	PacketTypeCollectorStatus = "collectorstatus"
)

type PacketType struct {
	Type string `json:"type"`
	Data any    `json:"data"`
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
	Decls  []GoDecl         `json:"decls,omitempty"`
}

type GoRoutineStack struct {
	GoId       int64    `json:"goid"`
	Ts         int64    `json:"ts"`
	Same       bool     `json:"same,omitempty"` // true if the GoId, State, Name, Tags, and StackTrace are the same as the previous sample (for delta collection)
	State      string   `json:"state,omitempty"`
	Name       string   `json:"name,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	StackTrace string   `json:"stacktrace,omitempty"` // does not include the goroutine header (goid + state)
}

type GoDecl struct {
	Name        string   `json:"name"`
	Tags        []string `json:"tags,omitempty"`
	Pkg         string   `json:"pkg,omitempty"`  // package name that created the goroutine
	Func        string   `json:"func,omitempty"` // function name that created the goroutine (without anonymous func suffixes)
	NewLine     string   `json:"newline,omitempty"`
	RunLine     string   `json:"runline,omitempty"`
	NoRecover   bool     `json:"norecover,omitempty"`
	GoId        int64    `json:"goid,omitempty"`
	ParentGoId  int64    `json:"parentgoid,omitempty"` // ID of the parent goroutine that created this one
	NumSpawned  int64    `json:"numspawned,omitempty"` // Number of goroutines spawned by this one
	State       int32    `json:"state,omitempty"`      // 0 = running, 1 = waiting, 2 = dead
	StartTs     int64    `json:"startts,omitempty"`    // exact start time (from .Run() API)
	EndTs       int64    `json:"endts,omitempty"`      // exact end time (from .Run() API)
	FirstPollTs int64    `json:"firstpollts,omitempty"`
	LastPollTs  int64    `json:"lastpollts,omitempty"`

	RealCreatedBy string `json:"-"` // the real creator of this goroutine (for routines created by the SDK Run() func)
}

type WatchInfo struct {
	Ts        int64            `json:"ts"`
	Delta     bool             `json:"delta,omitempty"`
	Decls     []WatchDecl      `json:"decls,omitempty"`
	Watches   []WatchSample    `json:"watches"`
	RegErrors []ErrWithContext `json:"regerrors,omitempty"`
}

type WatchDecl struct {
	Name         string   `json:"name"`
	Tags         []string `json:"tags,omitempty"`
	NewLine      string   `json:"newline,omitempty"`
	WatchType    string   `json:"watchtype"`
	Format       string   `json:"format"`
	Counter      bool     `json:"counter,omitempty"`
	Invalid      bool     `json:"invalid,omitempty"`
	Unregistered bool     `json:"unregistered,omitempty"`

	SyncLock sync.Locker `json:"-"`
	PollObj  any         `json:"-"`
}

type WatchSample struct {
	Name    string   `json:"name"`
	Ts      int64    `json:"ts"`              // timestamp in milliseconds
	Same    bool     `json:"same,omitempty"`  // true if kind, type, val, addr, error, cap, len, and fmt are the same as the previous sample (for delta collection)
	Kind    int      `json:"kind,omitempty"`  // same
	Type    string   `json:"type,omitempty"`  // same
	Val     string   `json:"val,omitempty"`   // same
	Error   string   `json:"error,omitempty"` // same
	Addr    []string `json:"addr,omitempty"`  // same
	Cap     int      `json:"cap,omitempty"`   // same
	Len     int      `json:"len,omitempty"`   // same
	Fmt     string   `json:"fmt,omitempty"`   // same
	PollDur int64    `json:"polldur,omitempty"`
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

// for internal use (import cycles)
type Controller interface {
	// Configuration
	GetConfig() config.Config

	// Transport
	SendPacket(pk *PacketType) (bool, error)

	ILog(format string, args ...any)
}

// ErrWithContext represents an error with a source code line reference
type ErrWithContext struct {
	Ref   string `json:"ref,omitempty"` // reference to the object that caused the error (e.g. watchname, goroutine id, log source, etc.)
	Error string `json:"error"`         // the error message
	Line  string `json:"line"`          // file:line
}

type CollectorStatus struct {
	Running         bool     `json:"running"`
	Info            string   `json:"info,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Errors          []string `json:"errors,omitempty"`
	CollectDuration int64    `json:"collectduration,omitempty"` // time in milliseconds of last collection
}
