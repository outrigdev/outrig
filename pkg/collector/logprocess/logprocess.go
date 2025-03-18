package logprocess

import (
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
)

const LogBufferSize = 2000

// LogCollector implements the collector.Collector interface for log collection
type LogCollector struct {
	firstConnectOnce sync.Once
	logChan          chan *ds.LogLine
	controller       ds.Controller
}

// CollectorName returns the unique name of the collector
func (lc *LogCollector) CollectorName() string {
	return "logprocess"
}

// singleton instance
var instance *LogCollector
var instanceOnce sync.Once

// GetInstance returns the singleton instance of LogCollector
func GetInstance() *LogCollector {
	instanceOnce.Do(func() {
		instance = &LogCollector{
			logChan: make(chan *ds.LogLine, LogBufferSize),
		}
	})
	return instance
}

// InitCollector initializes the log collector with a controller
func (lc *LogCollector) InitCollector(controller ds.Controller) error {
	lc.controller = controller
	lc.initInternal(controller)
	return nil
}

func (lc *LogCollector) Enable() {
	lc.firstConnectOnce.Do(func() {
		go lc.ConsumeLogLines()
	})
}

func (lc *LogCollector) Disable() {

}

// LogCallback is called when a log line is received
func (lc *LogCollector) LogCallback(line string, source string) {
	lc.addLogLine(line, source)
}

// addLogLine adds a log line to be processed
func (lc *LogCollector) addLogLine(line string, source string) {
	if !global.OutrigEnabled.Load() {
		return
	}
	logLine := &ds.LogLine{
		LineNum: 0, // Server will set the actual line number
		Ts:      time.Now().UnixMilli(),
		Msg:     line,
		Source:  source,
	}
	lc.logChan <- logLine
}

// ConsumeLogLines starts consuming log lines from the channel
func (lc *LogCollector) ConsumeLogLines() {
	for {
		logLine := <-lc.logChan
		pk := &ds.PacketType{
			Type: ds.PacketTypeLog,
			Data: logLine,
		}

		if lc.controller != nil {
			lc.controller.SendPacket(pk)
		}
	}
}
