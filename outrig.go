// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !no_outrig

package outrig

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/goid"
	"github.com/outrigdev/outrig/pkg/collector/goroutine"
	"github.com/outrigdev/outrig/pkg/collector/loginitex"
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
	"github.com/outrigdev/outrig/pkg/collector/runtimestats"
	"github.com/outrigdev/outrig/pkg/collector/watch"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/controller"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

var initOnce sync.Once

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
	// copy to avoid cross contamination with original
	finalCfg := *cfgParam

	var initErr error
	var wasFirstCall bool

	initOnce.Do(func() {
		wasFirstCall = true

		// init/register the collectors
		logprocess.Init(&finalCfg.LogProcessorConfig)
		goroutine.Init(&finalCfg.GoRoutineConfig)
		watch.Init(&finalCfg.WatchConfig)
		runtimestats.Init(&finalCfg.RuntimeStatsConfig)

		// Create and initialize the controller
		// (collectors are now initialized inside MakeController)
		ctrlImpl, err := controller.MakeController(appName, finalCfg)
		if err != nil {
			initErr = err
			return
		}
		// Store the controller in global.Controller
		var cif ds.Controller = ctrlImpl
		global.Controller.Store(&cif)
		ctrlImpl.InitialStart()
	})

	if !wasFirstCall {
		// This is a subsequent call to Init()
		if !finalCfg.Quiet && global.OutrigAutoInit.Load() {
			fmt.Printf("#outrig Warning: outrig.Init() called after importing github.com/outrigdev/outrig/autoinit; new config ignored. To customize settings, remove the autoinit import.\n")
		}
		return Enabled(), fmt.Errorf("controller already initialized")
	}

	if initErr != nil {
		return Enabled(), initErr
	}
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
	return config.GetAppRunId()
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
	return CurrentGR().WithName(name)
}

// to avoid circular references, when calling internal outrig functions from the SDK
type internalOutrig struct{}

func (i *internalOutrig) SetGoRoutineNameAndTags(name string, tags ...string) {
	CurrentGR().WithName(name).WithTags(tags...)
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
	return config.OutrigSDKVersion
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
// This log stream will never block your code for I/O. When Outrig is disabled, it discards the data after
// a simple atomic.Bool check (nanoseconds).  When Outrig is enabled it uses a non-blocking write to a
// buffered channel.
func MakeLogStream(name string) io.Writer {
	// Create a log stream writer using the logprocess package
	return logprocess.MakeLogStreamWriter(name)
}

func NewWatch(name string) *Watch {
	w := &Watch{
		decl: &ds.WatchDecl{
			Name:    utilfn.NormalizeName(name),
			NewLine: getCallerInfo(1),
		},
	}
	// names are validated when registering with the collector
	return w
}

// WithTags adds tags to the watch. Tags can be specified with or without a "#" prefix,
// which will be stripped if present. Empty, duplicate tags are removed, and all tags are trimmed.
func (w *Watch) WithTags(tags ...string) *Watch {
	w.decl.Tags = utilfn.CleanTagSlice(tags)
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

func (w *Watch) addConfigErr(err error, invalid bool) {
	if invalid {
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

// Static sets up a static watch that holds a constant value. The value is set once
// when the watch is created and never changes. This is useful for configuration
// values, URLs, or other constants that you want to monitor but don't need to poll.
//
// Example:
//
//	outrig.NewWatch("game-url").Static("http://localhost:8080")
func (w *Watch) Static(val any) *Watch {
	if !w.setType(watch.WatchType_Static) {
		w.addConfigErr(fmt.Errorf("cannot change watch type from %s to %s", w.decl.WatchType, watch.WatchType_Static), false)
		return w
	}
	w.registerWatch()

	// Immediately push the static value since it won't be polled
	wc := watch.GetInstance()
	wc.PushWatchSample(w.decl.Name, val)

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

func getCallerCreatedByInfo(skip int) string {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return fmt.Sprintf("%s:%d", file, line)
	}
	offset := pc - fn.Entry()
	goID := goid.Get()
	return fmt.Sprintf("created by %s in goroutine %d\n\t%s:%d +0x%x", fn.Name(), goID, file, line, offset)
}

func Go(name string) *GoRoutine {
	return &GoRoutine{
		decl: &ds.GoDecl{
			Name: utilfn.NormalizeName(name),
		},
	}
}

// CurrentGR returns a GoRoutine for the current goroutine.
// If the goroutine is already registered, it returns the existing declaration.
// Otherwise, it creates a new one and registers it.
func CurrentGR() *GoRoutine {
	goId := int64(goid.Get())
	if goId <= 0 {
		return nil
	}
	gc := goroutine.GetInstance()
	decl := gc.GetGoRoutineDecl(goId)
	if decl != nil {
		// return existing decl if it exists
		return &GoRoutine{decl: decl}
	}
	decl = goroutine.NewRunningGoDecl(goId)
	decl.StartTs = time.Now().UnixMilli()
	gc.RecordGoRoutineStart(decl, nil)
	return &GoRoutine{decl: decl}
}

func (g *GoRoutine) WithTags(tags ...string) *GoRoutine {
	cleanedTags := utilfn.CleanTagSlice(tags)
	state := atomic.LoadInt32(&g.decl.State)
	if state == goroutine.GoState_Init {
		// For goroutines that haven't started yet, directly set the tags
		g.decl.Tags = cleanedTags
	} else {
		// For running or completed goroutines, use the collector to update tags
		gc := goroutine.GetInstance()
		gc.UpdateGoRoutineTags(g.decl, cleanedTags)
	}
	return g
}

func (g *GoRoutine) WithName(name string) *GoRoutine {
	normalizedName := utilfn.NormalizeName(name)
	state := atomic.LoadInt32(&g.decl.State)
	if state == goroutine.GoState_Init {
		// For goroutines that haven't started yet, directly set the name
		g.decl.Name = normalizedName
	} else {
		// For running or completed goroutines, use the collector to update name
		gc := goroutine.GetInstance()
		gc.UpdateGoRoutineName(g.decl, normalizedName)
	}
	return g
}

func (g *GoRoutine) WithPkg(pkg string) *GoRoutine {
	state := atomic.LoadInt32(&g.decl.State)
	if state == goroutine.GoState_Init {
		// For goroutines that haven't started yet, directly set the package
		g.decl.Pkg = pkg
	} else {
		// For running or completed goroutines, use the collector to update package
		gc := goroutine.GetInstance()
		gc.UpdateGoRoutinePkg(g.decl, pkg)
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
	if !atomic.CompareAndSwapInt32(&g.decl.State, goroutine.GoState_Init, goroutine.GoState_Running) {
		return
	}
	gc := goroutine.GetInstance()
	g.decl.StartTs = time.Now().UnixMilli()
	g.decl.RealCreatedBy = getCallerCreatedByInfo(1)
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
