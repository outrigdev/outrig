// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package stacktrace

import (
	"testing"
)

func TestParseFrame(t *testing.T) {
	tests := []struct {
		name          string
		funcLine      string
		fileLine      string
		expectSuccess bool
		expectedFrame StackFrame
	}{
		{
			name:          "Method with receiver",
			funcLine:      "internal/poll.(*FD).Read(0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd})",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go:165 +0x1fc",
			expectSuccess: true,
			expectedFrame: StackFrame{
				Package:    "internal/poll",
				FuncName:   "(*FD).Read",
				FuncArgs:   "0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd}",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go",
				LineNumber: 165,
				PCOffset:   "+0x1fc",
			},
		},
		{
			name:          "Function without receiver",
			funcLine:      "runtime.doInit(0x12f7be0)",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go:6329",
			expectSuccess: true,
			expectedFrame: StackFrame{
				Package:    "runtime",
				FuncName:   "doInit",
				FuncArgs:   "0x12f7be0",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go",
				LineNumber: 6329,
				PCOffset:   "",
			},
		},
		{
			name:          "Function with dots in package name",
			funcLine:      "github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute.func2()",
			fileLine:      "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:326 +0x14c",
			expectSuccess: true,
			expectedFrame: StackFrame{
				Package:    "github.com/outrigdev/outrig/pkg/rpc",
				FuncName:   "(*WshRouter).RegisterRoute.func2",
				FuncArgs:   "",
				FilePath:   "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go",
				LineNumber: 326,
				PCOffset:   "+0x14c",
			},
		},
		{
			name:          "Function with ellipsis",
			funcLine:      "internal/poll.(*pollDesc).waitRead(...)",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_poll_runtime.go:89",
			expectSuccess: true,
			expectedFrame: StackFrame{
				Package:    "internal/poll",
				FuncName:   "(*pollDesc).waitRead",
				FuncArgs:   "...",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_poll_runtime.go",
				LineNumber: 89,
				PCOffset:   "",
			},
		},
		{
			name:          "Main function",
			funcLine:      "main.main()",
			fileLine:      "/Users/mike/work/outrig/server/main-server.go:291 +0x714",
			expectSuccess: true,
			expectedFrame: StackFrame{
				Package:    "main",
				FuncName:   "main",
				FuncArgs:   "",
				FilePath:   "/Users/mike/work/outrig/server/main-server.go",
				LineNumber: 291,
				PCOffset:   "+0x714",
			},
		},
		{
			name:          "Invalid function line",
			funcLine:      "this is not a valid function line",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go:6329",
			expectSuccess: false,
		},
		{
			name:          "Invalid file line",
			funcLine:      "runtime.doInit(0x12f7be0)",
			fileLine:      "this is not a valid file line",
			expectSuccess: false,
		},
		{
			name:          "Method with value receiver",
			funcLine:      "time.Time.Add(0x140003801e0, 0x140003ae723)",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/time/time.go:1076 +0x1a4",
			expectSuccess: true,
			expectedFrame: StackFrame{
				Package:    "time",
				FuncName:   "Time.Add",
				FuncArgs:   "0x140003801e0, 0x140003ae723",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/time/time.go",
				LineNumber: 1076,
				PCOffset:   "+0x1a4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, ok := parseFrame(tt.funcLine, tt.fileLine, true)
			if ok != tt.expectSuccess {
				t.Fatalf("parseFrame() success = %v, expected %v", ok, tt.expectSuccess)
			}
			if !tt.expectSuccess {
				return
			}
			if frame.Package != tt.expectedFrame.Package {
				t.Errorf("Package = %q, expected %q", frame.Package, tt.expectedFrame.Package)
			}
			if frame.FuncName != tt.expectedFrame.FuncName {
				t.Errorf("FuncName = %q, expected %q", frame.FuncName, tt.expectedFrame.FuncName)
			}
			if frame.FuncArgs != tt.expectedFrame.FuncArgs {
				t.Errorf("Args = %q, expected %q", frame.FuncArgs, tt.expectedFrame.FuncArgs)
			}
			if frame.FilePath != tt.expectedFrame.FilePath {
				t.Errorf("FilePath = %q, expected %q", frame.FilePath, tt.expectedFrame.FilePath)
			}
			if frame.LineNumber != tt.expectedFrame.LineNumber {
				t.Errorf("LineNumber = %d, expected %d", frame.LineNumber, tt.expectedFrame.LineNumber)
			}
			if frame.PCOffset != tt.expectedFrame.PCOffset {
				t.Errorf("PCOffset = %q, expected %q", frame.PCOffset, tt.expectedFrame.PCOffset)
			}
		})
	}
}

