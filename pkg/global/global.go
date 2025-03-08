package global

import "github.com/outrigdev/outrig/pkg/ds"

var OutrigEnabled int32 = 0
var OutrigForceDisabled int32 = 0
var OutrigConnected int32 = 0

var LineNum int64 = 0

var TransportErrors int64 = 0
var TransportPacketsSent int64 = 0

var GlobalController ds.Controller
