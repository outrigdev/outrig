// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/collector"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const MaxWatchVals = 10000

const (
	WatchFormat_Json     = "json"
	WatchFormat_Stringer = "stringer"
	WatchFormat_Gofmt    = "gofmt"

	WatchType_Sync   = "sync"
	WatchType_Atomic = "atomic"
	WatchType_Func   = "func"
	WatchType_Push   = "push"
)

// WatchCollector implements the collector.Collector interface for watch collection
type WatchCollector struct {
	lock              sync.Mutex
	executor          *collector.PeriodicExecutor
	controller        ds.Controller
	config            config.WatchConfig
	watchDecls        map[string]*ds.WatchDecl
	pushSamples       []ds.WatchSample
	lastWatchSamples  map[string]ds.WatchSample // last set of watch values for delta calculation
	nextSendFull      bool                      // true for full update, false for delta update
	regErrors         []ds.ErrWithContext       // errors encountered during watch registration
	regErrorsDeltaIdx int
	newDecls          []ds.WatchDecl // new declarations added since last delta
}

// CollectorName returns the unique name of the collector
func (wc *WatchCollector) CollectorName() string {
	return "watch"
}

// singleton instance
var instance *WatchCollector
var instanceOnce sync.Once

// GetInstance returns the singleton instance of WatchCollector
func GetInstance() *WatchCollector {
	instanceOnce.Do(func() {
		instance = &WatchCollector{
			watchDecls:       make(map[string]*ds.WatchDecl),
			lastWatchSamples: make(map[string]ds.WatchSample),
			nextSendFull:     true, // First send is always a full update
			regErrors:        make([]ds.ErrWithContext, 0),
		}
		instance.executor = collector.MakePeriodicExecutor("WatchCollector", 1*time.Second, instance.CollectWatches)
	})
	return instance
}

func (wc *WatchCollector) UnregisterWatch(decl *ds.WatchDecl) {
	wc.lock.Lock()
	defer wc.lock.Unlock()

	// Create a new decl with just the name and Unregistered set to true
	unregDecl := ds.WatchDecl{
		Name:         decl.Name,
		Unregistered: true,
	}

	// Add to newDecls to track the unregistration
	wc.newDecls = append(wc.newDecls, unregDecl)

	// Remove from watchDecls map
	delete(wc.watchDecls, decl.Name)
}

// RegisterWatchDecl registers a watch declaration in the watchDecls map
// Returns an error if a watch with the same name already exists
func (wc *WatchCollector) RegisterWatchDecl(decl *ds.WatchDecl) {
	wc.lock.Lock()
	defer wc.lock.Unlock()

	if decl == nil || decl.Name == "" {
		err := fmt.Errorf("cannot register a watch with nil or empty name")
		wc.regErrors = append(wc.regErrors, ds.ErrWithContext{
			Error: err.Error(),
			Line:  decl.NewLine,
		})
		return
	}

	// Check if a watch with this name already exists
	if _, exists := wc.watchDecls[decl.Name]; exists {
		err := fmt.Errorf("cannot register watch with duplicate name %q", decl.Name)
		wc.regErrors = append(wc.regErrors, ds.ErrWithContext{
			Error: err.Error(),
			Line:  decl.NewLine,
		})
		return
	}

	// Register the watch declaration
	wc.watchDecls[decl.Name] = decl
}

func (wc *WatchCollector) AddRegError(err ds.ErrWithContext) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	wc.regErrors = append(wc.regErrors, err)
}

// GetRegErrors returns a copy of the registration errors
func (wc *WatchCollector) GetRegErrors() []ds.ErrWithContext {
	wc.lock.Lock()
	defer wc.lock.Unlock()

	// Create a copy to avoid race conditions
	result := make([]ds.ErrWithContext, len(wc.regErrors))
	copy(result, wc.regErrors)
	return result
}

// getSendFullAndReset returns the current sendFull value and always sets it to false
func (wc *WatchCollector) getSendFullAndReset() bool {
	wc.lock.Lock()
	defer wc.lock.Unlock()

	sendFull := wc.nextSendFull
	wc.nextSendFull = false // Always set to false after getting the value
	return sendFull
}

// SetNextSendFull sets the nextSendFull flag to force a full update on the next dump
func (wc *WatchCollector) SetNextSendFull(full bool) {
	wc.lock.Lock()
	defer wc.lock.Unlock()

	wc.nextSendFull = full
}

