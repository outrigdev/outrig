package global

import (
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/ds"
)

var OutrigEnabled atomic.Bool
var OutrigConnected atomic.Bool
var OutrigForceDisabled atomic.Bool

var LineNum int64 = 0

var TransportErrors int64 = 0
var TransportPacketsSent int64 = 0

var GlobalController ds.Controller
