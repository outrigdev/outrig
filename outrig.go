package outrig

import (
	"time"

	"github.com/outrigdev/outrig/pkg/base"
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
		StartAsync:       false,
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
