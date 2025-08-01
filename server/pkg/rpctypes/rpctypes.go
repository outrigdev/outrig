// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package rpctypes

import (
	"context"
	"reflect"

	"github.com/outrigdev/outrig/pkg/ds"
)

const (
	Command_Message         = "message"
	Command_RouteAnnounce   = "routeannounce"
	Command_RouteUnannounce = "routeunannounce"
	Command_EventRecv       = "eventrecv"
)

const (
	Event_RouteDown       = "route:down"
	Event_RouteUp         = "route:up"
	Event_AppStatusUpdate = "app:statusupdate"
)

var EventToTypeMap = map[string]reflect.Type{
	Event_RouteDown:       nil,
	Event_RouteUp:         nil,
	Event_AppStatusUpdate: reflect.TypeOf(StatusUpdateData{}),
}

type FullRpcInterface interface {
	MessageCommand(ctx context.Context, data CommandMessageData) error

	LogSearchRequestCommand(ctx context.Context, data SearchRequestData) (SearchResultData, error)
	LogWidgetAdminCommand(ctx context.Context, data LogWidgetAdminData) error
	LogStreamUpdateCommand(ctx context.Context, data StreamUpdateData) error
	LogUpdateMarkedLinesCommand(ctx context.Context, data MarkedLinesData) error
	LogGetMarkedLinesCommand(ctx context.Context, data MarkedLinesRequestData) (MarkedLinesResultData, error)

	UpdateStatusCommand(ctx context.Context, data StatusUpdateData) error

	// app run commands
	GetAppRunsCommand(ctx context.Context, data AppRunUpdatesRequest) (AppRunsData, error)
	GetAppRunRuntimeStatsCommand(ctx context.Context, data AppRunRequest) (AppRunRuntimeStatsData, error)

	// goroutine search
	GetAppRunGoRoutinesByIdsCommand(ctx context.Context, data AppRunGoRoutinesByIdsRequest) (AppRunGoRoutinesData, error)
	GoRoutineSearchRequestCommand(ctx context.Context, data GoRoutineSearchRequestData) (GoRoutineSearchResultData, error)
	GoRoutineTimeSpansCommand(ctx context.Context, data GoRoutineTimeSpansRequest) (GoRoutineTimeSpansResponse, error)

	// watch search
	GetAppRunWatchesByIdsCommand(ctx context.Context, data AppRunWatchesByIdsRequest) (AppRunWatchesData, error)
	WatchSearchRequestCommand(ctx context.Context, data WatchSearchRequestData) (WatchSearchResultData, error)

	// event commands
	EventPublishCommand(ctx context.Context, data EventType) error
	EventSubCommand(ctx context.Context, data SubscriptionRequest) error
	EventUnsubCommand(ctx context.Context, data string) error
	EventUnsubAllCommand(ctx context.Context) error
	EventReadHistoryCommand(ctx context.Context, data EventReadHistoryData) ([]*EventType, error)

	// tevent commands
	SendTEventFeCommand(ctx context.Context, data TEventFeData) error

	// browser tab tracking
	UpdateBrowserTabUrlCommand(ctx context.Context, data BrowserTabUrlData) error

	// update check commands
	UpdateCheckCommand(ctx context.Context) (UpdateCheckData, error)
	TriggerTrayUpdateCommand(ctx context.Context) error

	// app peer management commands
	ClearNonActiveAppRunsCommand(ctx context.Context) error

	// demo controller commands
	LaunchDemoAppCommand(ctx context.Context) error
	KillDemoAppCommand(ctx context.Context) error
	GetDemoAppStatusCommand(ctx context.Context) (string, error)
}

// UpdateCheckData represents the response for update check information
type UpdateCheckData struct {
	NewerVersion string `json:"newerversion"`
	FromTrayApp  bool   `json:"fromtrayapp,omitempty"`
}

type RespUnion[T any] struct {
	Response T
	Error    error
}