// InitCollector initializes the watch collector with a controller and configuration
func (wc *WatchCollector) InitCollector(controller ds.Controller, cfg any, arCtx ds.AppRunContext) error {
	wc.controller = controller
	if watchConfig, ok := cfg.(config.WatchConfig); ok {
		wc.config = watchConfig
	}
	return nil
}

// Enable is called when the collector should start collecting data
func (wc *WatchCollector) Enable() {
	wc.executor.Enable()
}

// Disable stops the collector
func (wc *WatchCollector) Disable() {
	wc.executor.Disable()
}

func (wc *WatchCollector) GetWatchNames() []string {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	names := make([]string, 0, len(wc.watchDecls))
	for name := range wc.watchDecls {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (wc *WatchCollector) getWatchDecl(name string) *ds.WatchDecl {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	return wc.watchDecls[name]
}

func (wc *WatchCollector) PushWatchSample(name string, val any) {
	decl := wc.getWatchDecl(name)
	if decl == nil {
		return
	}
	sample := wc.newWatchSample(decl, reflect.ValueOf(val), 0)
	if sample == nil {
		return
	}
	wc.lock.Lock()
	defer wc.lock.Unlock()
	wc.pushSamples = append(wc.pushSamples, *sample)
}

func (wc *WatchCollector) getAndClearPushSamples() []ds.WatchSample {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	watchVals := wc.pushSamples
	wc.pushSamples = nil
	return watchVals
}

func (wc *WatchCollector) getLastSample(name string) (ds.WatchSample, bool) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	lastSample, exists := wc.lastWatchSamples[name]
	return lastSample, exists
}

// computeDeltaWatch compares current and last watch sample and returns a delta sample
// For delta updates, we start with a full copy of current and clear fields that haven't changed
func (wc *WatchCollector) computeDeltaWatch(name string, current ds.WatchSample) (ds.WatchSample, bool) {
	// For push values, we always include all fields and don't compute deltas
	decl := wc.getWatchDecl(name)
	if decl.WatchType == WatchType_Push {
		return current, false
	}
	lastSample, exists := wc.getLastSample(name)
	if !exists {
		// New watch, include all fields
		return current, false
	}
	deltaSample := current

	// Check if all the fields that should be compared are the same
	sameKind := current.Kind == lastSample.Kind
	sameType := current.Type == lastSample.Type
	sameVal := current.Val == lastSample.Val
	sameError := current.Error == lastSample.Error
	sameAddr := slices.Equal(current.Addr, lastSample.Addr)
	sameCap := current.Cap == lastSample.Cap
	sameLen := current.Len == lastSample.Len

	// If all fields are the same, set Same to true and clear the fields
	sameValue := sameKind && sameType && sameVal && sameError && sameAddr && sameCap && sameLen
	if sameValue {
		deltaSample.Same = true
		// Clear the fields that are the same as the previous sample
		deltaSample.Kind = 0
		deltaSample.Type = ""
		deltaSample.Val = ""
		deltaSample.Error = ""
		deltaSample.Addr = nil
		deltaSample.Cap = 0
		deltaSample.Len = 0
	}
	return deltaSample, sameValue
}

func (wc *WatchCollector) getDeclList(delta bool) []ds.WatchDecl {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	if !delta {
		wc.newDecls = nil
		declList := make([]ds.WatchDecl, 0, len(wc.watchDecls))
		for _, decl := range wc.watchDecls {
			declList = append(declList, *decl)
		}
		return declList
	}
	// Return only the new declarations since the last delta
	declList := wc.newDecls
	wc.newDecls = nil
	return declList
}

func (wc *WatchCollector) getRegErrors(delta bool) []ds.ErrWithContext {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	if !delta {
		wc.regErrorsDeltaIdx = len(wc.regErrors)
		return wc.regErrors
	}
	// Return only the new errors since the last delta
	return wc.regErrors[wc.regErrorsDeltaIdx:]
}

// CollectWatches collects watch information and sends it to the controller
// note we do not hold the lock for the duration of this function
func (wc *WatchCollector) CollectWatches() {
	if !global.OutrigEnabled.Load() || wc.controller == nil {
		return
	}
	var samples []ds.WatchSample
	sendFull := wc.getSendFullAndReset()
	watchNames := wc.GetWatchNames()
	for _, name := range watchNames {
		watchDecl := wc.getWatchDecl(name)
		if watchDecl == nil {
			continue
		}
		sample := wc.collectWatch2(watchDecl)
		if sample == nil {
			continue
		}
		samples = append(samples, *sample)
	}
	numSameValue := 0
	currentWatchValues := make(map[string]ds.WatchSample)
	// Process each watch value for delta calculation
	for idx, watch := range samples {
		// Store current watch value for next delta calculation
		currentWatchValues[watch.Name] = watch
		if sendFull {
			continue
		}
		deltaWatch, sameValue := wc.computeDeltaWatch(watch.Name, watch)
		if sameValue {
			numSameValue++
		}
		samples[idx] = deltaWatch
	}
	// Update the last watch values for next delta calculation
	wc.setLastWatchSamples(currentWatchValues)
	pushWatchVals := wc.getAndClearPushSamples()
	samples = append(samples, pushWatchVals...)

	watchInfo := &ds.WatchInfo{
		Ts:        time.Now().UnixMilli(),
		Delta:     !sendFull,
		Decls:     wc.getDeclList(!sendFull),
		Watches:   samples,
		RegErrors: wc.getRegErrors(!sendFull),
	}

	// Send the watch packet
	pk := &ds.PacketType{
		Type: ds.PacketTypeWatch,
		Data: watchInfo,
	}

	wc.controller.SendPacket(pk)
}

const MaxWatchWaitTime = 10 * time.Millisecond

// watchSampleErr creates a WatchSample with an error message
func watchSampleErr(decl *ds.WatchDecl, startTime time.Time, errMsg string) *ds.WatchSample {
	pollDur := time.Since(startTime).Microseconds()
	return &ds.WatchSample{
		Name:    decl.Name,
		Ts:      time.Now().UnixMilli(),
		PollDur: pollDur,
		Error:   errMsg,
	}
}

// getAtomicValue extracts the value from an atomic variable
func getAtomicValue(atomicVal any) (reflect.Value, error) {
	atomicValue := reflect.ValueOf(atomicVal)

	// Check if it's a pointer
	if atomicValue.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("atomic value must be a pointer")
	}

	// First try to use the Load() method (for atomic package types)
	loadMethod := atomicValue.MethodByName("Load")
	if loadMethod.IsValid() {
		results := loadMethod.Call(nil)
		if len(results) > 0 {
			return results[0], nil
		}
		return reflect.Value{}, fmt.Errorf("atomic Load method returned no values")
	}

	// If no Load() method, check if it's a primitive type that supports atomic operations
	elemType := atomicValue.Type().Elem()
	elemKind := elemType.Kind()

	switch elemKind {
	case reflect.Int32:
		if ptr, ok := atomicVal.(*int32); ok {
			val := atomic.LoadInt32(ptr)
			return reflect.ValueOf(val), nil
		}
	case reflect.Int64:
		if ptr, ok := atomicVal.(*int64); ok {
			val := atomic.LoadInt64(ptr)
			return reflect.ValueOf(val), nil
		}
	case reflect.Uint32:
		if ptr, ok := atomicVal.(*uint32); ok {
			val := atomic.LoadUint32(ptr)
			return reflect.ValueOf(val), nil
		}
	case reflect.Uint64:
		if ptr, ok := atomicVal.(*uint64); ok {
			val := atomic.LoadUint64(ptr)
			return reflect.ValueOf(val), nil
		}
	case reflect.Uintptr:
		if ptr, ok := atomicVal.(*uintptr); ok {
			val := atomic.LoadUintptr(ptr)
			return reflect.ValueOf(val), nil
		}
	}

	return reflect.Value{}, fmt.Errorf("unsupported atomic type: %s", elemType.String())
}

