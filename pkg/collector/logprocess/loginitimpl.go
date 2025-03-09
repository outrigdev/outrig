package logprocess

import (
	"os"
	"sync"
	"time"

	"github.com/outrigdev/outrig/pkg/ds"
)

var MaxInitBufferSize = 64 * 1024
var InitWaitTimeMs = 2000
var Initialized bool
var InitLock = &sync.Mutex{}

// Global variables for wrapping logic

var OrigStdout *os.File = os.Stdout
var OrigStderr *os.File = os.Stderr

var StdoutFileWrap, StderrFileWrap FileWrap

type LogCallbackFnType = func(string, string)

type FileWrap interface {
	Run()
	Restore() (*os.File, error)
	StopBuffering()
	GetOrigFile() *os.File
}

func (lc *LogCollector) initInternal(controller ds.Controller) error {
	var wrapStdout bool = true
	var wrapStderr bool = true

	// Get controller from the LogCollector instance if available
	if controller != nil {
		config := controller.GetConfig()
		if config.LogProcessorConfig != nil {
			wrapStdout = config.LogProcessorConfig.WrapStdout
			wrapStderr = config.LogProcessorConfig.WrapStderr
		}
	}
	InitLock.Lock()
	defer InitLock.Unlock()
	if Initialized {
		return nil
	}
	Initialized = true
	if wrapStdout {
		dw, err := MakeFileWrap(os.Stdout, "/dev/stdout", lc.LogCallback, true)
		if err != nil {
			return err
		}
		OrigStdout = dw.GetOrigFile()
		StdoutFileWrap = dw
		go dw.Run()
		time.AfterFunc(time.Duration(InitWaitTimeMs)*time.Millisecond, func() {
			StdoutFileWrap.StopBuffering()
		})
	}
	if wrapStderr {
		dw, err := MakeFileWrap(os.Stderr, "/dev/stderr", lc.LogCallback, true)
		if err != nil {
			return err
		}
		OrigStderr = dw.GetOrigFile()
		StderrFileWrap = dw
		go dw.Run()
		time.AfterFunc(time.Duration(InitWaitTimeMs)*time.Millisecond, func() {
			StderrFileWrap.StopBuffering()
		})
	}
	return nil
}

func DisableLogWrap() {
	InitLock.Lock()
	defer InitLock.Unlock()
	if !Initialized {
		return
	}
	Initialized = false
	if StdoutFileWrap != nil {
		orig, _ := StdoutFileWrap.Restore()
		if orig != nil {
			OrigStdout = orig
		}
		StdoutFileWrap = nil
	}
	if StderrFileWrap != nil {
		orig, _ := StderrFileWrap.Restore()
		if orig != nil {
			OrigStderr = orig
		}
		StderrFileWrap = nil
	}
}
