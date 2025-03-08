package logprocess

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/loginit/loginitimpl"
)

const LogBufferSize = 2000

// Global variables for wrapping logic
var (
	initOnce         sync.Once
	firstConnectOnce sync.Once
	logChan          chan *ds.LogLine
)

func InitLogProcess() {
	initOnce.Do(func() {
		logChan = make(chan *ds.LogLine, LogBufferSize)
		loginitimpl.InitLogWrap(nil)
	})
}

func OnFirstConnect() {
	firstConnectOnce.Do(func() {
		loginitimpl.InitLogWrap(LogCallback)
		go ConsumeLogLines()
	})
}

func LogCallback(line string, source string) {
	addLogLine(line, source)
}

func addLogLine(line string, source string) {
	if atomic.LoadInt32(&global.OutrigEnabled) == 0 {
		return
	}
	nextNum := atomic.AddInt64(&global.LineNum, 1)
	logLine := &ds.LogLine{
		LineNum: nextNum,
		Ts:      time.Now().UnixMilli(),
		Msg:     line,
		Source:  source,
	}
	logChan <- logLine
}

func ConsumeLogLines() {
	for {
		logLine := <-logChan
		pk := &ds.PacketType{
			Type: ds.PacketTypeLog,
			Data: logLine,
		}
		
		if global.GlobalController != nil {
			global.GlobalController.SendPacket(pk)
		}
	}
}
