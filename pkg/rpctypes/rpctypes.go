package rpctypes

import (
	"context"

	"github.com/outrigdev/outrig/pkg/ds"
)

const (
	Command_Message         = "message"
	Command_RouteAnnounce   = "routeannounce"
	Command_RouteUnannounce = "routeunannounce"
)

type FullRpcInterface interface {
	MessageCommand(ctx context.Context, data CommandMessageData) error

	SearchRequestCommand(ctx context.Context, data SearchRequestData) (SearchResultData, error)
	DropRequestCommand(ctx context.Context, data DropRequestData) error

	StreamUpdateCommand(ctx context.Context, data StreamUpdateData) error

	UpdateStatusCommand(ctx context.Context, data StatusUpdateData) error
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
	AppName       string `json:"appname"`
	Status        string `json:"status"`
	NumLogLines   int    `json:"numloglines"`
	NumGoRoutines int    `json:"numgoroutines"`
}