type CommandMessageData struct {
	Message string `json:"message"`
}

// for frontend
type ServerCommandMeta struct {
	CommandType string `json:"commandtype"`
}

type SearchRequestData struct {
	WidgetId     string `json:"widgetid"`
	AppRunId     string `json:"apprunid"`
	SearchTerm   string `json:"searchterm"`
	SystemQuery  string `json:"systemquery,omitempty"`
	PageSize     int    `json:"pagesize"`
	RequestPages []int  `json:"requestpages"`
	Streaming    bool   `json:"streaming"`
}

type PageData struct {
	PageNum int          `json:"pagenum"`
	Lines   []ds.LogLine `json:"lines"`
}

// SearchErrorSpan represents an error in a search query with position information
type SearchErrorSpan struct {
	Start        int    `json:"start"`        // Start position in the search term
	End          int    `json:"end"`          // End position in the search term
	ErrorMessage string `json:"errormessage"` // The error message
}

type SearchResultData struct {
	FilteredCount int               `json:"filteredcount"`
	SearchedCount int               `json:"searchedcount"`
	TotalCount    int               `json:"totalcount"`
	MaxCount      int               `json:"maxcount"`
	Pages         []PageData        `json:"pages"`
	ErrorSpans    []SearchErrorSpan `json:"errorspans,omitempty"` // Error spans in the search query
}

type StreamUpdateData struct {
	WidgetId      string       `json:"widgetid"`
	FilteredCount int          `json:"filteredcount"`
	SearchedCount int          `json:"searchedcount"`
	TotalCount    int          `json:"totalcount"`
	TrimmedLines  int          `json:"trimmedlines"`
	Offset        int          `json:"offset"`
	Lines         []ds.LogLine `json:"lines"`
}

type DropRequestData struct {
	WidgetId string `json:"widgetid"`
}

type LogWidgetAdminData struct {
	WidgetId  string `json:"widgetid"`
	Drop      bool   `json:"drop,omitempty"`
	KeepAlive bool   `json:"keepalive,omitempty"`
}

// MarkedLinesData represents the data for managing marked lines
type MarkedLinesData struct {
	WidgetId    string          `json:"widgetid"`
	MarkedLines map[string]bool `json:"markedlines"`
	Clear       bool            `json:"clear,omitempty"`
}

// MarkedLinesRequestData represents the request for getting marked lines
type MarkedLinesRequestData struct {
	WidgetId string `json:"widgetid"`
}

// MarkedLinesResultData represents the response with marked log lines
type MarkedLinesResultData struct {
	Lines []ds.LogLine `json:"lines"`
}

type StatusUpdateData struct {
	AppId         string `json:"appid"`
	Status        string `json:"status"`
	NumLogLines   int    `json:"numloglines"`
	NumGoRoutines int    `json:"numgoroutines"`
}

// BuildInfoData represents a simplified version of runtime/debug.BuildInfo
type BuildInfoData struct {
	GoVersion string            `json:"goversion"`
	Path      string            `json:"path"`
	Version   string            `json:"version,omitempty"`
	Settings  map[string]string `json:"settings,omitempty"`
}

// App run data types
type AppRunInfo struct {
	AppRunId                   string         `json:"apprunid"`
	AppName                    string         `json:"appname"`
	StartTime                  int64          `json:"starttime"`
	FirstGoRoutineCollectionTs int64          `json:"firstgoroutinecollectionts,omitempty"`
	IsRunning                  bool           `json:"isrunning"`
	Status                     string         `json:"status"`
	NumLogs                    int            `json:"numlogs"`
	NumTotalGoRoutines         int            `json:"numtotalgoroutines"`
	NumActiveGoRoutines        int            `json:"numactivegoroutines"`
	NumOutrigGoRoutines        int            `json:"numoutriggoroutines"`
	NumActiveWatches           int            `json:"numactivewatches"`
	NumTotalWatches            int            `json:"numtotalwatches"`
	LastModTime                int64          `json:"lastmodtime"`
	BuildInfo                  *BuildInfoData `json:"buildinfo,omitempty"`
	ModuleName                 string         `json:"modulename,omitempty"`
	Executable                 string         `json:"executable,omitempty"`
	OutrigSDKVersion           string         `json:"outrigsdkversion,omitempty"`
}

