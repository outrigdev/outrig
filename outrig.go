// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !no_outrig

package outrig

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/controller"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// Environment variables
const (
	DomainSocketEnvName = ds.DomainSocketEnvName
	DisabledEnvName     = ds.DisabledEnvName
	NoTelemetryEnvName  = ds.NoTelemetryEnvName
)

// Optionally re-export ds.Config so callers can do "outrig.Config" if you prefer:
type Config = ds.Config

// Integer is a constraint that permits any integer type.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Float is a constraint that permits any floating-point type.
type Float interface {
	~float32 | ~float64
}

// Number is a constraint that permits any numeric type.
type Number interface {
	Integer | Float
}

func init() {
	ioutrig.I = &internalOutrig{}
}

// Disable disables Outrig
func Disable(disconnect bool) {
	ctrlPtr := getController()
	if ctrlPtr != nil {
		ctrlPtr.Disable(disconnect)
	}
}

// Enable enables Outrig
func Enable() {
	ctrlPtr := getController()
	if ctrlPtr != nil {
		ctrlPtr.Enable()
	}
}

func Enabled() bool {
	return global.OutrigEnabled.Load()
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ds.Config {
	return config.DefaultConfig()
}

// Init initializes Outrig, returns (enabled, error)
func Init(appName string, cfgParam *ds.Config) (bool, error) {
	if cfgParam == nil {
		cfgParam = DefaultConfig()
	}
	finalCfg := *cfgParam
	if finalCfg.DomainSocketPath == "" {
		finalCfg.DomainSocketPath = base.GetDomainSocketNameForClient(finalCfg.Dev)
	}

	// Create and initialize the controller
	// (collectors are now initialized inside MakeController)
	ctrlImpl, err := controller.MakeController(appName, finalCfg)
	if err != nil {
		return Enabled(), err
	}
	// Store the controller in global.Controller
	var cif ds.Controller = ctrlImpl
	ok := global.Controller.CompareAndSwap(nil, &cif)
	if !ok {
		return Enabled(), fmt.Errorf("controller already initialized")
	}
	ctrlImpl.InitialStart()
	return Enabled(), nil
}

func getController() *controller.ControllerImpl {
	c := global.Controller.Load()
	if c == nil {
		return nil
	}
	if (*c) == nil {
		return nil
	}
	ctrlPtr := (*c).(*controller.ControllerImpl)
	return ctrlPtr
}

// Shutdown shuts down Outrig
func Shutdown() {
	ctrlPtr := getController()
	if ctrlPtr != nil {
		ctrlPtr.Shutdown()
	}
}

// GetAppRunId returns the unique identifier for the current application run
func GetAppRunId() string {
	ctrlPtr := getController()
	if ctrlPtr != nil {
		return ctrlPtr.GetAppRunId()
	}
	return ""
}

// AppDone signals that the application is done
// This should be deferred in the program's main function
func AppDone() {
	ctrlPtr := getController()
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

func WatchCounterSync[T Number](name string, lock sync.Locker, val *T) {
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

func WatchAtomicCounter[T Number](name string, val AtomicLoader[T]) {
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

func WatchCounterFunc[T Number](name string, getFn func() T) {
	if getFn == nil {
		return
	}
	wc := watch.GetInstance()
	wc.RegisterWatchFunc(name, getFn, nil, ds.WatchFlag_Func|ds.WatchFlag_Counter)
}

func WatchFunc[T any](name string, getFn func() T) {
	if getFn == nil {
		return
	}
	wc := watch.GetInstance()
	flags := ds.WatchFlag_Func
	wc.RegisterWatchFunc(name, getFn, nil, flags)
}

func TrackValue(name string, val any) {
	if !global.OutrigEnabled.Load() {
		return
	}
	wc := watch.GetInstance()
	var rval reflect.Value
	if val == nil {
		rval = reflect.Zero(reflect.TypeOf((*any)(nil)).Elem())
	} else {
		rval = reflect.ValueOf(val)
	}
	cleanName, tags := utilfn.ParseNameAndTags(name)
	wc.RecordWatchValue(cleanName, tags, nil, rval, rval.Type().String(), ds.WatchFlag_Push)
}

func TrackCounter[T Number](name string, val T) {
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

func (i *internalOutrig) Log(str string) {
	Log(str)
}

func (i *internalOutrig) Logf(format string, args ...any) {
	Logf(format, args...)
}

// OrigStdout returns the original stdout stream that was captured during initialization
func OrigStdout() *os.File {
	return loginitex.OrigStdout()
}

// OrigStderr returns the original stderr stream that was captured during initialization
func OrigStderr() *os.File {
	return loginitex.OrigStderr()
}

// semver
func OutrigVersion() string {
	return base.OutrigSDKVersion
}

func logInternal(str string) {
	ctrlPtr := getController()
	if ctrlPtr == nil {
		return
	}
	logLine := &ds.LogLine{
		Ts:     time.Now().UnixMilli(),
		Msg:    str,
		Source: "outrig",
	}
	packet := &ds.PacketType{
		Type: ds.PacketTypeLog,
		Data: logLine,
	}
	ctrlPtr.SendPacket(packet)
}

// Log sends a simple string message to the Outrig logger
func Log(str string) {
	if !global.OutrigEnabled.Load() {
		return
	}
	logInternal(str)
}

// Logf sends a formatted message to the Outrig logger
func Logf(format string, args ...any) {
	if !global.OutrigEnabled.Load() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logInternal(msg)
}
