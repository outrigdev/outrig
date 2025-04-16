// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build no_outrig

package outrig

import (
	"os"
	"sync"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/ds"
)

// Environment variables
const (
	DomainSocketEnvName = ds.DomainSocketEnvName
	DisabledEnvName     = ds.DisabledEnvName
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

type AtomicLoader[T any] interface {
	Load() T
}

type AtomicStorer[T any] interface {
	Store(val T)
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
func DefaultConfig() *ds.Config {
	return &ds.Config{
		// Empty but valid config to avoid nil pointer exceptions
		LogProcessorConfig: &ds.LogProcessorConfig{
			WrapStdout: false,
			WrapStderr: false,
		},
	}
}
}

// Init is a no-op when no_outrig is set
func Init(cfgParam *ds.Config) error {
	return nil
}

// Shutdown is a no-op when no_outrig is set
func Shutdown() {}

// GetAppRunId returns an empty string when no_outrig is set
func GetAppRunId() string {
	return ""
}

// AppDone is a no-op when no_outrig is set
func AppDone() {}

// WatchCounterSync is a no-op when no_outrig is set
func WatchCounterSync[T Number](name string, lock sync.Locker, val *T) {
}

// WatchSync is a no-op when no_outrig is set
func WatchSync[T any](name string, lock sync.Locker, val *T) {}

// WatchAtomicCounter is a no-op when no_outrig is set
func WatchAtomicCounter[T Number](name string, val AtomicLoader[T]) {
}

// WatchAtomic is a no-op when no_outrig is set
func WatchAtomic[T any](name string, val AtomicLoader[T]) {}

// WatchCounterFunc is a no-op when no_outrig is set
func WatchCounterFunc[T Number](name string, getFn func() T) {}

// WatchFunc is a no-op when no_outrig is set
func WatchFunc[T any](name string, getFn func() T, setFn func(T)) {}

// TrackValue is a no-op when no_outrig is set
func TrackValue(name string, val any) {}

// TrackCounter is a no-op when no_outrig is set
func TrackCounter[T Number](name string, val T) {}

// SetGoRoutineName is a no-op when no_outrig is set
func SetGoRoutineName(name string) {}

// OrigStdout returns nil when no_outrig is set
func OrigStdout() *os.File {
	return os.Stdout
}

// OrigStderr returns nil when no_outrig is set
func OrigStderr() *os.File {
	return os.Stderr
}

// to avoid circular references, when calling internal outrig functions from the SDK
type internalOutrig struct{}

func (i *internalOutrig) SetGoRoutineName(name string) {}

// semver
func OutrigVersion() string {
	return base.OutrigSDKVersion
}