func (wc *WatchCollector) collectWatch2(decl *ds.WatchDecl) *ds.WatchSample {
	startTime := time.Now()

	if decl == nil || decl.Invalid {
		return nil
	}

	var rval reflect.Value
	var err error

	switch decl.WatchType {
	case WatchType_Sync:
		locked, waitDuration := utilfn.TryLockWithTimeout(decl.SyncLock, MaxWatchWaitTime)
		if !locked {
			return watchSampleErr(decl, startTime, fmt.Sprintf("timeout waiting for lock after %v", waitDuration))
		}
		defer decl.SyncLock.Unlock()
		rval = reflect.ValueOf(decl.PollObj)

	case WatchType_Atomic:
		rval, err = getAtomicValue(decl.PollObj)
		if err != nil {
			return watchSampleErr(decl, startTime, err.Error())
		}

	case WatchType_Func:
		fnValue := reflect.ValueOf(decl.PollObj)
		results := fnValue.Call(nil)
		if len(results) == 0 {
			return watchSampleErr(decl, startTime, "function returned no values")
		}
		rval = results[0]

	case WatchType_Push:
		return nil

	default:
		return nil
	}

	pollDur := time.Since(startTime).Microseconds()
	return wc.newWatchSample(decl, rval, pollDur)
}

func (wc *WatchCollector) newWatchSample(decl *ds.WatchDecl, rval reflect.Value, pollDur int64) *ds.WatchSample {
	sample := ds.WatchSample{
		Name:    decl.Name,
		Ts:      time.Now().UnixMilli(),
		PollDur: pollDur,
	}
	if !rval.IsValid() {
		sample.Val = "nil"
		sample.Kind = int(reflect.Invalid)
		sample.Type = "nil"
		return &sample
	}
	sample.Type = rval.Type().String()
	const maxPtrDepth = 10
	for depth := 0; rval.Kind() == reflect.Ptr && depth < maxPtrDepth; depth++ {
		if rval.IsNil() {
			sample.Val = "nil"
			sample.Kind = int(reflect.Invalid)
			return &sample
		}
		sample.Addr = append(sample.Addr, fmt.Sprintf("%p", rval.Interface()))
		rval = rval.Elem()
	}
	sample.Kind = int(rval.Kind())
	switch rval.Kind() {
	case reflect.String:
		sample.Val = rval.String()
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		sample.Val = fmt.Sprint(rval.Interface())
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct, reflect.Interface:
		if (rval.Kind() == reflect.Interface || rval.Kind() == reflect.Slice || rval.Kind() == reflect.Map) && rval.IsNil() {
			sample.Val = "nil"
			return &sample
		}
		var err error
		sample.Val, err = formatWatchValue(decl, rval)
		if err != nil {
			sample.Error = err.Error()
		}
		if rval.Kind() == reflect.Slice || rval.Kind() == reflect.Array || rval.Kind() == reflect.Map {
			sample.Len = rval.Len()
		}
		if rval.Kind() == reflect.Slice {
			sample.Cap = rval.Cap()
		}
	case reflect.Chan:
		if rval.IsNil() {
			sample.Val = "nil"
		} else {
			sample.Val = fmt.Sprintf("(chan:%p)", rval.Interface())
		}
		sample.Len = rval.Len()
		sample.Cap = rval.Cap()
	case reflect.Func:
		if rval.IsNil() {
			sample.Val = "nil"
		} else {
			sample.Val = fmt.Sprintf("(func:%p)", rval.Interface())
		}
	case reflect.UnsafePointer, reflect.Ptr:
		sample.Val = fmt.Sprintf("%p", rval.Interface())
	default:
		sample.Error = fmt.Sprintf("unsupported kind: %s", rval.Kind())
	}
	return &sample
}

