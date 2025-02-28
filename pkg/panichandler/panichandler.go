// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package panichandler

import "fmt"

// to log NumPanics into the local telemetry system
// gets around import cycles
var PanicTelemetryHandler func(panicType string)

func PanicHandler(debugStr string, recoverVal any) error {
	if recoverVal == nil {
		return nil
	}
	// TODO hook up to internal logging
	// log.Printf("[panic] in %s: %v\n", debugStr, recoverVal)
	// debug.PrintStack()

	if err, ok := recoverVal.(error); ok {
		return fmt.Errorf("panic in %s: %w", debugStr, err)
	}
	return fmt.Errorf("panic in %s: %v", debugStr, recoverVal)
}
