// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package logprocess

import (
	"fmt"
	"io"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// LogStreamWriter implements io.Writer to send logs to Outrig
type LogStreamWriter struct {
	name    string
	lineBuf *utilfn.LineBuf
}

// Ensure LogStreamWriter implements io.Writer
var _ io.Writer = (*LogStreamWriter)(nil)

// Write implements the io.Writer interface
func (w *LogStreamWriter) Write(p []byte) (n int, err error) {
	if !global.OutrigEnabled.Load() {
		return len(p), nil // Still return success even when disabled
	}

	// Get controller from global
	c := global.Controller.Load()
	if c == nil || *c == nil {
		return 0, fmt.Errorf("controller not initialized")
	}
	controller := *c

	// Process the buffer into lines
	lines := w.lineBuf.ProcessBuf(p)

	var logTs int64
	// Send each complete line as a log packet
	for _, line := range lines {
		if logTs == 0 {
			logTs = time.Now().UnixMilli()
		}
		logLine := &ds.LogLine{
			Ts:     logTs,
			Msg:    line,
			Source: w.name,
		}
		packet := &ds.PacketType{
			Type: ds.PacketTypeLog,
			Data: logLine,
		}
		controller.SendPacket(packet)
	}

	return len(p), nil
}

// MakeLogStreamWriter creates a new LogStreamWriter
func MakeLogStreamWriter(name string) *LogStreamWriter {
	return &LogStreamWriter{
		name:    name,
		lineBuf: utilfn.MakeLineBuf(),
	}
}
