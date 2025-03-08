package global

import (
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/ds"
)

var OutrigEnabled int32 = 0
var OutrigForceDisabled int32 = 0
var OutrigConnected int32 = 0

var LineNum int64 = 0

var TransportErrors int64 = 0
var TransportPacketsSent int64 = 0

var InitInfo atomic.Pointer[ds.InitInfoType]

type Controller interface {
	Enable()
	Disable(disconnect bool)

	Connect() bool
	Disconnect()

	// Configuration
	GetConfig() *ds.Config

	// Transport
	SendPacket(pk *ds.PacketType) (bool, error)

	Shutdown()
}

var GlobalController Controller
