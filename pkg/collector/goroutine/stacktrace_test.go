package goroutine

import (
	"testing"
)

func TestParseGoRoutineStackTrace(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		expectedCount        int
		expectedGoId         int64
		expectedPrimaryState string
		expectedFrames       int
		hasCreatedBy         bool
		expectedDurationMs   int64    // Expected duration in milliseconds
		expectedExtraStates  []string // Expected extra states
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
			expectedCount:        1,
			expectedGoId:         38,
			expectedPrimaryState: "IO wait",
			expectedFrames:       15, // Actual number of frame lines parsed
			hasCreatedBy:         true,
			expectedDurationMs:   0,   // No duration
			expectedExtraStates:  nil, // No extra states
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
			expectedCount:        1,
			expectedGoId:         338,
			expectedPrimaryState: "chan receive",
			expectedFrames:       5, // Actual number of frame lines parsed
			hasCreatedBy:         true,
			expectedDurationMs:   101 * 60 * 1000, // 101 minutes in milliseconds
			expectedExtraStates:  nil,             // No extra states besides duration
		},
		{
			name: "goroutine 1 with no created by",
			input: `goroutine 1 [chan receive, 105 minutes]:
main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
			expectedCount:        1,
			expectedGoId:         1,
			expectedPrimaryState: "chan receive",
			expectedFrames:       2, // 1 frame * 2 lines
			hasCreatedBy:         false,
			expectedDurationMs:   105 * 60 * 1000, // 105 minutes in milliseconds
			expectedExtraStates:  nil,             // No extra states besides duration
		},
		{
			name: "goroutine with multiple extra states",
			input: `goroutine 42 [chan receive, 3 minutes, locked to thread]:
main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
			expectedCount:        1,
			expectedGoId:         42,
			expectedPrimaryState: "chan receive",
			expectedFrames:       2, // 1 frame * 2 lines
			hasCreatedBy:         false,
			expectedDurationMs:   3 * 60 * 1000,                // 3 minutes in milliseconds
			expectedExtraStates:  []string{"locked to thread"}, // Extra state
		},
		{
			name: "goroutine with lock info",
			input: `goroutine 55 [semacquire, 2 minutes]:
main.main()
	/Users/mike/work/outrig/server/main-server.go:291 +0x714`,
			expectedCount:        1,
			expectedGoId:         55,
			expectedPrimaryState: "semacquire",
			expectedFrames:       2, // 1 frame * 2 lines
			hasCreatedBy:         false,
			expectedDurationMs:   2 * 60 * 1000, // 2 minutes in milliseconds
			expectedExtraStates:  nil,           // No extra states besides duration
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGoRoutineStackTrace(tt.input)
			if err != nil {
				t.Fatalf("ParseGoRoutineStackTrace returned error: %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d goroutines, got %d", tt.expectedCount, len(result))
			}

			if len(result) == 0 {
				return
			}

			routine := result[0]

			if routine.GoId != tt.expectedGoId {
				t.Errorf("Expected GoId %d, got %d", tt.expectedGoId, routine.GoId)
			}

			// Check that PrimaryState matches the expected state
			if routine.PrimaryState != tt.expectedPrimaryState {
				t.Errorf("Expected PrimaryState %q, got %q", tt.expectedPrimaryState, routine.PrimaryState)
			}

			// Check duration if expected
			if routine.DurationMs != tt.expectedDurationMs {
				t.Errorf("Expected DurationMs %d, got %d", tt.expectedDurationMs, routine.DurationMs)
			}

			// Check extra states if expected
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

			if len(routine.Frames) != tt.expectedFrames {
				t.Errorf("Expected %d frame lines, got %d", tt.expectedFrames, len(routine.Frames))
			}

			if tt.hasCreatedBy && routine.CreatedBy == "" {
				t.Errorf("Expected CreatedBy to be set, but it was empty")
			}

			if !tt.hasCreatedBy && routine.CreatedBy != "" {
				t.Errorf("Expected CreatedBy to be empty, but got %q", routine.CreatedBy)
			}
		})
	}
}