type AppRunsData struct {
	AppRuns []AppRunInfo `json:"appruns"`
}

type AppRunUpdatesRequest struct {
	Since int64 `json:"since"`
}

type AppRunRequest struct {
	AppRunId string `json:"apprunid"`
	Since    int64  `json:"since,omitempty"`
}

// AppRunGoRoutinesByIdsRequest defines the request for getting specific goroutines by their IDs
type AppRunGoRoutinesByIdsRequest struct {
	AppRunId  string  `json:"apprunid"`
	GoIds     []int64 `json:"goids"`
	Timestamp int64   `json:"timestamp"` // if 0, get latest stack; otherwise get stack at this timestamp
}

// AppRunWatchesByIdsRequest defines the request for getting specific watches by their IDs
type AppRunWatchesByIdsRequest struct {
	AppRunId string  `json:"apprunid"`
	WatchIds []int64 `json:"watchids"`
}

type AppRunLogsData struct {
	AppRunId string       `json:"apprunid"`
	AppName  string       `json:"appname"`
	Logs     []ds.LogLine `json:"logs"`
}

type AppRunGoRoutinesData struct {
	AppRunId   string            `json:"apprunid"`
	AppName    string            `json:"appname"`
	GoRoutines []ParsedGoRoutine `json:"goroutines"`
}

type AppRunWatchesData struct {
	AppRunId string                `json:"apprunid"`
	AppName  string                `json:"appname"`
	Watches  []CombinedWatchSample `json:"watches"`
}

type RuntimeStatData struct {
	Ts             int64              `json:"ts"`
	GoRoutineCount int                `json:"goroutinecount"`
	GoMaxProcs     int                `json:"gomaxprocs"`
	NumCPU         int                `json:"numcpu"`
	GOOS           string             `json:"goos"`
	GOARCH         string             `json:"goarch"`
	GoVersion      string             `json:"goversion"`
	Pid            int                `json:"pid"`
	Cwd            string             `json:"cwd"`
	MemStats       ds.MemoryStatsInfo `json:"memstats"`
}

type AppRunRuntimeStatsData struct {
	AppRunId            string            `json:"apprunid"`
	AppName             string            `json:"appname"`
	NumTotalGoRoutines  int               `json:"numtotalgoroutines"`
	NumActiveGoRoutines int               `json:"numactivegoroutines"`
	NumOutrigGoRoutines int               `json:"numoutriggoroutines"`
	Stats               []RuntimeStatData `json:"stats"`
}

// GoRoutineSearchRequestData defines the request for goroutine search
type GoRoutineSearchRequestData struct {
	AppRunId    string `json:"apprunid"`
	SearchTerm  string `json:"searchterm"`
	SystemQuery string `json:"systemquery,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"` // Timestamp in milliseconds, 0 means use latest
	ShowOutrig  bool   `json:"showoutrig"`          // Whether to include outrig-tagged goroutines in state counts
	ActiveOnly  bool   `json:"activeonly"`          // Whether to filter to only active goroutines at the timestamp
}

// GoRoutineSearchResultData defines the response for goroutine search
type GoRoutineSearchResultData struct {
	SearchedCount            int               `json:"searchedcount"`
	TotalCount               int               `json:"totalcount"`
	TotalNonOutrig           int               `json:"totalnonoutrig,omitempty"`       // Total count excluding #outrig goroutines (only for goroutines search)
	GoRoutineStateCounts     map[string]int    `json:"goroutinestatecounts,omitempty"` // PrimaryState counts for all searched goroutines
	Results                  []int64           `json:"results"`
	ErrorSpans               []SearchErrorSpan `json:"errorspans,omitempty"`     // Error spans in the search query
	EffectiveSearchTimestamp int64             `json:"effectivesearchtimestamp"` // The actual timestamp used for the search
}

