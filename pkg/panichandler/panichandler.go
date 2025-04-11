// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package panichandler

import (
	"fmt"
	"runtime/debug"

	"github.com/outrigdev/outrig/pkg/global"
)

// to log NumPanics into the local telemetry system
// gets around import cycles
var PanicTelemetryHandler func(panicType string)

func PanicHandler(debugStr string, recoverVal any) error {
	if recoverVal == nil {
		return nil
	}
	c := global.Controller.Load()
	if c != nil {
		(*c).ILog("[panic] in %s: %v\n", debugStr, recoverVal)
		(*c).ILog("[panic] stack trace:\n%s", string(debug.Stack()))
	}
	if err, ok := recoverVal.(error); ok {
		return fmt.Errorf("panic in %s: %w", debugStr, err)
	}
	return fmt.Errorf("panic in %s: %v", debugStr, recoverVal)
}
