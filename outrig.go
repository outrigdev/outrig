package outrig

import (
	"os"
	"os/user"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/controller"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/logprocess"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// Optionally re-export ds.Config so callers can do "outrig.Config" if you prefer:
type Config = ds.Config

var ctrl *controller.ControllerImpl

// Disable disables Outrig
func Disable(disconnect bool) {
	if ctrl != nil {
		ctrl.Disable(disconnect)
	}
}

// Enable enables Outrig
func Enable() {
	if ctrl != nil {
		ctrl.Enable()
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ds.Config {
	return &ds.Config{
		DomainSocketPath: base.DefaultDomainSocketName,
		ServerAddr:       base.DefaultTCPAddr,
		AppName:          "",
		ModuleName:       "",
		WrapStdout:       true,
		WrapStderr:       true,
	}
}

// Init initializes Outrig
func Init(cfgParam *ds.Config) error {
	if cfgParam == nil {
		cfgParam = DefaultConfig()
	}
	finalCfg := *cfgParam
	if finalCfg.DomainSocketPath == "" {
		finalCfg.DomainSocketPath = base.DefaultDomainSocketName
	}
	if finalCfg.ServerAddr == "" {
		finalCfg.ServerAddr = base.DefaultTCPAddr
	}

	// Create and initialize the controller
	ctrl = controller.NewController()
	global.GlobalController = ctrl
	ctrl.SetConfig(&finalCfg)

	// Initialize the init info
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

	// Initialize log processing
	logprocess.InitLogProcess()

	// Initialize the controller
	return ctrl.Init(&finalCfg)
}

// Shutdown shuts down Outrig
func Shutdown() {
	if ctrl != nil {
		ctrl.Shutdown()
	}
}
