package outrig

import (
	"os"
	"os/user"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/core"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/logprocess"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// Optionally re-export ds.Config so callers can do "outrig.Config" if you prefer:
type Config = ds.Config

func Disable(disconnect bool) {
	atomic.StoreInt32(&global.OutrigForceDisabled, 1)
	atomic.StoreInt32(&global.OutrigEnabled, 0)
	if disconnect {
		core.Disconnect()
	}
}

func Enable() {
	atomic.StoreInt32(&global.OutrigForceDisabled, 0)
	if atomic.LoadInt32(&global.OutrigConnected) != 0 {
		atomic.StoreInt32(&global.OutrigEnabled, 1)
	}
	core.TryConnect()
}

func Init(cfgParam *ds.Config) error {
	if cfgParam == nil {
		cfgParam = &ds.Config{}
	}
	finalCfg := *cfgParam
	if finalCfg.DomainSocketPath == "" {
		finalCfg.DomainSocketPath = base.DefaultDomainSocketName
	}
	if finalCfg.ServerAddr == "" {
		finalCfg.ServerAddr = base.DefaultTCPAddr
	}
	core.SetConfig(&finalCfg)
	initInfo := ds.InitInfoType{
		StartTime: time.Now().UnixMilli(),
		Args:      utilfn.CopyStrArr(os.Args),
	}
	initInfo.Executable, _ = os.Executable()
	initInfo.Env = utilfn.CopyStrArr(os.Environ())
	initInfo.Pid = os.Getpid()
	user, err := user.Current()
	if err == nil {
		initInfo.User = user.Username
	}
	hostname, err := os.Hostname()
	if err == nil {
		initInfo.Hostname = hostname
	}
	global.InitInfo.Store(&initInfo)
	logprocess.InitLogProcess()
	go core.RunConnPoller()
	return nil
}

func Shutdown() {
	// TODO wait for last log lines to be sent
	// TODO send shutdown log lines
}
