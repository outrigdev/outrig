// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build no_outrig

package outrig

import (
	"io"
	"os"
	"sync"

	"github.com/outrigdev/outrig/pkg/config"
)

// Re-export config.Config so callers can use "outrig.Config"
type Config = config.Config

type Watch struct {
	// No actual implementation needed for no_outrig build
}

type Pusher struct {
	// No actual implementation needed for no_outrig build
}

type GoRoutine struct {
	// No actual implementation needed for no_outrig build
}

// Disable is a no-op when no_outrig is set
func Disable(disconnect bool) {}

// Enable is a no-op when no_outrig is set
func Enable() {}

// Enabled always returns false when no_outrig is set
func Enabled() bool {
	return false
}

// DefaultConfig returns an empty config when no_outrig is set
func DefaultConfig() *config.Config {
	// Empty but valid config to avoid nil pointer exceptions
	return &config.Config{}
}

// Init is a no-op when no_outrig is set
func Init(appName string, cfgParam *config.Config) (bool, error) {
	return false, nil
}

// Shutdown is a no-op when no_outrig is set
func Shutdown() {}

// GetAppRunId returns an empty string when no_outrig is set
func GetAppRunId() string {
	return ""
}

// AppDone is a no-op when no_outrig is set
func AppDone() {}

// NewWatch creates a new Watch with the given name
// This is a no-op implementation for no_outrig build
func NewWatch(name string) *Watch {
	return &Watch{}
}

// WithTags adds tags to the watch
// This is a no-op implementation for no_outrig build
func (w *Watch) WithTags(tags ...string) *Watch {
	return w
}

// AsCounter marks the watch as a counter
// This is a no-op implementation for no_outrig build
func (w *Watch) AsCounter() *Watch {
	return w
}

// AsJSON sets the watch format to JSON
// This is a no-op implementation for no_outrig build
func (w *Watch) AsJSON() *Watch {
	return w
}

// AsStringer sets the watch format to use the String() method
// This is a no-op implementation for no_outrig build
func (w *Watch) AsStringer() *Watch {
	return w
}

// AsGoFmt sets the watch format to use Go's %#v format
// This is a no-op implementation for no_outrig build
func (w *Watch) AsGoFmt() *Watch {
	return w
}

// ForPush creates a pusher for this watch
// This is a no-op implementation for no_outrig build
func (w *Watch) ForPush() *Pusher {
	return &Pusher{}
}

// PollFunc sets up a function-based watch
// This is a no-op implementation for no_outrig build
func (w *Watch) PollFunc(fn any) *Watch {
	return w
}

// PollAtomic sets up an atomic-based watch
// This is a no-op implementation for no_outrig build
func (w *Watch) PollAtomic(val any) *Watch {
	return w
}

// PollSync sets up a synchronization-based watch
// This is a no-op implementation for no_outrig build
func (w *Watch) PollSync(lock sync.Locker, val any) *Watch {
	return w
}

// Static sets up a static watch that holds a constant value
// This is a no-op implementation for no_outrig build
func (w *Watch) Static(val any) *Watch {
	return w
}

// Unregister unregisters the watch
// This is a no-op implementation for no_outrig build
func (w *Watch) Unregister() {
	// No-op
}

// Push pushes a value to the watch
// This is a no-op implementation for no_outrig build
func (p *Pusher) Push(val any) {
	// No-op
}

// Unregister unregisters the pusher's watch
// This is a no-op implementation for no_outrig build
func (p *Pusher) Unregister() {
	// No-op
}

// SetGoRoutineName is a no-op when no_outrig is set
func SetGoRoutineName(name string) *GoRoutine {
	return &GoRoutine{}
}

// Go creates a new GoRoutine with the given name
// This is a no-op implementation for no_outrig build
func Go(name string) *GoRoutine {
	return &GoRoutine{}
}

// CurrentGR returns a GoRoutine for the current goroutine
// This is a no-op implementation for no_outrig build
func CurrentGR() *GoRoutine {
	return &GoRoutine{}
}

// WithTags adds tags to the goroutine
// This is a no-op implementation for no_outrig build
func (g *GoRoutine) WithTags(tags ...string) *GoRoutine {
	return g
}

// WithName sets the name of the goroutine
// This is a no-op implementation for no_outrig build
func (g *GoRoutine) WithName(name string) *GoRoutine {
	return g
}

// WithGroup sets the group of the goroutine
// This is a no-op implementation for no_outrig build
func (g *GoRoutine) WithGroup(group string) *GoRoutine {
	return g
}

// WithPkg sets the package name of the goroutine
// This is a no-op implementation for no_outrig build
func (g *GoRoutine) WithPkg(pkg string) *GoRoutine {
	return g
}

// WithoutRecover disables panic recovery for the goroutine
// This is a no-op implementation for no_outrig build
func (g *GoRoutine) WithoutRecover() *GoRoutine {
	return g
}

// Run executes the given function in a new goroutine
// This is a no-op implementation for no_outrig build
func (g *GoRoutine) Run(fn func()) {
	go fn()
}

func OrigStdout() *os.File {
	return os.Stdout
}

func OrigStderr() *os.File {
	return os.Stderr
}

// to avoid circular references, when calling internal outrig functions from the SDK
type internalOutrig struct{}

func (i *internalOutrig) SetGoRoutineName(name string) {}

// semver
func OutrigVersion() string {
	return config.OutrigSDKVersion
}

// Log is a no-op when no_outrig is set
func Log(str string) {}

// Logf is a no-op when no_outrig is set
func Logf(format string, args ...any) {}

func MakeLogStream(name string) io.Writer {
	return io.Discard
}
