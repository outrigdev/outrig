package goroutine

import (
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
	GoId            int64
	RawState        string   // The complete state information
	PrimaryState    string   // The first part of the state (before any commas)
	StateDurationMs int64    // Duration of state in milliseconds (if available)
	ExtraStates     []string // Array of additional state information
	ParsedFrames    []Frame  // Structured frame information
	CreatedByGoId   int64    // ID of the goroutine that created this one
	CreatedByFrame  *Frame   // Frame information for the creation point
}

// RawStackFrame represents a pair of lines in a stack trace
type RawStackFrame struct {
	FuncLine string // The function call line
	FileLine string // The file location line (may be empty)
}

// PreprocessedGoRoutineLines represents a preprocessed goroutine stack trace
type PreprocessedGoRoutineLines struct {
	HeaderLine  string          // The goroutine header line (e.g., "goroutine 1 [running]:")
	StackFrames []RawStackFrame // Stack frames, where each frame has a function line and optional file line
	CreatedBy   RawStackFrame   // The "created by" information
}

// preprocessStackTrace processes a goroutine stack trace and groups the lines into header, stack frames, and created by sections
func preprocessStackTrace(stackTrace string) PreprocessedGoRoutineLines {
	// Split the stack trace into lines (don't trim yet to preserve indentation)
	lines := strings.Split(stackTrace, "\n")

	result := PreprocessedGoRoutineLines{}
	var currentFuncLine string

	headerRegex := regexp.MustCompile(`^goroutine\s+\d+\s+\[.*\]:$`)

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is a goroutine header line
		if headerRegex.MatchString(line) {
			// Found the header line
			result.HeaderLine = strings.TrimSpace(line)
			currentFuncLine = ""
			continue
		}

		// Skip if we haven't found a goroutine header yet
		if result.HeaderLine == "" {
			continue
		}

		// Check if this is a "created by" line
		if strings.HasPrefix(strings.TrimSpace(line), "created by ") {
			// Add any remaining frame
			if currentFuncLine != "" {
				result.StackFrames = append(result.StackFrames, RawStackFrame{
					FuncLine: currentFuncLine,
					FileLine: "",
				})
				currentFuncLine = ""
			}

			// Set the created by function line
			createdByLine := strings.TrimSpace(line)
			result.CreatedBy.FuncLine = createdByLine

			// Check if the next line is indented (part of the created by frame)
			if i+1 < len(lines) && len(lines[i+1]) > 0 && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
				result.CreatedBy.FileLine = strings.TrimSpace(lines[i+1])
				i++ // Skip the next line since we've processed it
			}

			continue
		}

		// Check if this line is indented (part of a stack frame)
		trimmedLine := strings.TrimSpace(line)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// This is the second line of a frame
			if currentFuncLine != "" {
				result.StackFrames = append(result.StackFrames, RawStackFrame{
					FuncLine: currentFuncLine,
					FileLine: trimmedLine,
				})
				currentFuncLine = ""
			} else {
				// This shouldn't happen in a well-formed stack trace, but handle it anyway
				// Just treat it as a function line for now
				currentFuncLine = trimmedLine
			}
		} else {
			// This is the first line of a new frame
			if currentFuncLine != "" {
				// Add the previous frame (which only had one line)
				result.StackFrames = append(result.StackFrames, RawStackFrame{
					FuncLine: currentFuncLine,
					FileLine: "",
				})
			}
			currentFuncLine = trimmedLine
		}
	}

	// Add any remaining frame
	if currentFuncLine != "" {
		result.StackFrames = append(result.StackFrames, RawStackFrame{
			FuncLine: currentFuncLine,
			FileLine: "",
		})
	}

	return result
}

// ParseGoRoutineStackTrace parses a Go routine stack trace string into a struct
func ParseGoRoutineStackTrace(stackTrace string) (ParsedGoRoutine, error) {
	// Preprocess the stack trace
	preprocessed := preprocessStackTrace(stackTrace)

	headerRegex := regexp.MustCompile(`^goroutine\s+(\d+)\s+\[(.*)\]:$`)

	// Parse the goroutine header
	match := headerRegex.FindStringSubmatch(preprocessed.HeaderLine)
	if match == nil {
		return ParsedGoRoutine{}, nil // Return empty struct if header doesn't match expected format
	}

	// Parse the goroutine ID and state
	goId, _ := strconv.ParseInt(match[1], 10, 64)
	state := match[2]

	// Create a new routine
	routine := ParsedGoRoutine{
		GoId:         goId,
		RawState:     state,
		ParsedFrames: []Frame{},
	}

	// Parse the state components
	parseStateComponents(&routine)

	// Parse stack frames
	for _, frame := range preprocessed.StackFrames {
		if frame.FileLine != "" {
			if parsedFrame, ok := parseFrame(frame.FuncLine, frame.FileLine); ok {
				routine.ParsedFrames = append(routine.ParsedFrames, parsedFrame)
			}
		}
		// Skip frames that don't have a file line
	}

	// Parse created by information
	if preprocessed.CreatedBy.FuncLine != "" {
		parseCreatedByFrame(preprocessed.CreatedBy, &routine)
	}

	return routine, nil
}