func formatWatchValue(decl *ds.WatchDecl, rval reflect.Value) (string, error) {
	if decl.Format == "" {
		// default to JSON, but fallback to %#v if JSON fails
		barr, err := json.Marshal(rval.Interface())
		if err == nil {
			return string(barr), nil
		}
		return fmt.Sprintf("%#v", rval.Interface()), nil
	}
	if decl.Format == WatchFormat_Json {
		barr, err := json.Marshal(rval.Interface())
		if err == nil {
			return string(barr), nil
		}
		return "", fmt.Errorf("json.Marshal: %w", err)
	} else if decl.Format == WatchFormat_Stringer {
		return fmt.Sprint(rval.Interface()), nil
	} else if decl.Format == WatchFormat_Gofmt {
		return fmt.Sprintf("%#v", rval.Interface()), nil
	} else {
		return "", fmt.Errorf("unsupported format: %s", decl.Format)
	}
}

func (wc *WatchCollector) setLastWatchSamples(watches map[string]ds.WatchSample) {
	wc.lock.Lock()
	defer wc.lock.Unlock()

	// Update last watch values with current ones
	for name, watch := range watches {
		wc.lastWatchSamples[name] = watch
	}

	// Remove watches that no longer exist
	for name := range wc.lastWatchSamples {
		if _, found := watches[name]; !found {
			delete(wc.lastWatchSamples, name)
		}
	}
}
