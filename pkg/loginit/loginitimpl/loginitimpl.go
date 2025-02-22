package loginitimpl

import (
	"os"
	"sync"
	"time"
)

var MaxInitBufferSize = 64 * 1024
var InitWaitTimeMs = 5000
var Initialized bool
var InitLock = &sync.Mutex{}

// Global variables for wrapping logic

var OrigStdout *os.File = os.Stdout
var OrigStderr *os.File = os.Stderr

var StdoutFileWrap FileWrap

type LogCallbackFnType = func(string, string)

type FileWrap interface {
	Run()
	Restore() (*os.File, error)
	SetCallback(callbackFn LogCallbackFnType)
	StopBuffering()
	GetOrigFile() *os.File
}

func InitLogWrap(callbackFn LogCallbackFnType) error {
	InitLock.Lock()
	defer InitLock.Unlock()
	if Initialized {
		if callbackFn != nil {
			StdoutFileWrap.SetCallback(callbackFn)
			StdoutFileWrap.StopBuffering()
		}
		return nil
	}
	Initialized = true
	dw, err := MakeFileWrap(os.Stdout, "/dev/stdout", callbackFn)
	if err != nil {
		return err
	}
	OrigStdout = dw.GetOrigFile()
	StdoutFileWrap = dw
	go dw.Run()
	time.AfterFunc(time.Duration(InitWaitTimeMs)*time.Millisecond, func() {
		StdoutFileWrap.StopBuffering()
	})
	return nil
}

func DisableLogWrap() {
	InitLock.Lock()
	defer InitLock.Unlock()
	if !Initialized {
		return
	}
	Initialized = false
	OrigStdout, _ = StdoutFileWrap.Restore()
	StdoutFileWrap = nil
}
