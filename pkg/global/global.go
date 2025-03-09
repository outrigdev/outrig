package global

import (
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/ds"
)

var OutrigEnabled atomic.Bool

var GlobalController ds.Controller
