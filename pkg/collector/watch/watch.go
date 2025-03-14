package watch

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const (
	WatchTypeSync = "sync"
)

const MaxWatchVals = 10000

// WatchCollector implements the collector.Collector interface for watch collection
type WatchCollector struct {
	lock       sync.Mutex
	controller ds.Controller
	ticker     *time.Ticker

	watchDecls map[string]*WatchDecl
	watchVals  []ds.Watch
}

type WatchDecl struct {
	WatchType string
	Name      string
	Lock      sync.Locker
	Ptr       *any
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

func (wc *WatchCollector) RegisterWatchSync(name string, ptr *any) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	wc.watchDecls[name] = &WatchDecl{
		WatchType: WatchTypeSync,
		Name:      name,
		Lock:      &sync.Mutex{},
		Ptr:       ptr,
	}
}

func (wc *WatchCollector) UnregisterWatch(name string) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	delete(wc.watchDecls, name)
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
	wc.CollectWatches()
	wc.ticker = time.NewTicker(1 * time.Second)
	go func() {
		for range wc.ticker.C {
			wc.CollectWatches()
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

func (wc *WatchCollector) getAndClearWatchVals() []ds.Watch {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	watchVals := wc.watchVals
	wc.watchVals = nil
	return watchVals
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
		switch watchDecl.WatchType {
		case WatchTypeSync:
			wc.doWatchSync(name, watchDecl.Lock, watchDecl.Ptr)
		default:
			continue
		}
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

func (wc *WatchCollector) recordWatch(watch ds.Watch) {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	if len(wc.watchVals) > MaxWatchVals {
		return
	}
	wc.watchVals = append(wc.watchVals, watch)
}

const MaxWatchWaitTime = 10 * time.Millisecond

func (wc *WatchCollector) doWatchSync(name string, lock sync.Locker, val *any) {
	watch := ds.Watch{Name: name}
	if val == nil {
		watch.Type = "nil"
		watch.Error = "nil pointer"
		wc.recordWatch(watch)
		return
	}
	rval := reflect.ValueOf(val)
	watch.Type = rval.Elem().Type().String()
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
	watch.Ts = time.Now().UnixMicro()
	for rval.Kind() == reflect.Ptr {
		if rval.IsNil() {
			watch.Value = "nil"
			wc.recordWatch(watch)
			return
		}
		watch.Addr = append(watch.Addr, fmt.Sprintf("%p", rval.Interface()))
		rval = rval.Elem()
	}
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
