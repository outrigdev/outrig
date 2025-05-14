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

type AtomicLoader[T any] interface {
	Load() T
}

type AtomicStorer[T any] interface {
	Store(val T)
}

// WatchCollector implements the collector.Collector interface for watch collection
type WatchCollector struct {
	lock            sync.Mutex
	executor        *collector.PeriodicExecutor
	controller      ds.Controller
	config          config.WatchConfig
	watchDecls2     map[string]*ds.WatchDecl
	watchDecls      map[string]*WatchDecl
	watchVals       []ds.WatchSample
	lastWatchValues map[string]ds.WatchSample // last set of watch values for delta calculation
	nextSendFull    bool                      // true for full update, false for delta update
	regErrors       []ds.ErrWithLine         // errors encountered during watch registration
}

type WatchDecl struct {
	Name      string
	Tags      []string
	Flags     int           // denotes the type of watch (Sync, Func, Atomic)
	Lock      sync.Locker   // for Sync
	PtrVal    reflect.Value // for Sync
	GetFn     any           // for Func
	SetFn     any           // for Func
	HookFn    any           // for Hook
	AtomicVal any           // for Atomic (AtomicLoader)
	HookSent  atomic.Bool
}

func (d *WatchDecl) IsSync() bool {
	return d.Flags&ds.WatchFlag_Sync != 0
}

func (d *WatchDecl) IsFunc() bool {
	return d.Flags&ds.WatchFlag_Func != 0
}

func (d *WatchDecl) IsAtomic() bool {
	return d.Flags&ds.WatchFlag_Atomic != 0
}

func (d *WatchDecl) IsHook() bool {
	return d.Flags&ds.WatchFlag_Hook != 0
}

func (d *WatchDecl) IsNumeric() bool {
	kind := reflect.Kind(d.Flags & ds.KindMask)
	switch kind {
	case reflect.Bool:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return true
	default:
		return false
	}
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
			watchDecls:      make(map[string]*WatchDecl),
			watchDecls2:     make(map[string]*ds.WatchDecl),
			lastWatchValues: make(map[string]ds.WatchSample),
			nextSendFull:    true, // First send is always a full update
			regErrors:       make([]ds.ErrWithLine, 0),
		}
		instance.executor = collector.MakePeriodicExecutor("WatchCollector", 1*time.Second, instance.CollectWatches)
	})
	return instance
}

func (wc *WatchCollector) RegisterWatchSync(name string, lock sync.Locker, rval reflect.Value, flags int) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	cleanName, tags := utilfn.ParseNameAndTags(name)
	wc.watchDecls[cleanName] = &WatchDecl{
		Name:   cleanName,
		Tags:   tags,
		Lock:   lock,
		PtrVal: rval,
		Flags:  flags,
	}
}

func (wc *WatchCollector) RegisterWatchFunc(name string, getFn any, setFn any, flags int) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	cleanName, tags := utilfn.ParseNameAndTags(name)
	wc.watchDecls[cleanName] = &WatchDecl{
		Name:  cleanName,
		Tags:  tags,
		GetFn: getFn,
		SetFn: setFn,
		Flags: flags,
	}
}

func (wc *WatchCollector) RegisterWatchAtomic(name string, atomicVal any, flags int) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	cleanName, tags := utilfn.ParseNameAndTags(name)
	wc.watchDecls[cleanName] = &WatchDecl{
		Name:      cleanName,
		Tags:      tags,
		AtomicVal: atomicVal,
		Flags:     flags,
	}
}

func (wc *WatchCollector) RegisterHook(name string, hook any, flags int) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	cleanName, tags := utilfn.ParseNameAndTags(name)
	wc.watchDecls[cleanName] = &WatchDecl{
		Name:   cleanName,
		Tags:   tags,
		HookFn: hook,
		Flags:  flags,
	}
}

func (wc *WatchCollector) UnregisterWatch(name string) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	cleanName, _ := utilfn.ParseNameAndTags(name)
	delete(wc.watchDecls, cleanName)
}

// RegisterWatchDecl registers a watch declaration in the watchDecls2 map
// Returns an error if a watch with the same name already exists
func (wc *WatchCollector) RegisterWatchDecl(decl *ds.WatchDecl) error {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	
	if decl == nil || decl.Name == "" {
		err := fmt.Errorf("watch declaration is nil or has empty name")
		wc.regErrors = append(wc.regErrors, ds.ErrWithLine{
			Error: err.Error(),
			Line:  decl.NewLine,
		})
		return err
	}
	
	// Check if a watch with this name already exists
	if _, exists := wc.watchDecls2[decl.Name]; exists {
		err := fmt.Errorf("watch with name '%s' already registered", decl.Name)
		wc.regErrors = append(wc.regErrors, ds.ErrWithLine{
			Error: err.Error(),
			Line:  decl.NewLine,
		})
		return err
	}
	
	// Register the watch declaration
	wc.watchDecls2[decl.Name] = decl
	return nil
}

