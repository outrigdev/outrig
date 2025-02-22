package logprocess

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/loginit/loginitimpl"
	"github.com/outrigdev/outrig/pkg/transport"
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
		err := loginitimpl.InitLogWrap(nil)
		fmt.Fprintf(os.Stderr, "[stderr] LogWrap init: %v\n", err)
	})
}

func OnFirstConnect() {
	firstConnectOnce.Do(func() {
		loginitimpl.InitLogWrap(LogCallback)
		fmt.Fprintf(os.Stderr, "[stderr] LogWrap OnFirstConnect\n")
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
		pk := &transport.PacketType{
			Type: "log",
			Data: logLine,
		}
		transport.SendPacket(pk)
	}
}
