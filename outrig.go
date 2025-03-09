package outrig

import (
	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
	"github.com/outrigdev/outrig/pkg/controller"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
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
		LogProcessorConfig: &ds.LogProcessorConfig{
			WrapStdout: true,
			WrapStderr: true,
		},
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
	var err error
	ctrl, err = controller.MakeController(finalCfg)
	if err != nil {
		return err
	}
	global.GlobalController = ctrl

	// Initialize log processing
	logCollector := logprocess.GetInstance()
	logCollector.InitCollector(ctrl)

	return nil
}

// Shutdown shuts down Outrig
func Shutdown() {
	if ctrl != nil {
		ctrl.Shutdown()
	}
}
