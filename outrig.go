// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !no_outrig

package outrig

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

// Re-export ds.Config so callers can use "outrig.Config"
type Config = config.Config

type Watch struct {
	decl *ds.WatchDecl
}

type Pusher struct {
	decl     *ds.WatchDecl
	disabled bool
}

type GoRoutine struct {
	decl *ds.GoDecl
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

// SetGoRoutineName sets a name for the current goroutine
func SetGoRoutineName(name string) *GoRoutine {
	name, tags := utilfn.ParseNameAndTags(name)
	return CurrentGR().WithName(name).WithTags(tags...)
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

// WithTags adds tags to the watch. Tags can be specified with or without a "#" prefix,
// which will be stripped if present. Empty, duplicate tags are removed, and all tags are trimmed.
func (w *Watch) WithTags(tags ...string) *Watch {
	var processedTags []string
	seen := make(map[string]bool)

	for _, tag := range tags {
		// Process the tag (strip # if present and trim whitespace)
		processed := tag
		if len(tag) > 0 && tag[0] == '#' {
			processed = tag[1:]
		}
		processed = strings.TrimSpace(processed)

		// Skip empty tags and duplicates
		if processed == "" || seen[processed] {
			continue
		}

		// Add to result and mark as seen
		processedTags = append(processedTags, processed)
		seen[processed] = true
	}

	w.decl.Tags = processedTags
	// tags are validated when registering with the collector
	return w
}

func (w *Watch) AsCounter() *Watch {
	w.decl.Counter = true
	return w
}

func (w *Watch) AsJSON() *Watch {
	w.decl.Format = watch.WatchFormat_Json
	return w
}

func (w *Watch) AsStringer() *Watch {
	w.decl.Format = watch.WatchFormat_Stringer
	return w
}

func (w *Watch) AsGoFmt() *Watch {
	w.decl.Format = watch.WatchFormat_Gofmt
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
	if !w.setType(watch.WatchType_Push) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watch.WatchType_Push), false)
		return &Pusher{decl: w.decl, disabled: true}
	}
	w.registerWatch()
	return &Pusher{decl: w.decl}
}

func (p *Pusher) Push(val any) {
	if p.disabled {
		return
	}
	wc := watch.GetInstance()
	wc.PushWatchSample(p.decl.Name, val)
}

// Unregister unregisters the pusher's watch from the watch collector
func (p *Pusher) Unregister() {
	wc := watch.GetInstance()
	wc.UnregisterWatch(p.decl)
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
func (w *Watch) PollFunc(fn any) *Watch {
	if !w.setType(watch.WatchType_Func) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watch.WatchType_Func), false)
		return w
	}
	err := watch.ValidatePollFunc(fn)
	if err != nil {
		w.addConfigErr(err, true)
	} else {
		w.decl.PollObj = fn
	}
	w.registerWatch()
	return w
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
func (w *Watch) PollAtomic(val any) *Watch {
	if !w.setType(watch.WatchType_Atomic) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watch.WatchType_Atomic), false)
		return w
	}
	err := watch.ValidatePollAtomic(val)
	if err != nil {
		w.addConfigErr(err, true)
	} else {
		w.decl.PollObj = val
	}
	w.registerWatch()
	return w
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
func (w *Watch) PollSync(lock sync.Locker, val any) *Watch {
	if !w.setType(watch.WatchType_Sync) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watch.WatchType_Sync), false)
		return w
	}
	err := watch.ValidatePollSync(lock, val)
	if err != nil {
		w.addConfigErr(err, true)
	} else {
		w.decl.SyncLock = lock
		w.decl.PollObj = val
	}
	w.registerWatch()
	return w
}

// Unregister unregisters the watch from the watch collector
func (w *Watch) Unregister() {
	wc := watch.GetInstance()
	wc.UnregisterWatch(w.decl)
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

func Go(name string) *GoRoutine {
	return &GoRoutine{
		decl: &ds.GoDecl{
			Name: name,
		},
	}
}

// CurrentGR returns a GoRoutine for the current goroutine.
// If the goroutine is already registered, it returns the existing declaration.
// Otherwise, it creates a new one and registers it.
func CurrentGR() *GoRoutine {
	goId := utilfn.GetGoroutineID()
	if goId <= 0 {
		return nil
	}
	gc := goroutine.GetInstance()
	decl := gc.GetGoRoutineDecl(goId)
	if decl != nil {
		// return existing decl if it exists
		return &GoRoutine{decl: decl}
	}
	decl = &ds.GoDecl{
		GoId:    goId,
		State:   goroutine.GoState_Running,
		StartTs: time.Now().UnixMilli(),
	}
	gc.RecordGoRoutineStart(decl, nil)
	return &GoRoutine{decl: decl}
}

func (g *GoRoutine) WithTags(tags ...string) *GoRoutine {
	state := atomic.LoadInt32(&g.decl.State)
	if state == goroutine.GoState_Init {
		// For goroutines that haven't started yet, directly set the tags
		g.decl.Tags = tags
	} else {
		// For running or completed goroutines, use the collector to update tags
		gc := goroutine.GetInstance()
		gc.UpdateGoRoutineTags(g.decl, tags)
	}
	return g
}

func (g *GoRoutine) WithName(name string) *GoRoutine {
	state := atomic.LoadInt32(&g.decl.State)
	if state == goroutine.GoState_Init {
		// For goroutines that haven't started yet, directly set the name
		g.decl.Name = name
	} else {
		// For running or completed goroutines, use the collector to update name
		gc := goroutine.GetInstance()
		gc.UpdateGoRoutineName(g.decl, name)
	}
	return g
}

func (g *GoRoutine) WithoutRecover() *GoRoutine {
	if atomic.LoadInt32(&g.decl.State) != goroutine.GoState_Init {
		return g
	}
	g.decl.NoRecover = true
	return g
}

func (g *GoRoutine) Run(fn func()) {
	if atomic.LoadInt32(&g.decl.State) != goroutine.GoState_Init {
		return
	}
	atomic.StoreInt32(&g.decl.State, goroutine.GoState_Running)
	gc := goroutine.GetInstance()
	g.decl.StartTs = time.Now().UnixMilli()
	go func() {
		gc.RecordGoRoutineStart(g.decl, nil)
		if g.decl.NoRecover {
			defer func() {
				r := recover()
				if r == nil {
					gc.RecordGoRoutineEnd(g.decl, nil, false)
					return
				}
				gc.RecordGoRoutineEnd(g.decl, r, true)
				panic(r)
			}()
		} else {
			defer func() {
				// the recover will stop the panic
				gc.RecordGoRoutineEnd(g.decl, recover(), false)
			}()
		}
		fn()
	}()
}
