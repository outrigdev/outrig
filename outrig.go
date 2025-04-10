// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !no_outrig

package outrig

import (
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/controller"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"golang.org/x/exp/constraints"
)

// Optionally re-export ds.Config so callers can do "outrig.Config" if you prefer:
type Config = ds.Config

var ctrl atomic.Pointer[controller.ControllerImpl]

func init() {
	ioutrig.I = &internalOutrig{}
}

// Disable disables Outrig
func Disable(disconnect bool) {
	ctrlPtr := ctrl.Load()
	if ctrlPtr != nil {
		ctrlPtr.Disable(disconnect)
	}
}

// Enable enables Outrig
func Enable() {
	ctrlPtr := ctrl.Load()
	if ctrlPtr != nil {
		ctrlPtr.Enable()
	}
}

func Enabled() bool {
	return global.OutrigEnabled.Load()
}

func getDefaultConfig(isDev bool) *ds.Config {
	wrapStdout := true
	wrapStderr := true

	if os.Getenv(base.ExternalLogCaptureEnvName) != "" {
		wrapStdout = false
		wrapStderr = false
	}

	return &ds.Config{
		DomainSocketPath: base.GetDomainSocketNameForClient(isDev),
		AppName:          "",
		ModuleName:       "",
		Dev:              isDev,
		StartAsync:       false,
		LogProcessorConfig: ds.LogProcessorConfig{
			Enabled:    true,
			WrapStdout: wrapStdout,
			WrapStderr: wrapStderr,
		},
		WatchConfig: ds.WatchConfig{
			Enabled: true,
		},
		GoRoutineConfig: ds.GoRoutineConfig{
			Enabled: true,
		},
		RuntimeStatsConfig: ds.RuntimeStatsConfig{
			Enabled: true,
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

	// Create and initialize the controller
	// (collectors are now initialized inside MakeController)
	ctrlImpl, err := controller.MakeController(finalCfg)
	if err != nil {
		return err
	}

	// Store the controller in the atomic pointer
	ctrl.Store(ctrlImpl)

	return nil
}

// Shutdown shuts down Outrig
func Shutdown() {
	ctrlPtr := ctrl.Load()
	if ctrlPtr != nil {
		ctrlPtr.Shutdown()
	}
}

// GetAppRunId returns the unique identifier for the current application run
func GetAppRunId() string {
	ctrlPtr := ctrl.Load()
	if ctrlPtr != nil {
		return ctrlPtr.GetAppRunId()
	}
	return ""
}

// AppDone signals that the application is done
// This should be deferred in the program's main function
func AppDone() {
	ctrlPtr := ctrl.Load()
	if ctrlPtr != nil {
		// Send an AppDone packet
		packet := &ds.PacketType{
			Type: ds.PacketTypeAppDone,
			Data: nil, // No data needed for AppDone
		}
		ctrlPtr.SendPacket(packet)

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
	wc.RegisterWatchSync(name, lock, rval, ds.WatchFlag_Sync|ds.WatchFlag_Settable|ds.WatchFlag_Counter)
}

func WatchSync[T any](name string, lock sync.Locker, val *T) {
	if val == nil {
		return
	}
	wc := watch.GetInstance()
	rval := reflect.ValueOf(val)
	wc.RegisterWatchSync(name, lock, rval, ds.WatchFlag_Sync|ds.WatchFlag_Settable)
}

func WatchAtomicCounter[T constraints.Integer | constraints.Float](name string, val AtomicLoader[T]) {
	if val == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterWatchAtomic(name, val, ds.WatchFlag_Atomic|ds.WatchFlag_Counter)
}

func WatchAtomic[T any](name string, val AtomicLoader[T]) {
	if val == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterWatchAtomic(name, val, ds.WatchFlag_Atomic|ds.WatchFlag_Settable)
}

func WatchCounterFunc[T constraints.Integer | constraints.Float](name string, getFn func() T) {
	if getFn == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterWatchFunc(name, getFn, nil, ds.WatchFlag_Func|ds.WatchFlag_Counter)
}

func WatchFunc[T any](name string, getFn func() T, setFn func(T)) {
	if getFn == nil {
		return
	}
	wc := watch.GetInstance()
	flags := ds.WatchFlag_Func
	if setFn != nil {
		flags |= ds.WatchFlag_Settable
	}
	wc.RegisterWatchFunc(name, getFn, setFn, flags)
}

func TrackValue(name string, val any) {
	if !global.OutrigEnabled.Load() {
		return
	}
	wc := watch.GetInstance()
	rval := reflect.ValueOf(val)
	cleanName, tags := utilfn.ParseNameAndTags(name)
	wc.RecordWatchValue(cleanName, tags, nil, rval, rval.Type().String(), ds.WatchFlag_Push)
}

func TrackCounter[T constraints.Integer | constraints.Float](name string, val T) {
	if !global.OutrigEnabled.Load() {
		return
	}
	wc := watch.GetInstance()
	rval := reflect.ValueOf(val)
	cleanName, tags := utilfn.ParseNameAndTags(name)
	wc.RecordWatchValue(cleanName, tags, nil, rval, rval.Type().String(), ds.WatchFlag_Push|ds.WatchFlag_Counter)
}

// SetGoRoutineName sets a name for the current goroutine
func SetGoRoutineName(name string) {
	goId := utilfn.GetGoroutineID()
	if goId > 0 {
		gc := goroutine.GetInstance()
		gc.SetGoRoutineName(goId, name)
	}
}

// to avoid circular references, when calling internal outrig functions from the SDK
type internalOutrig struct{}

func (i *internalOutrig) SetGoRoutineName(name string) {
	SetGoRoutineName(name)
}

// OrigStdout returns the original stdout stream that was captured during initialization
func OrigStdout() *os.File {
	return loginitex.OrigStdout()
}

// OrigStderr returns the original stderr stream that was captured during initialization
func OrigStderr() *os.File {
	return loginitex.OrigStderr()
}
