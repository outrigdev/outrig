// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package global

import (
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/ds"
)

// The main guard flag to indicate if Outrig is enabled.
// Most SDK functions check this flag before proceeding.
var OutrigEnabled atomic.Bool

// Reference to the main Outrig controller.
var Controller atomic.Pointer[ds.Controller]

// Set when Outrig was initialized via the "autoinit" import
// This flag is only used to control a special log message to warn users if they subsequently call Init() again.
var OutrigAutoInit atomic.Bool

func GetController() ds.Controller {
	c := Controller.Load()
	if c == nil || *c == nil {
		return nil
	}
	return *c
}
