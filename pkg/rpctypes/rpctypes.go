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
	GetAppRunGoRoutinesCommand(ctx context.Context, data AppRunRequest) (AppRunGoRoutinesData, error)
	GetAppRunWatchesCommand(ctx context.Context, data AppRunRequest) (AppRunWatchesData, error)
	GetAppRunRuntimeStatsCommand(ctx context.Context, data AppRunRequest) (AppRunRuntimeStatsData, error)

	// event commands
	EventPublishCommand(ctx context.Context, data EventType) error
	EventSubCommand(ctx context.Context, data SubscriptionRequest) error
	EventUnsubCommand(ctx context.Context, data string) error
	EventUnsubAllCommand(ctx context.Context) error
	EventReadHistoryCommand(ctx context.Context, data EventReadHistoryData) ([]*EventType, error)

	// browser tab tracking
	UpdateBrowserTabUrlCommand(ctx context.Context, data BrowserTabUrlData) error
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
	SystemQuery  string `json:"systemquery"`
	PageSize     int    `json:"pagesize"`
	RequestPages []int  `json:"requestpages"`
}

type PageData struct {
	PageNum int          `json:"pagenum"`
	Lines   []ds.LogLine `json:"lines"`
}

type SearchResultData struct {
	FilteredCount int        `json:"filteredcount"`
	SearchedCount int        `json:"searchedcount"`
	TotalCount    int        `json:"totalcount"`
	MaxCount      int        `json:"maxcount"`
	Pages         []PageData `json:"pages"`
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
	AppRunId            string         `json:"apprunid"`
	AppName             string         `json:"appname"`
	StartTime           int64          `json:"starttime"`
	IsRunning           bool           `json:"isrunning"`
	Status              string         `json:"status"`
	NumLogs             int            `json:"numlogs"`
	NumActiveGoRoutines int            `json:"numactivegoroutines"`
	NumTotalGoRoutines  int            `json:"numtotalgoroutines"`
	NumActiveWatches    int            `json:"numactivewatches"`
	NumTotalWatches     int            `json:"numtotalwatches"`
	LastModTime         int64          `json:"lastmodtime"`
	BuildInfo           *BuildInfoData `json:"buildinfo,omitempty"`
	ModuleName          string         `json:"modulename,omitempty"`
	Executable          string         `json:"executable,omitempty"`
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
	AppRunId string           `json:"apprunid"`
	AppName  string           `json:"appname"`
	Watches  []ds.WatchSample `json:"watches"`
}

type RuntimeStatData struct {
	Ts             int64              `json:"ts"`
	CPUUsage       float64            `json:"cpuusage"`
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
	AppRunId string            `json:"apprunid"`
	AppName  string            `json:"appname"`
	Stats    []RuntimeStatData `json:"stats"`
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

// ParsedGoRoutine represents a parsed goroutine stack trace
type ParsedGoRoutine struct {
	GoId            int64        `json:"goid"`
	Name            string       `json:"name,omitempty"`            // Optional name for the goroutine
	Tags            []string     `json:"tags,omitempty"`            // Optional tags for the goroutine
	RawStackTrace   string       `json:"rawstacktrace"`             // The raw stack trace string
	RawState        string       `json:"rawstate"`                  // The complete state information
	PrimaryState    string       `json:"primarystate"`              // The first part of the state (before any commas)
	StateDurationMs int64        `json:"statedurationms,omitempty"` // Duration of state in milliseconds (if available)
	StateDuration   string       `json:"stateduration,omitempty"`   // Duration of state as a string (if available)
	ExtraStates     []string     `json:"extrastates,omitempty"`     // Array of additional state information
	ParsedFrames    []StackFrame `json:"parsedframes,omitempty"`    // Structured frame information
	CreatedByGoId   int64        `json:"createdbygoid,omitempty"`   // ID of the goroutine that created this one
	CreatedByFrame  *StackFrame  `json:"createdbyframe,omitempty"`  // Frame information for the creation point
	Parsed          bool         `json:"parsed"`                    // Whether the stack trace was successfully parsed
	ParseError      string       `json:"parseerror,omitempty"`      // Error message if parsing failed
}
