package goroutine

import (
	"testing"
)

func TestParseFrame(t *testing.T) {
	tests := []struct {
		name          string
		funcLine      string
		fileLine      string
		expectSuccess bool
		expectedFrame Frame
	}{
		{
			name:          "Method with receiver",
			funcLine:      "internal/poll.(*FD).Read(0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd})",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go:165 +0x1fc",
			expectSuccess: true,
			expectedFrame: Frame{
				Package:    "internal/poll",
				Receiver:   "(*FD)",
				FuncName:   "Read",
				Args:       "(0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd})",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go",
				LineNumber: 165,
				PCOffset:   "+0x1fc",
				FuncLine:   "internal/poll.(*FD).Read(0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd})",
				FileLine:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go:165 +0x1fc",
			},
		},
		{
			name:          "Function without receiver",
			funcLine:      "runtime.doInit(0x12f7be0)",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go:6329",
			expectSuccess: true,
			expectedFrame: Frame{
				Package:    "runtime",
				Receiver:   "",
				FuncName:   "doInit",
				Args:       "(0x12f7be0)",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go",
				LineNumber: 6329,
				PCOffset:   "",
				FuncLine:   "runtime.doInit(0x12f7be0)",
				FileLine:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/runtime/proc.go:6329",
			},
		},
		{
			name:          "Function with dots in package name",
			funcLine:      "github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute.func2()",
			fileLine:      "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:326 +0x14c",
			expectSuccess: true,
			expectedFrame: Frame{
				Package:    "github.com/outrigdev/outrig/pkg/rpc",
				Receiver:   "(*WshRouter)",
				FuncName:   "RegisterRoute.func2",
				Args:       "()",
				FilePath:   "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go",
				LineNumber: 326,
				PCOffset:   "+0x14c",
				FuncLine:   "github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute.func2()",
				FileLine:   "/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:326 +0x14c",
			},
		},
		{
			name:          "Function with ellipsis",
			funcLine:      "internal/poll.(*pollDesc).waitRead(...)",
			fileLine:      "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_poll_runtime.go:89",
			expectSuccess: true,
			expectedFrame: Frame{
				Package:    "internal/poll",
				Receiver:   "(*pollDesc)",
				FuncName:   "waitRead",
				Args:       "(...)",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_poll_runtime.go",
				LineNumber: 89,
				PCOffset:   "",
				FuncLine:   "internal/poll.(*pollDesc).waitRead(...)",
				FileLine:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_poll_runtime.go:89",
			},
		},
		{
			name:          "Main function",
			funcLine:      "main.main()",
			fileLine:      "/Users/mike/work/outrig/server/main-server.go:291 +0x714",
			expectSuccess: true,
			expectedFrame: Frame{
				Package:    "main",
				Receiver:   "",
				FuncName:   "main",
				Args:       "()",
				FilePath:   "/Users/mike/work/outrig/server/main-server.go",
				LineNumber: 291,
				PCOffset:   "+0x714",
				FuncLine:   "main.main()",
				FileLine:   "/Users/mike/work/outrig/server/main-server.go:291 +0x714",
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
			expectedFrame: Frame{
				Package:    "time",
				Receiver:   "Time",
				FuncName:   "Add",
				Args:       "(0x140003801e0, 0x140003ae723)",
				FilePath:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/time/time.go",
				LineNumber: 1076,
				PCOffset:   "+0x1a4",
				FuncLine:   "time.Time.Add(0x140003801e0, 0x140003ae723)",
				FileLine:   "/opt/homebrew/Cellar/go/1.23.4/libexec/src/time/time.go:1076 +0x1a4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, ok := parseFrame(tt.funcLine, tt.fileLine)

			if ok != tt.expectSuccess {
				t.Fatalf("parseFrame() success = %v, expected %v", ok, tt.expectSuccess)
			}

			if !tt.expectSuccess {
				return
			}

			if frame.Package != tt.expectedFrame.Package {
				t.Errorf("Package = %q, expected %q", frame.Package, tt.expectedFrame.Package)
			}

			if frame.Receiver != tt.expectedFrame.Receiver {
				t.Errorf("Receiver = %q, expected %q", frame.Receiver, tt.expectedFrame.Receiver)
			}

			if frame.FuncName != tt.expectedFrame.FuncName {
				t.Errorf("FuncName = %q, expected %q", frame.FuncName, tt.expectedFrame.FuncName)
			}

			if frame.Args != tt.expectedFrame.Args {
				t.Errorf("Args = %q, expected %q", frame.Args, tt.expectedFrame.Args)
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
		expectedExtraStates []string
	}{
		{
			name:                "Simple state",
			rawState:            "running",
			expectedPrimary:     "running",
			expectedDurationMs:  0,
			expectedExtraStates: nil,
		},
		{
			name:                "State with duration",
			rawState:            "chan receive, 101 minutes",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  101 * 60 * 1000,
			expectedExtraStates: nil,
		},
		{
			name:                "State with extra states",
			rawState:            "chan receive, locked to thread",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  0,
			expectedExtraStates: []string{"locked to thread"},
		},
		{
			name:                "State with duration and extra states",
			rawState:            "chan receive, 3 minutes, locked to thread",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  3 * 60 * 1000,
			expectedExtraStates: []string{"locked to thread"},
		},
		{
			name:                "State with multiple extra states",
			rawState:            "chan receive, locked to thread, syscall",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  0,
			expectedExtraStates: []string{"locked to thread", "syscall"},
		},
		{
			name:                "State with seconds duration",
			rawState:            "chan receive, 45 seconds",
			expectedPrimary:     "chan receive",
			expectedDurationMs:  45 * 1000,
			expectedExtraStates: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primaryState, durationMs, extraStates := parseStateComponents(tt.rawState)

			if primaryState != tt.expectedPrimary {
				t.Errorf("Expected primary state %q, got %q", tt.expectedPrimary, primaryState)
			}

			if durationMs != tt.expectedDurationMs {
				t.Errorf("Expected duration %d ms, got %d ms", tt.expectedDurationMs, durationMs)
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

func TestParseHeaderLine(t *testing.T) {
	tests := []struct {
		name                 string
		headerLine           string
		expectError          bool
		expectedGoId         int64
		expectedRawState     string
		expectedPrimaryState string
		expectedDurationMs   int64
		expectedExtraStates  []string
	}{
		{
			name:                 "Valid header with simple state",
			headerLine:           "goroutine 38 [IO wait]:",
			expectError:          false,
			expectedGoId:         38,
			expectedRawState:     "IO wait",
			expectedPrimaryState: "IO wait",
			expectedDurationMs:   0,
			expectedExtraStates:  nil,
		},
		{
			name:                 "Valid header with duration",
			headerLine:           "goroutine 338 [chan receive, 101 minutes]:",
			expectError:          false,
			expectedGoId:         338,
			expectedRawState:     "chan receive, 101 minutes",
			expectedPrimaryState: "chan receive",
			expectedDurationMs:   101 * 60 * 1000,
			expectedExtraStates:  nil,
		},
		{
			name:                 "Valid header with extra states",
			headerLine:           "goroutine 42 [chan receive, locked to thread]:",
			expectError:          false,
			expectedGoId:         42,
			expectedRawState:     "chan receive, locked to thread",
			expectedPrimaryState: "chan receive",
			expectedDurationMs:   0,
			expectedExtraStates:  []string{"locked to thread"},
		},
		{
			name:                 "Valid header with duration and extra states",
			headerLine:           "goroutine 42 [chan receive, 3 minutes, locked to thread]:",
			expectError:          false,
			expectedGoId:         42,
			expectedRawState:     "chan receive, 3 minutes, locked to thread",
			expectedPrimaryState: "chan receive",
			expectedDurationMs:   3 * 60 * 1000,
			expectedExtraStates:  []string{"locked to thread"},
		},
		{
			name:        "Invalid header format",
			headerLine:  "not a valid goroutine header",
			expectError: true,
		},
		{
			name:        "Invalid goroutine ID",
			headerLine:  "goroutine abc [running]:",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routine, err := parseHeaderLine(tt.headerLine)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if routine.GoId != tt.expectedGoId {
				t.Errorf("Expected GoId %d, got %d", tt.expectedGoId, routine.GoId)
			}

			if routine.RawState != tt.expectedRawState {
				t.Errorf("Expected RawState %q, got %q", tt.expectedRawState, routine.RawState)
			}

			if routine.PrimaryState != tt.expectedPrimaryState {
				t.Errorf("Expected PrimaryState %q, got %q", tt.expectedPrimaryState, routine.PrimaryState)
			}

			if routine.StateDurationMs != tt.expectedDurationMs {
				t.Errorf("Expected StateDurationMs %d, got %d", tt.expectedDurationMs, routine.StateDurationMs)
			}

			if tt.expectedExtraStates == nil {
				if len(routine.ExtraStates) > 0 {
					t.Errorf("Expected no extra states, got %v", routine.ExtraStates)
				}
			} else {
				if len(routine.ExtraStates) != len(tt.expectedExtraStates) {
					t.Errorf("Expected %d extra states, got %d", len(tt.expectedExtraStates), len(routine.ExtraStates))
				} else {
					for i, expected := range tt.expectedExtraStates {
						if routine.ExtraStates[i] != expected {
							t.Errorf("Expected extra state %q at index %d, got %q", expected, i, routine.ExtraStates[i])
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
			input: `goroutine 38 [IO wait]:
internal/poll.runtime_pollWait(0x1010b0a98, 0x72)
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
			input: `goroutine 338 [chan receive, 101 minutes]:
github.com/outrigdev/outrig/pkg/rpc.(*WshRpcProxy).RecvRpcMessage(0x103bab9e0?)
	/Users/mike/work/outrig/pkg/rpc/rpcproxy.go:34 +0x2c
github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute.func2()
	/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:326 +0x14c
created by github.com/outrigdev/outrig/pkg/rpc.(*WshRouter).RegisterRoute in goroutine 327
	/Users/mike/work/outrig/pkg/rpc/rpcrouter.go:315 +0x3cc`,
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
			input: `goroutine 1 [chan receive, 105 minutes]:
main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
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
			input: `goroutine 42 [chan receive, 3 minutes, locked to thread]:
main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
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
			input: `goroutine 55 [semacquire, 2 minutes]:
main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
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
			routine, err := ParseGoRoutineStackTrace(tt.input)
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
