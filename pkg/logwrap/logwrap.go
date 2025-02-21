package core

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/global"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

// Global variables for wrapping logic
var (
	oldStdout, oldStderr     *os.File
	pipeStdoutR, pipeStdoutW *os.File
	pipeStderrR, pipeStderrW *os.File
	initOnce                 sync.Once
	logChan                  chan ds.LogLine
	logWrapLock              sync.Mutex
)

// InitWrap swaps out os.Stdout/os.Stderr and starts goroutines
// to capture and emit log lines into logChan.
func InitWrap() error {
	logWrapLock.Lock()
	defer logWrapLock.Unlock()

	initOnce.Do(func() {
		logChan = make(chan ds.LogLine, 1000)
		go ProcessLogs()
	})

	var err error

	// Create pipes for stdout
	pipeStdoutR, pipeStdoutW, err = os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Create pipes for stderr
	pipeStderrR, pipeStderrW, err = os.Pipe()
	if err != nil {
		_ = pipeStdoutR.Close()
		_ = pipeStdoutW.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Save old file descriptors
	oldStdout = os.Stdout
	oldStderr = os.Stderr

	// Replace them
	os.Stdout = pipeStdoutW
	os.Stderr = pipeStderrW

	go readAndTee(pipeStdoutR, oldStdout, "")
	go readAndTee(pipeStderrR, oldStderr, "stderr")

	return nil
}

// DisableWrap restores os.Stdout/os.Stderr, closes pipes
func DisableWrap() {
	logWrapLock.Lock()
	defer logWrapLock.Unlock()
	// Restore old stdout and stderr
	if oldStdout != nil {
		os.Stdout = oldStdout
	}
	if oldStderr != nil {
		os.Stderr = oldStderr
	}
	_ = pipeStdoutR.Close()
	_ = pipeStdoutW.Close()
	_ = pipeStderrR.Close()
	_ = pipeStderrW.Close()
}

func ProcessLogs() {
	for range logChan {
		if atomic.LoadInt32(&global.OutrigEnabled) == 0 {
			continue
		}
		// Here you can forward to Outrig server, store in DB, etc.
		// TODO
	}
}

func readAndTee(r *os.File, origFD *os.File, source string) {
	lb := utilfn.MakeLineBuf() // our fixed buffer for line accumulation
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if err == io.EOF || err == io.ErrClosedPipe {
			break
		}
		if err != nil {
			return
		}
		chunk := buf[:n]
		_, _ = origFD.Write(chunk)
		lines := lb.ProcessBuf(chunk)
		for _, line := range lines {
			addLogLine(line, source)
		}
	}
	lastLine := lb.GetPartialAndReset()
	if lastLine != "" {
		addLogLine(lastLine, source)
	}
}

func addLogLine(line string, source string) {
	nextNum := atomic.AddInt64(&global.LineNum, 1)
	logChan <- ds.LogLine{
		LineNum: nextNum,
		Ts:      time.Now().UnixMilli(),
		Msg:     line,
		Source:  source,
	}
}
