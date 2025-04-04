package goroutine

import (
	"bytes"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/ioutrig"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// GoroutineCollector implements the collector.Collector interface for goroutine collection
type GoroutineCollector struct {
	lock           sync.Mutex
	controller     ds.Controller
	ticker         *time.Ticker
	done           chan struct{}
	goroutineNames map[int64]string // map from goroutine ID to name
}

// CollectorName returns the unique name of the collector
func (gc *GoroutineCollector) CollectorName() string {
	return "goroutine"
}

// singleton instance
var instance *GoroutineCollector
var instanceOnce sync.Once

// GetInstance returns the singleton instance of GoroutineCollector
func GetInstance() *GoroutineCollector {
	instanceOnce.Do(func() {
		instance = &GoroutineCollector{
			goroutineNames: make(map[int64]string),
		}
	})
	return instance
}

// InitCollector initializes the goroutine collector with a controller
func (gc *GoroutineCollector) InitCollector(controller ds.Controller) error {
	gc.controller = controller
	return nil
}

// Enable is called when the collector should start collecting data
func (gc *GoroutineCollector) Enable() {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	if gc.ticker != nil {
		return
	}

	gc.done = make(chan struct{})
	doneCh := gc.done // Local copy to ensure goroutines use the right channel

	// First immediate collection
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig GoRoutineCollector:first")
		gc.DumpGoroutines()
	}()

	gc.ticker = time.NewTicker(1 * time.Second)
	localTicker := gc.ticker // Local copy of ticker

	// Periodic collection
	go func() {
		ioutrig.I.SetGoRoutineName("#outrig GoRoutineCollector")
		for {
			select {
			case <-doneCh:
				return
			case <-localTicker.C:
				gc.DumpGoroutines()
			}
		}
	}()
}

func (gc *GoroutineCollector) Disable() {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	if gc.ticker == nil {
		return
	}

	// Signal goroutines to exit
	close(gc.done)
	gc.done = nil

	// Stop the ticker
	gc.ticker.Stop()
	gc.ticker = nil
}

// DumpGoroutines dumps all goroutines and sends the information
func (gc *GoroutineCollector) DumpGoroutines() {
	if !global.OutrigEnabled.Load() || gc.controller == nil {
		return
	}

	// Get all goroutine stacks
	buf := make([]byte, 1<<20)
	stackLen := runtime.Stack(buf, true)
	stackData := buf[:stackLen]

	// Parse the stack data
	goroutineInfo := gc.parseGoroutineStacks(stackData)

	// Send the goroutine packet
	pk := &ds.PacketType{
		Type: ds.PacketTypeGoroutine,
		Data: goroutineInfo,
	}

	gc.controller.SendPacket(pk)
}

// SetGoRoutineName sets a name for a goroutine
func (gc *GoroutineCollector) SetGoRoutineName(goId int64, name string) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	gc.goroutineNames[goId] = name
}

// GetGoRoutineName gets the name for a goroutine
func (gc *GoroutineCollector) GetGoRoutineName(goId int64) (string, bool) {
	gc.lock.Lock()
	defer gc.lock.Unlock()
	name, ok := gc.goroutineNames[goId]
	return name, ok
}

var startRe = regexp.MustCompile(`(?m)^goroutine\s+\d+`)
var stackRe = regexp.MustCompile(`goroutine (\d+) \[([^\]]+)\].*\n((?s).*)`)

func (gc *GoroutineCollector) parseGoroutineStacks(stackData []byte) *ds.GoroutineInfo {
	goroutineStacks := make([]ds.GoRoutineStack, 0)
	activeGoroutines := make(map[int64]bool)

	startIndices := startRe.FindAllIndex(stackData, -1)
	for i, startIdx := range startIndices {
		start := startIdx[0]
		end := len(stackData)
		if i+1 < len(startIndices) {
			end = startIndices[i+1][0]
		}
		goroutineData := stackData[start:end]
		matches := stackRe.FindSubmatch(goroutineData)
		if len(matches) < 4 {
			continue
		}
		id, _ := strconv.ParseInt(string(matches[1]), 10, 64) // this is safe because the regex guarantees a number
		activeGoroutines[id] = true
		grStack := ds.GoRoutineStack{
			GoId:       id,
			State:      string(matches[2]),
			StackTrace: string(bytes.TrimSpace(matches[3])),
		}
		if name, ok := gc.GetGoRoutineName(id); ok {
			grStack.Name, grStack.Tags = utilfn.ParseNameAndTags(name)
		}
		goroutineStacks = append(goroutineStacks, grStack)
	}

	gc.cleanupGoroutineNames(activeGoroutines)
	return &ds.GoroutineInfo{
		Ts:     time.Now().UnixMilli(),
		Count:  len(goroutineStacks),
		Stacks: goroutineStacks,
	}
}

// cleanupGoroutineNames removes names for goroutines that are no longer active
func (gc *GoroutineCollector) cleanupGoroutineNames(activeGoroutines map[int64]bool) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	// Remove names for goroutines that no longer exist
	for id := range gc.goroutineNames {
		if !activeGoroutines[id] {
			delete(gc.goroutineNames, id)
		}
	}
}