// WatchSearchRequestData defines the request for watch search
type WatchSearchRequestData struct {
	AppRunId    string `json:"apprunid"`
	SearchTerm  string `json:"searchterm"`
	SystemQuery string `json:"systemquery,omitempty"`
}

// WatchSearchResultData defines the response for watch search
type WatchSearchResultData struct {
	SearchedCount int               `json:"searchedcount"`
	TotalCount    int               `json:"totalcount"`
	Results       []int64           `json:"results"`
	ErrorSpans    []SearchErrorSpan `json:"errorspans,omitempty"` // Error spans in the search query
}

type EventReadHistoryData struct {
	Event    string `json:"event"`
	Scope    string `json:"scope"`
	MaxItems int    `json:"maxitems"`
}

// for FE (for discrimated union)
type EventCommonFields struct {
	Scopes  []string `json:"scopes,omitempty"`
	Sender  string   `json:"sender,omitempty"`
	Persist int      `json:"persist,omitempty"`
}

type EventType struct {
	Event   string   `json:"event"`
	Scopes  []string `json:"scopes,omitempty"`
	Sender  string   `json:"sender,omitempty"`
	Persist int      `json:"persist,omitempty"`
	Data    any      `json:"data,omitempty"`
}

type SubscriptionRequest struct {
	Event     string   `json:"event"`
	Scopes    []string `json:"scopes,omitempty"`
	AllScopes bool     `json:"allscopes,omitempty"`
}

// BrowserTabUrlData represents the data for tracking browser tabs
type BrowserTabUrlData struct {
	Url        string `json:"url"`
	AppRunId   string `json:"apprunid,omitempty"`
	Focused    bool   `json:"focused"`
	AutoFollow bool   `json:"autofollow"`
}

// StackFrame represents a single frame in a goroutine stack trace
type StackFrame struct {
	// Function information
	Package  string `json:"package"`            // The package name (e.g., "internal/poll")
	FuncName string `json:"funcname"`           // Just the function/method name, may include the receiver (e.g., "Read")
	FuncArgs string `json:"funcargs,omitempty"` // Raw argument string, no parens (e.g., "0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd}")

	// Source file information
	FilePath   string `json:"filepath"`           // Full path to the source file (e.g., "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go")
	LineNumber int    `json:"linenumber"`         // Line number in the source file (e.g., 165)
	PCOffset   string `json:"pcoffset,omitempty"` // Program counter offset (e.g., "+0x1fc")

	// Classification flags
	IsImportant bool `json:"isimportant,omitempty"` // True if the frame is from the user's own module
	IsSys       bool `json:"issys,omitempty"`       // True if the frame is from a system module (e.g., "os", "net", "internal")
}

type TimeSpan struct {
	Label    string `json:"label,omitempty"`    // Label for the time span (e.g., "Running", "Waiting")
	Start    int64  `json:"start"`              // Start time in milliseconds
	StartIdx int    `json:"startidx"`           // Start index in the logical time sequence (if applicable)
	End      int64  `json:"end"`                // End time in milliseconds (-1 means ongoing)
	EndIdx   int    `json:"endidx"`             // End index in the logical time sequence (-1 means ongoing)
	Exact    bool   `json:"exact,omitempty"`    // True if the start and end times are exact (not approximate)
}

func (span TimeSpan) IsWithinSpanTs(ts int64) bool {
	if ts < span.Start {
		return false
	}
	return (span.End == -1 || ts <= span.End)
}

func (span TimeSpan) IsWithinSpanIdx(idx int) bool {
	if idx < span.StartIdx {
		return false
	}
	return (span.EndIdx == -1 || idx <= span.EndIdx)
}

