// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package global

import (
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/ds"
)

var OutrigEnabled atomic.Bool
var Controller atomic.Pointer[ds.Controller]
