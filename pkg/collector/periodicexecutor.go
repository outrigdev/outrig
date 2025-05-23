// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/ioutrig"
)

type PeriodicExecutor struct {
	lock             sync.Mutex
	name             string
	done             chan struct{}
	ticker           *time.Ticker
	duration         time.Duration
	execFn           func()
	isFnRunning      atomic.Bool
	lastExecDuration atomic.Int64 // duration in milliseconds
	lastErr          error         // protected by lock
}

func MakePeriodicExecutor(name string, dur time.Duration, execFn func()) *PeriodicExecutor {
	if dur <= 0 {
		panic("duration must be greater than 0")
	}
	if execFn == nil {
		panic("execFn must not be nil")
	}
	return &PeriodicExecutor{
		name:     name,
		duration: dur,
		execFn:   execFn,
	}
}

func (p *PeriodicExecutor) IsEnabled() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.ticker != nil
}

func (p *PeriodicExecutor) Enable() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.ticker != nil {
		// already enabled
		return
	}
	doneCh := make(chan struct{})
	p.done = doneCh
	ticker := time.NewTicker(p.duration)
	p.ticker = ticker
	go func() {
		ioutrig.I.SetGoRoutineName(p.name + " #outrig")
		p.runFunc()
		for {
			select {
			case <-doneCh:
				return
			case <-ticker.C:
				p.runFunc()
			}
		}
	}()
}

func (p *PeriodicExecutor) Disable() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.ticker == nil {
		// not enabled
		return
	}
	p.ticker.Stop()
	close(p.done)
	p.ticker = nil
	p.done = nil
}

func (p *PeriodicExecutor) runFunc() {
	ok := p.isFnRunning.CompareAndSwap(false, true)
	if !ok {
		return
	}
	defer p.isFnRunning.Store(false)
	
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		p.lastExecDuration.Store(duration)
		
		if r := recover(); r != nil {
			stack := debug.Stack()
			err := fmt.Errorf("panic in %s: %v\nStack trace:\n%s", p.name, r, string(stack))
			p.setLastErr(err)
		} else {
			p.setLastErr(nil) // Clear error on successful execution
		}
	}()
	
	p.execFn()
}

// setLastErr sets the last error with proper locking
func (p *PeriodicExecutor) setLastErr(err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.lastErr = err
}

// GetLastErr returns the last error with proper locking
func (p *PeriodicExecutor) GetLastErr() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.lastErr
}

// GetLastExecDuration returns the duration of the last execution in milliseconds
func (p *PeriodicExecutor) GetLastExecDuration() int64 {
	return p.lastExecDuration.Load()
}
