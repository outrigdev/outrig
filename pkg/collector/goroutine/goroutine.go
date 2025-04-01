package goroutine

import (
	"bytes"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

// parseGoroutineStacks parses the output of runtime.Stack()
func (gc *GoroutineCollector) parseGoroutineStacks(stackData []byte) *ds.GoroutineInfo {
	stacks := bytes.Split(stackData, []byte("\n\n"))
	goroutineStacks := make([]ds.GoRoutineStack, 0, len(stacks))
	activeGoroutines := make(map[int64]bool)

	// Regular expression to extract goroutine ID and state
	re := regexp.MustCompile(`goroutine (\d+) \[([^\]]+)\]`)

	for _, stack := range stacks {
		if len(stack) == 0 {
			continue
		}

		stackStr := string(stack)
		matches := re.FindStringSubmatch(stackStr)
		if len(matches) < 3 {
			continue
		}

		id, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}

		// Mark this goroutine as active
		activeGoroutines[id] = true

		state := matches[2]

		grStack := ds.GoRoutineStack{
			GoId:       id,
			State:      state,
			StackTrace: strings.TrimSpace(stackStr),
		}

		// Add name if available
		if name, ok := gc.GetGoRoutineName(id); ok {
			grStack.Name, grStack.Tags = utilfn.ParseNameAndTags(name)
		}

		goroutineStacks = append(goroutineStacks, grStack)
	}

	// Clean up names for goroutines that no longer exist
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
