package outrig

import (
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector/watch"
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

func getDefaultConfig(isDev bool) *ds.Config {
	return &ds.Config{
		DomainSocketPath: base.GetDomainSocketNameForClient(isDev),
		ServerAddr:       base.GetTCPAddrForClient(isDev),
		AppName:          "",
		ModuleName:       "",
		Dev:              isDev,
		StartAsync:       false,
		LogProcessorConfig: &ds.LogProcessorConfig{
			WrapStdout: true,
			WrapStderr: true,
		},
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ds.Config {
	return getDefaultConfig(false)
}

func DefaultDevConfig() *ds.Config {
	return getDefaultConfig(true)
}

// Init initializes Outrig
func Init(cfgParam *ds.Config) error {
	if cfgParam == nil {
		cfgParam = DefaultConfig()
	}
	finalCfg := *cfgParam
	if finalCfg.DomainSocketPath == "" {
		finalCfg.DomainSocketPath = base.GetDomainSocketNameForClient(finalCfg.Dev)
	}
	if finalCfg.ServerAddr == "" {
		finalCfg.ServerAddr = base.GetTCPAddrForClient(finalCfg.Dev)
	}

	// Create and initialize the controller
	// (collectors are now initialized inside MakeController)
	var err error
	ctrl, err = controller.MakeController(finalCfg)
	if err != nil {
		return err
	}
	global.GlobalController = ctrl

	return nil
}

// Shutdown shuts down Outrig
func Shutdown() {
	if ctrl != nil {
		ctrl.Shutdown()
	}
}

// AppDone signals that the application is done
// This should be deferred in the program's main function
func AppDone() {
	if ctrl != nil {
		// Send an AppDone packet
		packet := &ds.PacketType{
			Type: ds.PacketTypeAppDone,
			Data: nil, // No data needed for AppDone
		}
		ctrl.SendPacket(packet)

		// Give a small delay to allow the packet to be sent
		time.Sleep(50 * time.Millisecond)
	}
}

func WatchSync(name string, val *any) {
	wc := watch.GetInstance()
	wc.RegisterWatchSync(name, val)
}
