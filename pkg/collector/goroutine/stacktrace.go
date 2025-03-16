package goroutine

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// Frame represents a single frame in a goroutine stack trace
type Frame struct {
	// Function information
	Package  string // The package name (e.g., "internal/poll")
	Receiver string // The receiver type if it's a method (e.g., "(*FD)")
	FuncName string // Just the function/method name (e.g., "Read")

	// Arguments
	Args string // Raw argument string (e.g., "(0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd})")

	// Source file information
	FilePath   string // Full path to the source file (e.g., "/opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go")
	LineNumber int    // Line number in the source file (e.g., 165)
	PCOffset   string // Program counter offset (e.g., "+0x1fc")

	// Raw lines for reference
	FuncLine string // The raw function call line
	FileLine string // The raw file location line
}

// ParsedGoRoutine represents a parsed goroutine stack trace
type ParsedGoRoutine struct {
	GoId         int64
	RawState     string   // The complete state information
	PrimaryState string   // The first part of the state (before any commas)
	DurationMs   int64    // Duration in milliseconds (if available)
	ExtraStates  []string // Array of additional state information
	Frames       []string // Original raw frame lines (for backward compatibility)
	ParsedFrames []Frame  // Structured frame information
	CreatedBy    string
}

// ParseGoRoutineStackTrace parses a Go routine stack trace string into a struct
func ParseGoRoutineStackTrace(stackTrace string) ([]ParsedGoRoutine, error) {
	var result []ParsedGoRoutine

	// Process the stack trace line by line
	scanner := bufio.NewScanner(strings.NewReader(stackTrace))

	var currentRoutine *ParsedGoRoutine
	var currentFrame []string

	headerRegex := regexp.MustCompile(`^goroutine\s+(\d+)\s+\[(.*)\]:$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this is a goroutine header line
		if match := headerRegex.FindStringSubmatch(line); match != nil {
			// If we have a current routine, add it to the result
			if currentRoutine != nil {
				// Handle any remaining frame
				if len(currentFrame) > 0 {
					currentRoutine.Frames = append(currentRoutine.Frames, currentFrame...)

					// Parse the frame if we have both lines
					if len(currentFrame) == 2 {
						if frame, ok := parseFrame(currentFrame[0], currentFrame[1]); ok {
							currentRoutine.ParsedFrames = append(currentRoutine.ParsedFrames, frame)
						}
					}

					currentFrame = nil
				}

				result = append(result, *currentRoutine)
			}

			// Parse the new goroutine header
			goId, _ := strconv.ParseInt(match[1], 10, 64)
			state := match[2]

			// Create a new routine
			currentRoutine = &ParsedGoRoutine{
				GoId:         goId,
				RawState:     state,
				ParsedFrames: []Frame{},
			}

			// Parse the state components
			parseStateComponents(currentRoutine)

			// Reset the current frame
			currentFrame = nil
			continue
		}

		// Skip if we haven't found a goroutine header yet
		if currentRoutine == nil {
			continue
		}

		// Process the line
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Check if this is a "created by" line
		if strings.HasPrefix(line, "created by ") {
			currentRoutine.CreatedBy = line
			continue
		}

		// Add to current frame
		currentFrame = append(currentFrame, line)

		// If we have 2 lines in the current frame, add it to frames and reset
		if len(currentFrame) == 2 {
			currentRoutine.Frames = append(currentRoutine.Frames, currentFrame...)

			// Parse the frame
			if frame, ok := parseFrame(currentFrame[0], currentFrame[1]); ok {
				currentRoutine.ParsedFrames = append(currentRoutine.ParsedFrames, frame)
			}

			currentFrame = nil
		}
	}

	// Add the last routine if there is one
	if currentRoutine != nil {
		// Handle any remaining frame
		if len(currentFrame) > 0 {
			currentRoutine.Frames = append(currentRoutine.Frames, currentFrame...)

			// Parse the frame if we have both lines
			if len(currentFrame) == 2 {
				if frame, ok := parseFrame(currentFrame[0], currentFrame[1]); ok {
					currentRoutine.ParsedFrames = append(currentRoutine.ParsedFrames, frame)
				}
			}
		}

		result = append(result, *currentRoutine)
	}

	return result, nil
}

// parseFrame parses a pair of stack trace lines into a Frame struct
// The first line contains the function call, the second line contains the file path and line number
func parseFrame(funcLine, fileLine string) (Frame, bool) {
	frame := Frame{
		FuncLine: funcLine,
		FileLine: fileLine,
	}
	
	// Use regular expressions to parse the function line
	// We need to handle several cases:
	// 1. Method with pointer receiver: internal/poll.(*FD).Read(0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd})
	// 2. Method with value receiver: time.Time.Add(0x140003801e0, 0x140003ae723)
	// 3. Function without receiver: runtime.doInit(0x12f7be0)
	
	// Pattern for method with pointer receiver: package.(*Type).Method(args)
	pointerReceiverRegex := regexp.MustCompile(`^(.+)\.(\(\*[^)]+\))\.([^(]+)(\(.+)$`)
	
	// Pattern for method with value receiver: package.Type.Method(args)
	valueReceiverRegex := regexp.MustCompile(`^(.+)\.([^.(]+)\.([^(]+)(\(.+)$`)
	
	// Pattern for function without receiver: package.Function(args)
	functionRegex := regexp.MustCompile(`^(.+)\.([^.(]+)(\(.+)$`)
	
	// Try to match with pointer receiver pattern first
	if match := pointerReceiverRegex.FindStringSubmatch(funcLine); match != nil {
		frame.Package = match[1]
		frame.Receiver = match[2]
		frame.FuncName = match[3]
		frame.Args = match[4]
	} else if match := valueReceiverRegex.FindStringSubmatch(funcLine); match != nil {
		// Try to match with value receiver pattern
		frame.Package = match[1]
		frame.Receiver = match[2]
		frame.FuncName = match[3]
		frame.Args = match[4]
	} else if match := functionRegex.FindStringSubmatch(funcLine); match != nil {
		// Try to match with function pattern
		frame.Package = match[1]
		frame.FuncName = match[2]
		frame.Args = match[3]
	} else {
		// If none of the patterns match, return false
		return frame, false
	}

	// Parse file line
	// Example: /opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go:165 +0x1fc
	fileRegex := regexp.MustCompile(`^\s*(.*\.go):(\d+)(?:\s+(\+0x[0-9a-f]+))?$`)
	if match := fileRegex.FindStringSubmatch(fileLine); match != nil {
		frame.FilePath = match[1]
		if lineNum, err := strconv.Atoi(match[2]); err == nil {
			frame.LineNumber = lineNum
		}
		if len(match) > 3 && match[3] != "" {
			frame.PCOffset = match[3]
		}
	} else {
		// If we can't parse the file line, return false
		return frame, false
	}

	return frame, true
}

// parseStateComponents parses the RawState into its components
func parseStateComponents(routine *ParsedGoRoutine) {
	// Split the state by commas
	components := strings.Split(routine.RawState, ",")

	// The first component is always the primary state
	routine.PrimaryState = strings.TrimSpace(components[0])

	// Process additional components
	if len(components) > 1 {
		routine.ExtraStates = make([]string, 0, len(components)-1)

		for _, component := range components[1:] {
			component = strings.TrimSpace(component)

			// Check if this component is a duration
			if isDuration, durationMs := parseDuration(component); isDuration {
				routine.DurationMs = durationMs
			} else {
				routine.ExtraStates = append(routine.ExtraStates, component)
			}
		}
	}
}

// parseDuration attempts to parse a duration string and convert it to milliseconds
func parseDuration(s string) (bool, int64) {
	// Common duration formats in goroutine states
	type durationPattern struct {
		regex      *regexp.Regexp
		multiplier int64
	}

	patterns := []durationPattern{
		{regexp.MustCompile(`^(\d+)\s*minutes?$`), 60 * 1000},       // minutes to ms
		{regexp.MustCompile(`^(\d+)\s*seconds?$`), 1000},            // seconds to ms
		{regexp.MustCompile(`^(\d+)\s*hours?$`), 60 * 60 * 1000},    // hours to ms
		{regexp.MustCompile(`^(\d+)\s*milliseconds?$`), 1},          // ms to ms
		{regexp.MustCompile(`^(\d+)\s*nanoseconds?$`), 1 / 1000000}, // ns to ms (effectively 0 for small values)
	}

	for _, pattern := range patterns {
		if match := pattern.regex.FindStringSubmatch(s); match != nil {
			if value, err := strconv.ParseInt(match[1], 10, 64); err == nil {
				return true, value * pattern.multiplier
			}
		}
	}

	return false, 0
}
