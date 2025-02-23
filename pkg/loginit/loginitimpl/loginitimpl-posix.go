//go:build !windows

package loginitimpl

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"

	"github.com/outrigdev/outrig/pkg/utilfn"
)

type DupWrap struct {
	WrapedFDNum   int
	DupedOrigFD   int
	DupedOrigFile *os.File
	Source        string

	PipeR, PipeW   *os.File
	BufferedOutput []byte
	CallbackFn     atomic.Pointer[LogCallbackFnType]
	ShouldBuffer   atomic.Bool

	Buffer *utilfn.LineBuf
}

func MakeFileWrap(origFile *os.File, source string, callbackFn LogCallbackFnType) (FileWrap, error) {
	fd := int(origFile.Fd())
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}
	duppedFdNum, err := syscall.Dup(fd)
	if err != nil {
		_ = pipeR.Close()
		_ = pipeW.Close()
		return nil, fmt.Errorf("failed to dup fd: %w", err)
	}
	err = syscall.Dup2(int(pipeW.Fd()), fd)
	if err != nil {
		_ = pipeR.Close()
		_ = pipeW.Close()
		_ = syscall.Close(duppedFdNum)
		return nil, fmt.Errorf("failed to dup2 fd: %w", err)
	}
	rtn := &DupWrap{
		WrapedFDNum:   fd,
		DupedOrigFD:   duppedFdNum,
		DupedOrigFile: os.NewFile(uintptr(duppedFdNum), source),
		PipeR:         pipeR,
		PipeW:         pipeW,
		Source:        source,
	}
	if callbackFn != nil {
		rtn.CallbackFn.Store(&callbackFn)
	}
	rtn.ShouldBuffer.Store(callbackFn == nil)
	return rtn, nil
}

func (d *DupWrap) GetOrigFile() *os.File {
	return d.DupedOrigFile
}

func (d *DupWrap) Restore() (*os.File, error) {
	err := syscall.Dup2(int(d.DupedOrigFD), d.WrapedFDNum)
	if err != nil {
		return nil, fmt.Errorf("failed to restore fd: %w", err)
	}
	_ = d.PipeR.Close()
	_ = d.PipeW.Close()
	_ = os.NewFile(uintptr(d.DupedOrigFD), d.Source).Close()
	return os.NewFile(uintptr(d.WrapedFDNum), d.Source), nil
}

func (d *DupWrap) handleData(data []byte) {
	callbackFnPtr := d.CallbackFn.Load()
	if callbackFnPtr != nil && *callbackFnPtr != nil {
		if d.Buffer == nil {
			d.Buffer = utilfn.MakeLineBuf()
		}
		var lines []string
		if len(d.BufferedOutput) > 0 {
			lines = d.Buffer.ProcessBuf(d.BufferedOutput)
			d.BufferedOutput = nil
		}
		lines = append(lines, d.Buffer.ProcessBuf(data)...)
		for _, line := range lines {
			(*callbackFnPtr)(line, d.Source)
		}
	} else {
		if !d.ShouldBuffer.Load() {
			d.BufferedOutput = nil
			return
		}
		if len(d.BufferedOutput) >= MaxInitBufferSize {
			return
		}
		if len(d.BufferedOutput)+len(data) > MaxInitBufferSize {
			data = data[:MaxInitBufferSize-len(d.BufferedOutput)]
		}
		d.BufferedOutput = append(d.BufferedOutput, data...)
	}
}

func (d *DupWrap) Run() {
	buf := make([]byte, 4096)
	for {
		n, err := d.PipeR.Read(buf)
		if n > 0 {
			_, _ = d.DupedOrigFile.Write(buf[:n])
			d.handleData(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func (d *DupWrap) SetCallback(callbackFn LogCallbackFnType) {
	d.CallbackFn.Store(&callbackFn)
}

func (d *DupWrap) StopBuffering() {
	d.ShouldBuffer.Store(false)
}
