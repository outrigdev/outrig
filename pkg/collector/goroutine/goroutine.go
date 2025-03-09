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
)

// GoroutineCollector implements the collector.Collector interface for goroutine collection
type GoroutineCollector struct {
	firstConnectOnce sync.Once
	controller       ds.Controller
}

// singleton instance
var instance *GoroutineCollector
var instanceOnce sync.Once

// GetInstance returns the singleton instance of GoroutineCollector
func GetInstance() *GoroutineCollector {
	instanceOnce.Do(func() {
		instance = &GoroutineCollector{}
	})
	return instance
}

// InitCollector initializes the goroutine collector with a controller
func (gc *GoroutineCollector) InitCollector(controller ds.Controller) error {
	gc.controller = controller
	return nil
}

// OnFirstConnect is called when the first connection is established
func (gc *GoroutineCollector) OnFirstConnect() {
	gc.firstConnectOnce.Do(func() {
		// Immediately dump goroutines
		gc.DumpGoroutines()
	})
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

// parseGoroutineStacks parses the output of runtime.Stack()
func (gc *GoroutineCollector) parseGoroutineStacks(stackData []byte) *ds.GoroutineInfo {
	stacks := bytes.Split(stackData, []byte("\n\n"))
	goroutineStacks := make([]ds.GoroutineStack, 0, len(stacks))

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

		state := matches[2]
		
		goroutineStacks = append(goroutineStacks, ds.GoroutineStack{
			ID:         id,
			State:      state,
			StackTrace: strings.TrimSpace(stackStr),
		})
	}

	return &ds.GoroutineInfo{
		Timestamp: time.Now().UnixMilli(),
		Count:     len(goroutineStacks),
		Stacks:    goroutineStacks,
	}
}
