package outrig

import (
	"reflect"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/controller"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"golang.org/x/exp/constraints"
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

type AtomicLoader[T any] interface {
	Load() T
}

type AtomicStorer[T any] interface {
	Store(val T)
}

func WatchCounterSync[T constraints.Integer | constraints.Float](name string, lock sync.Locker, val *T) {
	if val == nil {
		return
	}
	wc := watch.GetInstance()
	rval := reflect.ValueOf(val)
	wc.RegisterWatchSync(name, lock, rval, watch.WatchFlag_Sync|watch.WatchFlag_Settable|watch.WatchFlag_Counter)
}

func WatchSync[T any](name string, lock sync.Locker, val *T) {
	if val == nil {
		return
	}
	wc := watch.GetInstance()
	rval := reflect.ValueOf(val)
	wc.RegisterWatchSync(name, lock, rval, watch.WatchFlag_Sync|watch.WatchFlag_Settable)
}

func WatchAtomicCounter[T constraints.Integer | constraints.Float](name string, val AtomicLoader[T]) {
	if val == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterWatchAtomic(name, val, watch.WatchFlag_Atomic|watch.WatchFlag_Counter)
}

func WatchAtomic[T any](name string, val AtomicLoader[T]) {
	if val == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterWatchAtomic(name, val, watch.WatchFlag_Atomic|watch.WatchFlag_Settable)
}

func WatchCounterFunc[T constraints.Integer | constraints.Float](name string, getFn func() T) {
	if getFn == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterWatchFunc(name, getFn, nil, watch.WatchFlag_Func|watch.WatchFlag_Counter)
}

func WatchFunc[T any](name string, getFn func() T, setFn func(T)) {
	if getFn == nil {
		return
	}
	wc := watch.GetInstance()
	flags := watch.WatchFlag_Func
	if setFn != nil {
		flags |= watch.WatchFlag_Settable
	}
	wc.RegisterWatchFunc(name, getFn, setFn, flags)
}

func TrackValue(name string, val any) {
	if !global.OutrigEnabled.Load() {
		return
	}
	wc := watch.GetInstance()
	rval := reflect.ValueOf(val)
	wc.RecordWatchValue(name, nil, rval, rval.Type().String(), watch.WatchFlag_Push)
}

func TrackCounter[T constraints.Integer | constraints.Float](name string, val T) {
	if !global.OutrigEnabled.Load() {
		return
	}
	wc := watch.GetInstance()
	rval := reflect.ValueOf(val)
	wc.RecordWatchValue(name, nil, rval, rval.Type().String(), watch.WatchFlag_Push|watch.WatchFlag_Counter)
}

func RegisterHook(name string, hookFn any) {
	if hookFn == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterHook(name, hookFn, watch.WatchFlag_Hook|watch.WatchFlag_Settable)
}
