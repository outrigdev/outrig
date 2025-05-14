// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !no_outrig

package outrig

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
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

const (
	watchFormat_Json     = "json"
	watchFormat_Stringer = "stringer"
	watchFormat_Gofmt    = "gofmt"

	watchType_Sync   = "sync"
	watchType_Atomic = "atomic"
	watchType_Func   = "func"
	watchType_Push   = "push"
)

// Re-export ds.Config so callers can do "outrig.Config" if you prefer:
type Config = config.Config

// Number is a constraint that permits any numeric type.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

type Watch struct {
	decl *ds.WatchDecl
}

type Pusher struct {
	decl     *ds.WatchDecl
	disabled bool
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
func DefaultConfig() *config.Config {
	return config.DefaultConfig()
}

// Init initializes Outrig, returns (enabled, error)
func Init(appName string, cfgParam *config.Config) (bool, error) {
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

// MakeLogStream creates an io.Writer that sends written data as log lines to Outrig
// The name parameter specifies the source of the logs
func MakeLogStream(name string) (io.Writer, error) {
	// Create a log stream writer using the logprocess package
	return logprocess.MakeLogStreamWriter(name), nil
}

func NewWatch(name string) *Watch {
	w := &Watch{
		decl: &ds.WatchDecl{
			Name:    name,
			NewLine: getCallerInfo(1),
		},
	}
	// names are validated when registering with the collector
	return w
}

func (w *Watch) WithTags(tags ...string) *Watch {
	w.decl.Tags = tags
	// tags are validated when registering with the collector
	return w
}

func (w *Watch) AsCounter() *Watch {
	w.decl.Counter = true
	return w
}

func (w *Watch) AsJSON() *Watch {
	w.decl.Format = watchFormat_Json
	return w
}

func (w *Watch) AsStringer() *Watch {
	w.decl.Format = watchFormat_Stringer
	return w
}

func (w *Watch) AsGoFmt() *Watch {
	w.decl.Format = watchFormat_Gofmt
	return w
}

func (w *Watch) setType(typ string) bool {
	if w.decl.WatchType != "" {
		return false
	}
	w.decl.WatchType = typ
	return true
}

func (w *Watch) addConfigErr(err error, invalide bool) {
	if invalide {
		w.decl.Invalid = true
	}
	errCtx := ds.ErrWithContext{
		Ref:   w.decl.Name,
		Error: err.Error(),
		Line:  getCallerInfo(2),
	}
	wc := watch.GetInstance()
	wc.AddRegError(errCtx)
}

func (w *Watch) registerWatch() {
	wc := watch.GetInstance()
	wc.RegisterWatchDecl(w.decl)
}

func (w *Watch) ForPush() *Pusher {
	if !w.setType(watchType_Push) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watchType_Push), false)
		return &Pusher{decl: w.decl, disabled: true}
	}
	w.registerWatch()
	return &Pusher{decl: w.decl}
}

func (p *Pusher) Push(val any) {
	if p.disabled {
		return
	}
	// todo send push value
}

// PollFunc sets up a function-based watch that periodically calls the provided function
// to retrieve its current value. The provided function must take no arguments and must return
// exactly one value (of any type).
//
// Requirements for fn:
//   - Non-nil function
//   - Zero arguments
//   - Exactly one return value
//
// Example:
//
//	outrig.NewWatch("counter").PollFunc(func() int { return myCounter })
func (w *Watch) PollFunc(fn any) {
	if !w.setType(watchType_Func) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watchType_Func), false)
		return
	}
	err := watch.ValidatePollFunc(fn)
	if err != nil {
		w.addConfigErr(err, true)
	} else {
		w.decl.PollObj = fn
	}
	w.registerWatch()
}

// PollAtomic sets up an atomic-based watch that reads values from atomic variables.
// This method is optimized for monitoring values that are updated using atomic operations.
//
// The val parameter must be:
//   - A non-nil pointer
//   - A pointer to one of the following types:
//   - sync/atomic package types (atomic.Bool, atomic.Int32, atomic.Int64, etc.)
//   - Primitive types that support atomic operations (int32, int64, uint32, uint64, uintptr)
//   - unsafe.Pointer
//
// Example:
//
//	var counter atomic.Int64
//	outrig.NewWatch("atomic-counter").PollAtomic(&counter)
func (w *Watch) PollAtomic(val any) {
	if !w.setType(watchType_Atomic) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watchType_Atomic), false)
		return
	}
	err := watch.ValidatePollAtomic(val)
	if err != nil {
		w.addConfigErr(err, true)
	} else {
		w.decl.PollObj = val
	}
	w.registerWatch()
}

// PollSync sets up a synchronization-based watch to monitor values protected by a mutex or other locker.
// It's intended for values updated concurrently, ensuring thread-safe access during polling.
//
// Requirements:
//   - lock: A non-nil sync.Locker (e.g., *sync.Mutex, *sync.RWMutex)
//   - val: A non-nil pointer to the value being watched
//
// Example:
//
//	var mu sync.Mutex
//	var counter int
//	outrig.NewWatch("sync-counter").PollSync(&mu, &counter)
func (w *Watch) PollSync(lock sync.Locker, val any) {
	if !w.setType(watchType_Sync) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watchType_Sync), false)
		return
	}
	err := watch.ValidatePollSync(lock, val)
	if err != nil {
		w.addConfigErr(err, true)
	} else {
		w.decl.SyncLock = lock
		w.decl.PollObj = val
	}
	w.registerWatch()
}

// getCallerInfo returns the file and line number of the caller.
// The skip parameter specifies how many stack frames to skip before reporting.
// A skip value of 0 returns the file and line number of the getCallerInfo call itself.
// A skip value of 1 returns the file and line number of the function that called getCallerInfo.
// Higher values continue up the call stack.
// If the requested stack frame doesn't exist, it returns an empty string.
func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", file, line)
}
