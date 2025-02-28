package rpcimpl

import (
	"context"
)

const (
	Command_Message = "message"
)

type FullRpcInterface interface {
	MessageCommand(ctx context.Context, data CommandMessageData) error
}

type CommandMessageData struct {
	Message string `json:"message"`
}
