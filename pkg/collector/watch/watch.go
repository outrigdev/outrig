package watch

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
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
	lock       sync.Mutex
	controller ds.Controller
	ticker     *time.Ticker
	done       chan struct{}

	watchDecls map[string]*WatchDecl
	watchVals  []ds.WatchSample
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
			watchDecls: make(map[string]*WatchDecl),
		}
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

// InitCollector initializes the watch collector with a controller
func (wc *WatchCollector) InitCollector(controller ds.Controller) error {
	wc.controller = controller
	return nil
}

// Enable is called when the collector should start collecting data
func (wc *WatchCollector) Enable() {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	if wc.ticker != nil {
		return
	}

	wc.done = make(chan struct{})
	doneCh := wc.done // Local copy to ensure goroutines use the right channel

	// First immediate collection
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig WatchCollector:first")
		wc.CollectWatches()
	}()

	wc.ticker = time.NewTicker(1 * time.Second)
	localTicker := wc.ticker // Local copy of ticker

	// Periodic collection
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig WatchCollector")
		for {
			select {
			case <-doneCh:
				return
			case <-localTicker.C:
				wc.CollectWatches()
			}
		}
	}()
}

// Disable stops the collector
func (wc *WatchCollector) Disable() {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	if wc.ticker == nil {
		return
	}

	// Signal goroutines to exit
	close(wc.done)
	wc.done = nil

	// Stop the ticker
	wc.ticker.Stop()
	wc.ticker = nil
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
func (wc *WatchCollector) CollectWatches() {
	if !global.OutrigEnabled.Load() || wc.controller == nil {
		return
	}

	watchNames := wc.GetWatchNames()
	for _, name := range watchNames {
		watchDecl := wc.getWatchDecl(name)
		if watchDecl == nil {
			continue
		}
		wc.collectWatch(watchDecl)
	}

	// For now, we're just stubbing this out
	watchInfo := &ds.WatchInfo{
		Ts:      time.Now().UnixMilli(),
		Watches: wc.getAndClearWatchVals(),
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
	for rval.Kind() == reflect.Ptr {
		if rval.IsNil() {
			watch.Value = "nil"
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
		watch.Value = rval.String()
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		watch.Value = fmt.Sprint(rval.Interface())
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct, reflect.Interface:
		barr, err := json.Marshal(rval.Interface())
		if err != nil {
			watch.Error = fmt.Sprintf("error marshalling value: %v", err)
		} else {
			watch.Value = string(barr)
		}
		if rval.Kind() == reflect.Slice || rval.Kind() == reflect.Array || rval.Kind() == reflect.Map {
			watch.Len = rval.Len()
		}
		if rval.Kind() == reflect.Slice {
			watch.Cap = rval.Cap()
		}
	case reflect.Chan:
		watch.Len = rval.Len()
		watch.Cap = rval.Cap()
	case reflect.Func:
		// no value
	case reflect.UnsafePointer:
		watch.Value = fmt.Sprintf("%p", rval.Interface())
	default:
		watch.Error = fmt.Sprintf("unsupported kind: %s", rval.Kind())
	}
	wc.recordWatch(watch)
}
