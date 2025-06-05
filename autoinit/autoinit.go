// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Package autoinit provides automatic initialization of Outrig when imported.
// Simply import this package with a blank identifier to automatically call outrig.Init("", nil):
//
//	import _ "github.com/outrigdev/outrig/autoinit"
package autoinit

import (
	"github.com/outrigdev/outrig"
	"github.com/outrigdev/outrig/pkg/global"
)

func init() {
	_, err := outrig.Init("", nil)
	if err == nil {
		global.OutrigAutoInit.Store(true)
	}
}
