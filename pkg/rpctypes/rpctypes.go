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
	GetAppRunLogsCommand(ctx context.Context, data AppRunRequest) (AppRunLogsData, error)
	GetAppRunGoroutinesCommand(ctx context.Context, data AppRunRequest) (AppRunGoroutinesData, error)
	GetAppRunWatchesCommand(ctx context.Context, data AppRunRequest) (AppRunWatchesData, error)

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
	SearchType   string `json:"searchtype,omitempty"`
	PageSize     int    `json:"pagesize"`
	RequestPages []int  `json:"requestpages"`
	Stream       bool   `json:"stream"`
}

type PageData struct {
	PageNum int          `json:"pagenum"`
	Lines   []ds.LogLine `json:"lines"`
}

type SearchResultData struct {
	FilteredCount int        `json:"filteredcount"`
	TotalCount    int        `json:"totalcount"`
	Pages         []PageData `json:"pages"`
}

type StreamUpdateData struct {
	FilteredCount int          `json:"filteredcount"`
	TotalCount    int          `json:"totalcount"`
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

// App run data types
type AppRunInfo struct {
	AppRunId            string `json:"apprunid"`
	AppName             string `json:"appname"`
	StartTime           int64  `json:"starttime"`
	IsRunning           bool   `json:"isrunning"`
	Status              string `json:"status"`
	NumLogs             int    `json:"numlogs"`
	NumActiveGoRoutines int    `json:"numactivegoroutines"`
	NumTotalGoRoutines  int    `json:"numtotalgoroutines"`
	NumActiveWatches    int    `json:"numactivewatches"`
	NumTotalWatches     int    `json:"numtotalwatches"`
	LastModTime         int64  `json:"lastmodtime"`
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

type GoroutineData struct {
	GoId       int64  `json:"goid"`
	State      string `json:"state"`
	StackTrace string `json:"stacktrace"`
}

type AppRunGoroutinesData struct {
	AppRunId   string          `json:"apprunid"`
	AppName    string          `json:"appname"`
	Goroutines []GoroutineData `json:"goroutines"`
}

type AppRunWatchesData struct {
	AppRunId string     `json:"apprunid"`
	AppName  string     `json:"appname"`
	Watches  []ds.Watch `json:"watches"`
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
	Url      string `json:"url"`
	AppRunId string `json:"apprunid,omitempty"`
}