// ParsedGoRoutine represents a parsed goroutine stack trace
type ParsedGoRoutine struct {
	GoId            int64        `json:"goid"`
	Name            string       `json:"name,omitempty"`            // Optional name for the goroutine
	Tags            []string     `json:"tags,omitempty"`            // Optional tags for the goroutine
	CSNum           int          `json:"csnum,omitempty"`           // Call site number for goroutines spawned from the same location
	ActiveTimeSpan  TimeSpan     `json:"activetimespan"`            // Time span when the goroutine was active
	Active          bool         `json:"active"`                    // Whether the goroutine is currently active
	RawStackTrace   string       `json:"rawstacktrace"`             // The raw stack trace string
	RawState        string       `json:"rawstate"`                  // The complete state information
	PrimaryState    string       `json:"primarystate"`              // The first part of the state (before any commas)
	StateDurationMs int64        `json:"statedurationms,omitempty"` // Duration of state in milliseconds (if available)
	StateDuration   string       `json:"stateduration,omitempty"`   // Duration of state as a string (if available)
	ParsedFrames    []StackFrame `json:"parsedframes,omitempty"`    // Structured frame information
	CreatedByGoId   int64        `json:"createdbygoid,omitempty"`   // ID of the goroutine that created this one
	CreatedByFrame  *StackFrame  `json:"createdbyframe,omitempty"`  // Frame information for the creation point
	Parsed          bool         `json:"parsed"`                    // Whether the stack trace was successfully parsed
	ParseError      string       `json:"parseerror,omitempty"`      // Error message if parsing failed
}

// TEventFeProps represents frontend-related properties for a telemetry event
type TEventFeProps struct {
	FrontendTab            string   `json:"frontend:tab,omitempty"`
	FrontendSearchFeatures []string `json:"frontend:logsearchfeatures,omitempty"`
	FrontendSearchLatency  int      `json:"frontend:searchlatency,omitempty"`
	FrontendSearchItems    int      `json:"frontend:searchitems,omitempty"`
	FrontendClickType      string   `json:"frontend:clicktype,omitempty"`
}

// TEventFeData represents a simplified telemetry event for frontend use
type TEventFeData struct {
	Event string        `json:"event"`
	Props TEventFeProps `json:"props"`
}

type CombinedWatchSample struct {
	WatchNum int64          `json:"watchnum"`
	Decl     ds.WatchDecl   `json:"decl"`
	Sample   ds.WatchSample `json:"sample"`
}

// GoTimeSpan represents a goroutine's time span with its ID
type GoTimeSpan struct {
	GoId uint64   `json:"goid"`
	Span TimeSpan `json:"span"`
}

// GoRoutineTimeSpansRequest defines the request for getting goroutine time spans since a tick index
type GoRoutineTimeSpansRequest struct {
	AppRunId      string `json:"apprunid"`
	SinceTickIdx  int64  `json:"sincetickidx"`
	ShowOutrig    bool   `json:"showoutrig"` // Whether to include outrig-tagged goroutines in time spans
}

type GoRoutineActiveCount struct {
	Count   int   `json:"count"`
	TimeIdx int   `json:"timeidx"` // Index in the logical time sequence
	Ts      int64 `json:"ts"`      // Timestamp in milliseconds
}

type Tick struct {
	Idx int   `json:"idx"` // Index in the logical time sequence
	Ts  int64 `json:"ts"`  // Timestamp in milliseconds
}

// GoRoutineTimeSpansResponse defines the response with updated time spans
type GoRoutineTimeSpansResponse struct {
	Data         []GoTimeSpan           `json:"data"`
	ActiveCounts []GoRoutineActiveCount `json:"activecounts"`
	FullTimeSpan TimeSpan               `json:"fulltimespan"`
	LastTick     Tick                   `json:"lasttick"`
	DroppedCount int64                  `json:"droppedcount"`
}
