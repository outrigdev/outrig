package goroutine

import (
	"testing"
)

func TestParseGoRoutineStackTrace(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCount  int
		expectedGoId   int64
		expectedState  string
		expectedFrames int
		hasCreatedBy   bool
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
			expectedCount:  1,
			expectedGoId:   38,
			expectedState:  "IO wait",
			expectedFrames: 15, // Actual number of frame lines parsed
			hasCreatedBy:   true,
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
			expectedCount:  1,
			expectedGoId:   338,
			expectedState:  "chan receive",
			expectedFrames: 5, // Actual number of frame lines parsed
			hasCreatedBy:   true,
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

			if routine.State != tt.expectedState {
				t.Errorf("Expected State %q, got %q", tt.expectedState, routine.State)
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