// GetRegErrors returns a copy of the registration errors
func (wc *WatchCollector) GetRegErrors() []ds.ErrWithLine {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	
	// Create a copy to avoid race conditions
	result := make([]ds.ErrWithLine, len(wc.regErrors))
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

func (wc *WatchCollector) getWatchDecl(name string) *WatchDecl {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	return wc.watchDecls[name]
}

func (wc *WatchCollector) PushWatchValue(w *ds.WatchSample) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	wc.watchVals = append(wc.watchVals, *w)
}

func (wc *WatchCollector) getAndClearWatchVals() []ds.WatchSample {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	watchVals := wc.watchVals
	wc.watchVals = nil
	return watchVals
}

// computeDeltaWatch compares current and last watch sample and returns a delta sample
// For delta updates, we start with a full copy of current and clear fields that haven't changed
func (wc *WatchCollector) computeDeltaWatch(name string, current ds.WatchSample) (ds.WatchSample, bool) {
	// For push values, we always include all fields and don't compute deltas
	if current.IsPush() {
		return current, false
	}
	wc.lock.Lock()
	lastSample, exists := wc.lastWatchValues[name]
	wc.lock.Unlock()
	if !exists {
		// New watch, include all fields
		return current, false
	}
	deltaSample := current
	sameValue := true
	// delta only operates for flags, type, strval, gofmtval, jsonval, addr, tags
	if lastSample.Flags == current.Flags {
		deltaSample.Flags = 0
	}
	if lastSample.Type == current.Type {
		deltaSample.Type = ""
	}
	if lastSample.StrVal == current.StrVal {
		deltaSample.StrVal = ""
	} else {
		sameValue = false
	}
	if lastSample.GoFmtVal == current.GoFmtVal {
		deltaSample.GoFmtVal = ""
	} else {
		sameValue = false
	}
	if lastSample.JsonVal == current.JsonVal {
		deltaSample.JsonVal = ""
	} else {
		sameValue = false
	}
	if slices.Equal(lastSample.Addr, current.Addr) {
		deltaSample.Addr = nil
	}
	if slices.Equal(lastSample.Tags, current.Tags) {
		deltaSample.Tags = nil
	}
	return deltaSample, sameValue
}

func (wc *WatchCollector) collectWatch(decl *WatchDecl) {
	if decl.IsSync() {
		typeStr := decl.PtrVal.Elem().Type().String()
		wc.RecordWatchValue(decl.Name, decl.Tags, decl.Lock, decl.PtrVal, typeStr, decl.Flags)
		return
	}
	if decl.IsFunc() {
		getFnValue := reflect.ValueOf(decl.GetFn)
		results := getFnValue.Call(nil)
		typeStr := getFnValue.Type().Out(0).String()
		value := results[0]
		wc.RecordWatchValue(decl.Name, decl.Tags, nil, value, typeStr, decl.Flags)
		return
	}
	if decl.IsAtomic() {
		typeStr := reflect.TypeOf(decl.AtomicVal).String()
		atomicValue := reflect.ValueOf(decl.AtomicVal)
		loadMethod := atomicValue.MethodByName("Load")
		results := loadMethod.Call(nil)
		value := results[0]
		wc.RecordWatchValue(decl.Name, decl.Tags, nil, value, typeStr, decl.Flags)
		return
	}
	if decl.IsHook() {
		if decl.HookSent.Load() {
			return
		}
		decl.HookSent.Store(true)
		watch := ds.WatchSample{
			Name:  decl.Name,
			Tags:  decl.Tags,
			Ts:    time.Now().UnixMilli(),
			Flags: decl.Flags,
			Type:  reflect.TypeOf(decl.HookFn).String(),
			Addr:  []string{fmt.Sprintf("%p", decl.HookFn)},
		}
		// Set the kind to Func for hooks
		watch.SetKind(uint(reflect.Func))
		wc.PushWatchValue(&watch)
		return
	}
}

