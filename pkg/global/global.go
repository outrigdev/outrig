package global

import (
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/ds"
)

var OutrigEnabled int32 = 0
var OutrigForceDisabled int32 = 0
var OutrigConnected int32 = 0

var LineNum int64 = 0

var ConfigPtr atomic.Pointer[ds.Config]

var TransportErrors int64 = 0
var TransportPacketsSent int64 = 0

var InitInfo atomic.Pointer[ds.InitInfoType]

type Controller interface {
	// Connection management
	Connect() bool
	Disconnect()
	IsConnected() bool
	IsEnabled() bool
	Enable()
	Disable(disconnect bool)

	// Configuration
	GetConfig() *ds.Config
	SetConfig(cfg *ds.Config)

	// Transport
	SendPacket(pk *ds.PacketType) (bool, error)
	GetTransportStats() (int64, int64) // errors, packets sent

	// Initialization
	Init(cfg *ds.Config) error
	Shutdown()
}

var GlobalController Controller