func TestParseStateComponents(t *testing.T) {
	tests := []struct {
		name                string
		rawState            string
		expectedPrimary     string
		expectedDurationMs  int64
		expectedDuration    string
		expectedExtraStates []string
	}{
		{
			name:                "Simple state",
			rawState:            "running",
			expectedPrimary:     "running",
			expectedDurationMs:  0,
			expectedDuration:    "",
			expectedExtraStates: nil,
		},
		{
			name:                "State with duration",
			rawState:            "chan receive, 101 minutes",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  101 * 60 * 1000,
			expectedDuration:    "101 minutes",
			expectedExtraStates: nil,
		},
		{
			name:                "State with extra states",
			rawState:            "chan receive, locked to thread",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  0,
			expectedDuration:    "",
			expectedExtraStates: []string{"locked to thread"},
		},
		{
			name:                "State with duration and extra states",
			rawState:            "chan receive, 3 minutes, locked to thread",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  3 * 60 * 1000,
			expectedDuration:    "3 minutes",
			expectedExtraStates: []string{"locked to thread"},
		},
		{
			name:                "State with multiple extra states",
			rawState:            "chan receive, locked to thread, syscall",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  0,
			expectedDuration:    "",
			expectedExtraStates: []string{"locked to thread", "syscall"},
		},
		{
			name:                "State with seconds duration",
			rawState:            "chan receive, 45 seconds",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  45 * 1000,
			expectedDuration:    "45 seconds",
			expectedExtraStates: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primaryState, durationMs, duration, extraStates := parseStateComponents(tt.rawState)

			if primaryState != tt.expectedPrimary {
				t.Errorf("Expected primary state %q, got %q", tt.expectedPrimary, primaryState)
			}

			if durationMs != tt.expectedDurationMs {
				t.Errorf("Expected duration %d ms, got %d ms", tt.expectedDurationMs, durationMs)
			}

			if duration != tt.expectedDuration {
				t.Errorf("Expected duration string %q, got %q", tt.expectedDuration, duration)
			}

			if tt.expectedExtraStates == nil {
				if len(extraStates) > 0 {
					t.Errorf("Expected no extra states, got %v", extraStates)
				}
			} else {
				if len(extraStates) != len(tt.expectedExtraStates) {
					t.Errorf("Expected %d extra states, got %d", len(tt.expectedExtraStates), len(extraStates))
				} else {
					for i, expected := range tt.expectedExtraStates {
						if extraStates[i] != expected {
							t.Errorf("Expected extra state %q at index %d, got %q", expected, i, extraStates[i])
						}
					}
				}
			}
		})
	}
}

