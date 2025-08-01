// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilds

import (
	"sync"
	"sync/atomic"
)

// SetOnceConfig provides a thread-safe way to set a configuration value exactly once
// with fallback to a default configuration if nil is provided
type SetOnceConfig[T any] struct {
	once          sync.Once
	config        atomic.Pointer[T]
	defaultConfig T
}

// NewSetOnceConfig creates a new SetOnceConfig with the provided default configuration
// The default configuration is immediately stored and available via Get()
func NewSetOnceConfig[T any](defaultCfg T) *SetOnceConfig[T] {
	soc := &SetOnceConfig[T]{
		defaultConfig: defaultCfg,
	}
	soc.config.Store(&soc.defaultConfig)
	return soc
}

// SetOnce attempts to set the configuration value exactly once, overriding the default
// If cfg is nil, keeps the default configuration
// Returns true if the configuration was set, false if it was already set
func (soc *SetOnceConfig[T]) SetOnce(cfg *T) bool {
	var ok bool
	soc.once.Do(func() {
		if cfg != nil {
			cfgCopy := *cfg
			soc.config.Store(&cfgCopy)
		}
		ok = true
	})
	return ok
}

// Get returns the current configuration value
// Always safe to call as default is set during NewSetOnceConfig
func (soc *SetOnceConfig[T]) Get() T {
	cfg := soc.config.Load()
	return *cfg
}
