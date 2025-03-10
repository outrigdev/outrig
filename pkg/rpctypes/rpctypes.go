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

	SearchRequestCommand(ctx context.Context, data SearchRequestData) (SearchResultData, error)
	DropRequestCommand(ctx context.Context, data DropRequestData) error

	StreamUpdateCommand(ctx context.Context, data StreamUpdateData) error

	UpdateStatusCommand(ctx context.Context, data StatusUpdateData) error

	// app run commands
	GetAppRunsCommand(ctx context.Context) (AppRunsData, error)
	GetAppRunLogsCommand(ctx context.Context, data AppRunRequest) (AppRunLogsData, error)

	// event commands
	EventPublishCommand(ctx context.Context, data EventType) error
	EventSubCommand(ctx context.Context, data SubscriptionRequest) error
	EventUnsubCommand(ctx context.Context, data string) error
	EventUnsubAllCommand(ctx context.Context) error
	EventReadHistoryCommand(ctx context.Context, data EventReadHistoryData) ([]*EventType, error)
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
	WidgetID     string `json:"widgetid"`
	SearchTerm   string `json:"searchterm"`
	ViewOffset   int    `json:"offset"`
	ViewLimit    int    `json:"limit"`
	ScrollBuffer int    `json:"buffer"`
	Stream       bool   `json:"stream"`
}

type SearchResultData struct {
	WidgetID      string       `json:"widgetid"`
	FilteredCount int          `json:"filteredcount"`
	TotalCount    int          `json:"totalcount"`
	Lines         []ds.LogLine `json:"lines"`
}

type StreamUpdateData struct {
	WidgetID      string       `json:"widgetid"`
	FilteredCount int          `json:"filteredcount"`
	TotalCount    int          `json:"totalcount"`
	Lines         []ds.LogLine `json:"lines"`
}

type DropRequestData struct {
	WidgetID string `json:"widgetid"`
}

type StatusUpdateData struct {
	AppId         string `json:"appid"`
	Status        string `json:"status"`
	NumLogLines   int    `json:"numloglines"`
	NumGoRoutines int    `json:"numgoroutines"`
}

// App run data types
type AppRunInfo struct {
	AppRunId  string `json:"apprunid"`
	AppName   string `json:"appname"`
	StartTime int64  `json:"starttime"`
	IsRunning bool   `json:"isrunning"`
	NumLogs   int    `json:"numlogs"`
}

type AppRunsData struct {
	AppRuns []AppRunInfo `json:"appruns"`
}

type AppRunRequest struct {
	AppRunId string `json:"apprunid"`
}

type AppRunLogsData struct {
	AppRunId string       `json:"apprunid"`
	AppName  string       `json:"appname"`
	Logs     []ds.LogLine `json:"logs"`
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