func TestParseGoRoutineStackTrace(t *testing.T) {
	tests := []struct {
		name                  string
		input                 string
		goId                  int64
		state                 string
		moduleName            string
		expectedGoId          int64
		expectedPrimaryState  string
		expectedFrames        int
		hasCreatedBy          bool
		expectedDurationMs    int64
		expectedExtraStates   []string
		expectedCreatedByGoId int64
	}{
		{
			name: "IO wait goroutine",
			input: `internal/poll.runtime_pollWait(0x1010b0a98, 0x72)
	/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/netpoll.go:351 +0xa0
internal/poll.(*pollDesc).wait(0x140001223c0?, 0x140000a4f98?, 0x1)
	/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_poll_runtime.go:84 +0x28
internal/poll.(*pollDesc).waitRead(...)
	/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0x140001223c0, {0x140000a4f98, 0x1000, 0x1000})
	/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go:165 +0x1fc
os.(*File).read(...)
	/opt/homebrew/Cellar/go/1.23.4/libexec/src/os/file_posix.go:29
os.(*File).Read(0x14000116190, {0x140000a4f98?, 0x1eb?, 0x1000?})
	/opt/homebrew/Cellar/go/1.23.4/libexec/src/os/file.go:124 +0x70
github.com/outrigdev/outrig/pkg/collector/logprocess.(*DupWrap).Run(0x14000146700)
	/Users/mike/work/outrig/pkg/collector/logprocess/loginitimpl-posix.go:110 +0x64
created by github.com/outrigdev/outrig/pkg/collector/logprocess.(*LogCollector).initInternal in goroutine 1
	/Users/mike/work/outrig/pkg/collector/logprocess/loginitimpl.go:69 +0x3dc`,
			goId:                  38,
			state:                 "IO wait",
			expectedGoId:          38,
			expectedPrimaryState:  "IO wait",
			expectedFrames:        7,
			hasCreatedBy:          true,
			expectedDurationMs:    0,
			expectedExtraStates:   nil,
			expectedCreatedByGoId: 1,
		},
		{
			name: "chan receive goroutine with duration",
			input: `github.com/outrigdev/outrig/pkg/rpc.(*WshRpcProxy).RecvRpcMessage(0x103bab9e0?)
	/Users/mike/work/outrig/pkg/rpc/rpcproxy.go:34 +0x2c
github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute.func2()
	/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:326 +0x14c
created by github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute in goroutine 327
	/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:315 +0x3cc`,
			goId:                  338,
			state:                 "chan receive, 101 minutes",
			expectedGoId:          338,
			expectedPrimaryState:  "chan receive",
			expectedFrames:        2,
			hasCreatedBy:          true,
			expectedDurationMs:    101 * 60 * 1000,
			expectedExtraStates:   nil,
			expectedCreatedByGoId: 327,
		},
		{
			name: "goroutine 1 with no created by",
			input: `main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
			goId:                  1,
			state:                 "chan receive, 105 minutes",
			expectedGoId:          1,
			expectedPrimaryState:  "chan receive",
			expectedFrames:        1,
			hasCreatedBy:          false,
			expectedDurationMs:    105 * 60 * 1000,
			expectedExtraStates:   nil,
			expectedCreatedByGoId: 0,
		},
		{
			name: "goroutine with multiple extra states",
			input: `main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
			goId:                  42,
			state:                 "chan receive, 3 minutes, locked to thread",
			expectedGoId:          42,
			expectedPrimaryState:  "chan receive",
			expectedFrames:        1,
			hasCreatedBy:          false,
			expectedDurationMs:    3 * 60 * 1000,
			expectedExtraStates:   []string{"locked to thread"},
			expectedCreatedByGoId: 0,
		},
		{
			name: "goroutine with lock info",
			input: `main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
			goId:                  55,
			state:                 "semacquire, 2 minutes",
			expectedGoId:          55,
			expectedPrimaryState:  "semacquire",
			expectedFrames:        1,
			hasCreatedBy:          false,
			expectedDurationMs:    2 * 60 * 1000,
			expectedExtraStates:   nil,
			expectedCreatedByGoId: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use empty string for module name in tests
			routine, err := ParseGoRoutineStackTrace(tt.input, "", tt.goId, tt.state)
			if err != nil {
				t.Fatalf("ParseGoRoutineStackTrace returned error: %v", err)
			}

			if routine.GoId != tt.expectedGoId {
				t.Errorf("Expected GoId %d, got %d", tt.expectedGoId, routine.GoId)
			}

			if routine.PrimaryState != tt.expectedPrimaryState {
				t.Errorf("Expected PrimaryState %q, got %q", tt.expectedPrimaryState, routine.PrimaryState)
			}

			if routine.StateDurationMs != tt.expectedDurationMs {
				t.Errorf("Expected DurationMs %d, got %d", tt.expectedDurationMs, routine.StateDurationMs)
			}

			if tt.expectedExtraStates != nil {
				if len(routine.ExtraStates) != len(tt.expectedExtraStates) {
					t.Errorf("Expected %d extra states, got %d", len(tt.expectedExtraStates), len(routine.ExtraStates))
				} else {
					for i, expectedExtra := range tt.expectedExtraStates {
						if i < len(routine.ExtraStates) && routine.ExtraStates[i] != expectedExtra {
							t.Errorf("Expected extra state %q at index %d, got %q", expectedExtra, i, routine.ExtraStates[i])
						}
					}
				}
			} else if len(routine.ExtraStates) > 0 {
				t.Errorf("Expected no extra states, got %v", routine.ExtraStates)
			}

			if len(routine.ParsedFrames) != tt.expectedFrames {
				t.Errorf("Expected %d parsed frames, got %d", tt.expectedFrames, len(routine.ParsedFrames))
			}

			if routine.CreatedByGoId != tt.expectedCreatedByGoId {
				t.Errorf("Expected CreatedByGoId %d, got %d", tt.expectedCreatedByGoId, routine.CreatedByGoId)
			}

			if tt.hasCreatedBy {
				if routine.CreatedByFrame == nil {
					t.Errorf("Expected CreatedByFrame to be set, but it was nil")
				} else {
					if routine.CreatedByFrame.Package == "" {
						t.Errorf("Expected CreatedByFrame.Package to be set, but it was empty")
					}
					if routine.CreatedByFrame.FuncName == "" {
						t.Errorf("Expected CreatedByFrame.FuncName to be set, but it was empty")
					}
					if tt.name == "IO wait goroutine" {
						expectedPath := "/Users/mike/work/outrig/pkg/collector/logprocess/loginitimpl.go"
						if routine.CreatedByFrame.FilePath != expectedPath {
							t.Errorf("Expected CreatedByFrame.FilePath %q, got %q", expectedPath, routine.CreatedByFrame.FilePath)
						}
						expectedLine := 69
						if routine.CreatedByFrame.LineNumber != expectedLine {
							t.Errorf("Expected CreatedByFrame.LineNumber %d, got %d", expectedLine, routine.CreatedByFrame.LineNumber)
						}
					}
				}
			} else {
				if routine.CreatedByFrame != nil {
					t.Errorf("Expected CreatedByFrame to be nil, but it was set")
				}
			}
		})
	}
}

func TestParseFileLine(t *testing.T) {
	tests := []struct {
		name             string
		fileLine         string
		expectSuccess    bool
		expectedFilePath string
		expectedLine     int
		expectedPCOffset string
	}{
		{
			name:             "Standard file line with PC offset",
			fileLine:         "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go:165 +0x1fc",
			expectSuccess:    true,
			expectedFilePath: "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go",
			expectedLine:     165,
			expectedPCOffset: "+0x1fc",
		},
		{
			name:             "File line without PC offset",
			fileLine:         "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go:6329",
			expectSuccess:    true,
			expectedFilePath: "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go",
			expectedLine:     6329,
			expectedPCOffset: "",
		},
		{
			name:             "File line with leading whitespace",
			fileLine:         "  /Users/mike/work/outrig/pkg/rpc/rpcrouter.go:326 +0x14c",
			expectSuccess:    true,
			expectedFilePath: "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go",
			expectedLine:     326,
			expectedPCOffset: "+0x14c",
		},
		{
			name:             "File line with different PC offset format",
			fileLine:         "/Users/mike/work/outrig/pkg/collector/logprocess/loginitimpl.go:69 +0x3dc",
			expectSuccess:    true,
			expectedFilePath: "/Users/mike/work/outrig/pkg/collector/logprocess/loginitimpl.go",
			expectedLine:     69,
			expectedPCOffset: "+0x3dc",
		},
		{
			name:          "Invalid file line - no line number",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go",
			expectSuccess: false,
		},
		{
			name:          "Invalid file line - not a go file",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.c:123",
			expectSuccess: false,
		},
		{
			name:          "Invalid file line - non-numeric line number",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go:abc",
			expectSuccess: false,
		},
		{
			name:          "Empty file line",
			fileLine:      "",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, lineNumber, pcOffset, ok := parseFileLine(tt.fileLine)

			if ok != tt.expectSuccess {
				t.Fatalf("parseFileLine() success = %v, expected %v", ok, tt.expectSuccess)
			}

			if !tt.expectSuccess {
				return
			}

			if filePath != tt.expectedFilePath {
				t.Errorf("FilePath = %q, expected %q", filePath, tt.expectedFilePath)
			}

			if lineNumber != tt.expectedLine {
				t.Errorf("LineNumber = %d, expected %d", lineNumber, tt.expectedLine)
			}

			if pcOffset != tt.expectedPCOffset {
				t.Errorf("PCOffset = %q, expected %q", pcOffset, tt.expectedPCOffset)
			}
		})
	}
}

func TestParseFuncLine(t *testing.T) {
	tests := []struct {
		name             string
		funcLine         string
		expectSuccess    bool
		expectedPackage  string
		expectedFuncName string
		expectedArgs     string
	}{
		{
			name:             "Method with pointer receiver",
			funcLine:         "github.com/outrigdev/outrig/pkg/collector/runtimestats.(*RuntimeStatsCollector).Enable.func1()",
			expectSuccess:    true,
			expectedPackage:  "github.com/outrigdev/outrig/pkg/collector/runtimestats",
			expectedFuncName: "(*RuntimeStatsCollector).Enable.func1",
			expectedArgs:     "",
		},
		{
			name:             "Method with arguments",
			funcLine:         "github.com/outrigdev/outrig/pkg/collector/runtimestats.(*RuntimeStatsCollector).CollectRuntimeStats(0x14000136400)",
			expectSuccess:    true,
			expectedPackage:  "github.com/outrigdev/outrig/pkg/collector/runtimestats",
			expectedFuncName: "(*RuntimeStatsCollector).CollectRuntimeStats",
			expectedArgs:     "0x14000136400",
		},
		{
			name:             "Method with ellipsis",
			funcLine:         "main.Foo.footest(...)",
			expectSuccess:    true,
			expectedPackage:  "main",
			expectedFuncName: "Foo.footest",
			expectedArgs:     "...",
		},
		{
			name:             "Simple function",
			funcLine:         "runtime.doInit(0x12f7be0, ...)",
			expectSuccess:    true,
			expectedPackage:  "runtime",
			expectedFuncName: "doInit",
			expectedArgs:     "0x12f7be0, ...",
		},
		{
			name:             "Function without arguments",
			funcLine:         "main.main()",
			expectSuccess:    true,
			expectedPackage:  "main",
			expectedFuncName: "main",
			expectedArgs:     "",
		},
		{
			name:          "Invalid function line - no dot",
			funcLine:      "invalidfunctionline",
			expectSuccess: false,
		},
		{
			name:          "Invalid function line - dot at start",
			funcLine:      ".invalidfunctionline",
			expectSuccess: false,
		},
		{
			name:             "Function with complex arguments",
			funcLine:         "internal/poll.(*FD).Read(0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd})",
			expectSuccess:    true,
			expectedPackage:  "internal/poll",
			expectedFuncName: "(*FD).Read",
			expectedArgs:     "0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd}",
		},
		{
			name:             "Method with value receiver",
			funcLine:         "time.Time.Add(0x140003801e0, 0x140003ae723)",
			expectSuccess:    true,
			expectedPackage:  "time",
			expectedFuncName: "Time.Add",
			expectedArgs:     "0x140003801e0, 0x140003ae723",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageName, funcName, args, ok := parseFuncLine(tt.funcLine, true)

			if ok != tt.expectSuccess {
				t.Fatalf("parseFuncLine() success = %v, expected %v", ok, tt.expectSuccess)
			}

			if !tt.expectSuccess {
				return
			}

			if packageName != tt.expectedPackage {
				t.Errorf("Package = %q, expected %q", packageName, tt.expectedPackage)
			}

			if funcName != tt.expectedFuncName {
				t.Errorf("FuncName = %q, expected %q", funcName, tt.expectedFuncName)
			}

			if args != tt.expectedArgs {
				t.Errorf("Args = %q, expected %q", args, tt.expectedArgs)
			}
		})
	}
}

func TestParseCreatedByFrame(t *testing.T) {
	tests := []struct {
		name             string
		funcLine         string
		fileLine         string
		expectSuccess    bool
		expectedPackage  string
		expectedFuncName string
		expectedGoId     int
	}{
		{
			name:             "Standard created by line with goroutine ID",
			funcLine:         "created by github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute in goroutine 327",
			fileLine:         "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:315 +0x3cc",
			expectSuccess:    true,
			expectedPackage:  "github.com/outrigdev/outrig/pkg/rpc",
			expectedFuncName: "(*WshRouter).RegisterRoute",
			expectedGoId:     327,
		},
		{
			name:             "Created by line with complex package name",
			funcLine:         "created by github.com/outrigdev/outrig/pkg/collector/logprocess.(*LogCollector).initInternal in goroutine 1",
			fileLine:         "/Users/mike/work/outrig/pkg/collector/logprocess/loginitimpl.go:69 +0x3dc",
			expectSuccess:    true,
			expectedPackage:  "github.com/outrigdev/outrig/pkg/collector/logprocess",
			expectedFuncName: "(*LogCollector).initInternal",
			expectedGoId:     1,
		},
		{
			name:             "Created by line without file line",
			funcLine:         "created by main.main in goroutine 1",
			fileLine:         "",
			expectSuccess:    true,
			expectedPackage:  "main",
			expectedFuncName: "main",
			expectedGoId:     1,
		},
		{
			name:          "Invalid created by line - missing prefix",
			funcLine:      "github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute in goroutine 327",
			fileLine:      "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:315 +0x3cc",
			expectSuccess: false,
		},
		{
			name:          "Invalid created by line - invalid function format",
			funcLine:      "created by invalidfunction in goroutine 1",
			fileLine:      "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:315 +0x3cc",
			expectSuccess: false,
		},
		{
			name:          "Invalid created by line - invalid goroutine ID",
			funcLine:      "created by main.main in goroutine abc",
			fileLine:      "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:315 +0x3cc",
			expectSuccess: false,
		},
		{
			name:          "Invalid file line",
			funcLine:      "created by main.main in goroutine 1",
			fileLine:      "this is not a valid file line",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, goId, ok := parseCreatedByFrame(tt.funcLine, tt.fileLine)

			if ok != tt.expectSuccess {
				t.Fatalf("parseCreatedByFrame() success = %v, expected %v", ok, tt.expectSuccess)
			}

			if !tt.expectSuccess {
				return
			}

			if frame.Package != tt.expectedPackage {
				t.Errorf("Package = %q, expected %q", frame.Package, tt.expectedPackage)
			}

			if frame.FuncName != tt.expectedFuncName {
				t.Errorf("FuncName = %q, expected %q", frame.FuncName, tt.expectedFuncName)
			}

			if goId != tt.expectedGoId {
				t.Errorf("GoId = %d, expected %d", goId, tt.expectedGoId)
			}

			if tt.fileLine != "" {
				// Verify that file path and line number were parsed correctly
				if frame.FilePath == "" {
					t.Errorf("FilePath should not be empty")
				}

				if frame.LineNumber == 0 {
					t.Errorf("LineNumber should not be zero")
				}
			}
		})
	}
}