// CollectWatches collects watch information and sends it to the controller
// note we do not hold the lock for the duration of this function
func (wc *WatchCollector) CollectWatches() {
	if !global.OutrigEnabled.Load() || wc.controller == nil {
		return
	}

	sendFull := wc.getSendFullAndReset()
	currentWatchValues := make(map[string]ds.WatchSample)

	watchNames := wc.GetWatchNames()
	for _, name := range watchNames {
		watchDecl := wc.getWatchDecl(name)
		if watchDecl == nil {
			continue
		}
		wc.collectWatch(watchDecl)
	}

	// Get all collected watch values
	allWatchVals := wc.getAndClearWatchVals()
	deltaWatchVals := make([]ds.WatchSample, 0, len(allWatchVals))
	numSameValue := 0

	// Process each watch value for delta calculation
	for _, watch := range allWatchVals {
		// Store current watch value for next delta calculation
		currentWatchValues[watch.Name] = watch

		// For delta updates, only include changed fields
		if !sendFull {
			deltaWatch, sameValue := wc.computeDeltaWatch(watch.Name, watch)
			if sameValue {
				numSameValue++
			}
			deltaWatchVals = append(deltaWatchVals, deltaWatch)
		} else {
			// Full update, include all fields
			deltaWatchVals = append(deltaWatchVals, watch)
		}
	}

	// Update the last watch values for next delta calculation
	wc.setLastWatchValues(currentWatchValues)

	watchInfo := &ds.WatchInfo{
		Ts:      time.Now().UnixMilli(),
		Delta:   !sendFull,
		Watches: deltaWatchVals,
	}

	// Send the watch packet
	pk := &ds.PacketType{
		Type: ds.PacketTypeWatch,
		Data: watchInfo,
	}

	wc.controller.SendPacket(pk)
}

func (wc *WatchCollector) recordWatch(watch ds.WatchSample) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	if len(wc.watchVals) > MaxWatchVals {
		return
	}
	wc.watchVals = append(wc.watchVals, watch)
}

const MaxWatchWaitTime = 10 * time.Millisecond

func (wc *WatchCollector) RecordWatchValue(name string, tags []string, lock sync.Locker, rval reflect.Value, typeStr string, flags int) {
	watch := ds.WatchSample{Name: name, Tags: tags, Flags: flags}
	watch.Type = typeStr
	if lock != nil {
		locked, waitDuration := utilfn.TryLockWithTimeout(lock, MaxWatchWaitTime)
		watch.WaitTime = int64(waitDuration / time.Microsecond)
		if !locked {
			watch.Error = "timeout waiting for lock"
			wc.recordWatch(watch)
			return
		}
		defer lock.Unlock()
	}
	watch.Ts = time.Now().UnixMilli()
	const maxPtrDepth = 10
	for depth := 0; rval.Kind() == reflect.Ptr && depth < maxPtrDepth; depth++ {
		if rval.IsNil() {
			watch.StrVal = "nil"
			wc.recordWatch(watch)
			return
		}
		watch.Addr = append(watch.Addr, fmt.Sprintf("%p", rval.Interface()))
		rval = rval.Elem()
	}
	// Store the kind in the lower 5 bits of the flags
	watch.SetKind(uint(rval.Kind()))
	switch rval.Kind() {
	case reflect.String:
		watch.StrVal = rval.String()
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		watch.StrVal = fmt.Sprint(rval.Interface())
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct, reflect.Interface:
		if (rval.Kind() == reflect.Interface || rval.Kind() == reflect.Slice || rval.Kind() == reflect.Map) && rval.IsNil() {
			watch.StrVal = "nil"
			wc.recordWatch(watch)
			return
		}
		watch.GoFmtVal = fmt.Sprintf("%#v", rval.Interface())
		barr, err := json.Marshal(rval.Interface())
		if err == nil {
			watch.JsonVal = string(barr)
		}
		if strer, ok := rval.Interface().(fmt.Stringer); ok {
			watch.StrVal = strer.String()
		}
		if rval.Kind() == reflect.Slice || rval.Kind() == reflect.Array || rval.Kind() == reflect.Map {
			watch.Len = rval.Len()
		}
		if rval.Kind() == reflect.Slice {
			watch.Cap = rval.Cap()
		}
	case reflect.Chan:
		if rval.IsNil() {
			watch.StrVal = "nil"
		} else {
			watch.StrVal = fmt.Sprintf("(chan:%p)", rval.Interface())
		}
		watch.Len = rval.Len()
		watch.Cap = rval.Cap()
	case reflect.Func:
		if rval.IsNil() {
			watch.StrVal = "nil"
		} else {
			watch.StrVal = fmt.Sprintf("(func:%p)", rval.Interface())
		}
	case reflect.UnsafePointer, reflect.Ptr:
		watch.StrVal = fmt.Sprintf("%p", rval.Interface())
	default:
		watch.Error = fmt.Sprintf("unsupported kind: %s", rval.Kind())
	}
	wc.recordWatch(watch)
}

func (wc *WatchCollector) setLastWatchValues(watches map[string]ds.WatchSample) {
	wc.lock.Lock()
	defer wc.lock.Unlock()

	// Update last watch values with current ones
	for name, watch := range watches {
		wc.lastWatchValues[name] = watch
	}

	// Remove watches that no longer exist
	for name := range wc.lastWatchValues {
		if _, found := watches[name]; !found {
			delete(wc.lastWatchValues, name)
		}
	}
}