// parseFileLine parses a file line from a stack trace
// Example: /opt/homebrew/Cellar/go/1.23.4/libexec/src/internal/poll/fd_unix.go:165 +0x1fc
// Returns the file path, line number, and PC offset
func parseFileLine(fileLine string) (string, int, string, bool) {
	fileRegex := regexp.MustCompile(`^\s*(.*\.go):(\d+)(?:\s+(\+0x[0-9a-f]+))?$`)
	if match := fileRegex.FindStringSubmatch(fileLine); match != nil {
		filePath := match[1]
		lineNumber, err := strconv.Atoi(match[2])
		if err != nil {
			return "", 0, "", false
		}

		pcOffset := ""
		if len(match) > 3 && match[3] != "" {
			pcOffset = match[3]
		}

		return filePath, lineNumber, pcOffset, true
	}

	return "", 0, "", false
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
	// The args part is optional for "created by" lines
	pointerReceiverRegex := regexp.MustCompile(`^(.+)\.(\(\*[^)]+\))\.([^(]+)(\(.+)?$`)

	// Pattern for method with value receiver: package.Type.Method(args)
	// The args part is optional for "created by" lines
	valueReceiverRegex := regexp.MustCompile(`^(.+)\.([^.(]+)\.([^(]+)(\(.+)?$`)

	// Pattern for function without receiver: package.Function(args)
	// The args part is optional for "created by" lines
	// This needs to handle package names with dots (e.g., github.com/outrigdev/outrig/pkg/collector/logprocess.initLogger)
	functionRegex := regexp.MustCompile(`^((?:[^.]+(?:\.[^.]+)*(?:/[^.]+)*)+)\.([^.(]+)(\(.+)?$`)

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
	filePath, lineNumber, pcOffset, ok := parseFileLine(fileLine)
	if !ok {
		return frame, false
	}

	frame.FilePath = filePath
	frame.LineNumber = lineNumber
	frame.PCOffset = pcOffset

	return frame, true
}

// parseCreatedByFrame parses the "created by" frame of a goroutine stack trace
// It extracts the goroutine ID that created the current goroutine and the function/file information
func parseCreatedByFrame(createdBy RawStackFrame, routine *ParsedGoRoutine) {
	if createdBy.FuncLine == "" {
		return
	}

	line := createdBy.FuncLine

	// Extract the goroutine ID
	goIdRegex := regexp.MustCompile(`in goroutine (\d+)`)
	if match := goIdRegex.FindStringSubmatch(line); match != nil {
		if goId, err := strconv.ParseInt(match[1], 10, 64); err == nil {
			routine.CreatedByGoId = goId
		}
	}

	// Extract the function name from the "created by" line
	// Format: "created by package.function in goroutine X"
	funcNameRegex := regexp.MustCompile(`created by (.+) in goroutine`)
	var funcName string
	if match := funcNameRegex.FindStringSubmatch(line); match != nil {
		funcName = match[1]
	} else {
		// If we can't extract with the regex, just take everything after "created by "
		funcName = strings.TrimPrefix(line, "created by ")
		// Remove " in goroutine X" if present
		if idx := strings.Index(funcName, " in goroutine "); idx > 0 {
			funcName = funcName[:idx]
		}
	}

	// Check if we have a file line
	if createdBy.FileLine != "" && strings.Contains(createdBy.FileLine, ".go:") {
		// Use the existing parseFrame function to parse the created by frame
		if frame, ok := parseFrame(funcName, createdBy.FileLine); ok {
			routine.CreatedByFrame = &frame
		}
	}
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
				routine.StateDurationMs = durationMs
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
		{regexp.MustCompile(`^(\d+)\s*minutes?$`), 60 * 1000},    // minutes to ms
		{regexp.MustCompile(`^(\d+)\s*seconds?$`), 1000},         // seconds to ms
		{regexp.MustCompile(`^(\d+)\s*hours?$`), 60 * 60 * 1000}, // hours to ms
		{regexp.MustCompile(`^(\d+)\s*milliseconds?$`), 1},       // ms to ms
		{regexp.MustCompile(`^(\d+)\s*nanoseconds?$`), 0},        // ns to ms (effectively 0 for small values)
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
