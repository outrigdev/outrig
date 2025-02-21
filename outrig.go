package outrig

import (
	"sync/atomic"

	"github.com/outrigdev/outrig/pkg/core"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
)

const OutrigHome = "~/.outrig"
const DefaultDomainSocket = OutrigHome + "/outrig.sock"
const DefaultTCPAddr = "http://localhost:5005"

// Optionally re-export ds.Config so callers can do "outrig.Config" if you prefer:
type Config = ds.Config

func Disable() {
	atomic.StoreInt32(&global.OutrigForceDisabled, 1)
	core.Disconnect()
}

func Enable() {
	atomic.StoreInt32(&global.OutrigForceDisabled, 0)
	core.TryConnect()
}

func Init(cfgParam *ds.Config) error {
	finalCfg := *cfgParam
	if finalCfg.DomainSocketPath == "" {
		finalCfg.DomainSocketPath = DefaultDomainSocket
	}
	if finalCfg.ServerAddr == "" {
		finalCfg.ServerAddr = DefaultTCPAddr
	}
	core.SetConfig(&finalCfg)
	core.TryConnect()
	return nil
}
