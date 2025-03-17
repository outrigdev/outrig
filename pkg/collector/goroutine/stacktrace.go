package goroutine

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// Frame represents a single frame in a goroutine stack trace
type Frame struct {
	// Function information
	Package  string // The package name (e.g., "internal/poll")
	FuncName string // Just the function/method name, may include the receiver (e.g., "Read")
	FuncArgs string // Raw argument string, no parens (e.g., "0x140003801e0, {0x140003ae723, 0x8dd, 0x8dd}")

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

var headerRe = regexp.MustCompile(`^goroutine\s+(\d+)\s+\[(.*)\]:$`)

// parseHeaderLine parses a goroutine header line and returns a ParsedGoRoutine
func parseHeaderLine(headerLine string) (ParsedGoRoutine, error) {
	// Parse the goroutine header
	match := headerRe.FindStringSubmatch(headerLine)
	if match == nil {
		return ParsedGoRoutine{}, fmt.Errorf("invalid header format: %s", headerLine)
	}

	// Parse the goroutine ID and state
	goId, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return ParsedGoRoutine{}, fmt.Errorf("failed to parse goroutine ID: %v", err)
	}
	state := match[2]

	// Parse the state components
	primaryState, stateDurationMs, extraStates := parseStateComponents(state)

	// Create a new routine
	routine := ParsedGoRoutine{
		GoId:            goId,
		RawState:        state,
		PrimaryState:    primaryState,
		StateDurationMs: stateDurationMs,
		ExtraStates:     extraStates,
		ParsedFrames:    []Frame{},
	}

	return routine, nil
}

// ParseGoRoutineStackTrace parses a Go routine stack trace string into a struct
func ParseGoRoutineStackTrace(stackTrace string) (ParsedGoRoutine, error) {
	// Preprocess the stack trace
	preprocessed := preprocessStackTrace(stackTrace)

	// Return empty struct and error if header line is empty
	if preprocessed.HeaderLine == "" {
		return ParsedGoRoutine{}, fmt.Errorf("no goroutine header found in stack trace")
	}

	// Parse the header line
	routine, err := parseHeaderLine(preprocessed.HeaderLine)
	if err != nil {
		return ParsedGoRoutine{}, err
	}

	// Parse stack frames
	for _, frame := range preprocessed.StackFrames {
		if parsedFrame, ok := parseFrame(frame.FuncLine, frame.FileLine); ok {
			routine.ParsedFrames = append(routine.ParsedFrames, parsedFrame)
		}
	}

	// Parse created by information
	if preprocessed.CreatedBy.FuncLine != "" {
		frame, goId, ok := parseCreatedByFrame(preprocessed.CreatedBy.FuncLine, preprocessed.CreatedBy.FileLine)
		if ok {
			routine.CreatedByGoId = int64(goId)
			routine.CreatedByFrame = frame
		}
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
	var ok bool
	frame.Package, frame.FuncName, frame.FuncArgs, ok = parseFuncLine(funcLine)
	if !ok {
		return frame, false
	}
	// Parse file line
	if fileLine != "" {
		filePath, lineNumber, pcOffset, ok := parseFileLine(fileLine)
		if !ok {
			return frame, false
		}
		frame.FilePath = filePath
		frame.LineNumber = lineNumber
		frame.PCOffset = pcOffset
	}
	return frame, true
}

var inGoRoutineRe = regexp.MustCompile(`\s*in goroutine (\d+)`)

// parseCreatedByFrame parses the "created by" frame of a goroutine stack trace
// returns a Frame struct, goId, and a boolean indicating success
func parseCreatedByFrame(funcLine string, fileLine string) (*Frame, int, bool) {
	// the trick is just removing "created by" off the front and "in goroutine X" off the end
	if !strings.HasPrefix(funcLine, "created by ") {
		log.Printf("no created by in %s\n", funcLine)
		return nil, 0, false
	}
	funcLine = strings.TrimPrefix(funcLine, "created by ")
	// now parse the goroutine id and remove it
	var goId int
	match := inGoRoutineRe.FindStringSubmatch(funcLine)
	if match == nil {
		return nil, 0, false
	}
	funcLine = strings.TrimSuffix(funcLine, match[0])
	goId, err := strconv.Atoi(match[1])
	if err != nil {
		log.Printf("failed to parse goroutine ID: %v\n", err)
		return nil, 0, false
	}
	// now parse the frame
	frame, ok := parseFrame(funcLine, fileLine)
	if !ok {
		log.Printf("failed to parse created by frame: %q\n", funcLine)
		return nil, 0, false
	}
	log.Printf("parsed created by frame: %s\n", funcLine)
	return &frame, goId, true
}

// parseStateComponents parses a raw state string into its components
func parseStateComponents(rawState string) (string, int64, []string) {
	// Split the state by commas
	components := strings.Split(rawState, ",")

	// The first component is always the primary state
	primaryState := strings.TrimSpace(components[0])

	// Initialize variables for additional components
	var stateDurationMs int64
	var extraStates []string

	// Process additional components
	if len(components) > 1 {
		extraStates = make([]string, 0, len(components)-1)

		for _, component := range components[1:] {
			component = strings.TrimSpace(component)

			// Check if this component is a duration
			if isDuration, durationMs := parseDuration(component); isDuration {
				stateDurationMs = durationMs
			} else {
				extraStates = append(extraStates, component)
			}
		}
	}

	return primaryState, stateDurationMs, extraStates
}

// parseFuncLine parses a function line from a stack trace and extracts the package, function name, and args
// The function name includes the receiver if present
func parseFuncLine(funcLine string) (pkgName string, funcName string, funcArgs string, valid bool) {
	// first extract the arguments, we can find the arguments by the last parens

	if strings.HasSuffix(funcLine, ")") {
		// Find the last opening parenthesis
		lastOpenParenIdx := strings.LastIndex(funcLine, "(")
		if lastOpenParenIdx < 0 {
			return "", "", "", false
		}
		// Extract everything from the last opening parenthesis to the end
		funcArgs = funcLine[lastOpenParenIdx+1 : len(funcLine)-1]
		funcLine = funcLine[:lastOpenParenIdx]
	}
	// now we strip the package name
	// we find the final "/", and then the first "." after that
	finalSlashIdx := strings.LastIndex(funcLine, "/")
	if finalSlashIdx < 0 {
		finalSlashIdx = 0 // this is fine, for a package like "os" which won't have any slashes
	}
	firstDotIdx := strings.Index(funcLine[finalSlashIdx:], ".")
	if firstDotIdx < 0 {
		return "", "", "", false
	}
	firstDotIdx += finalSlashIdx // adjust for the offset of the last slash
	pkgName = funcLine[:firstDotIdx]
	funcLine = funcLine[firstDotIdx+1:] // remove the package name
	// now the funcName is what's left!
	funcName = funcLine
	return pkgName, funcName, funcArgs, true
}

// parseDuration attempts to parse a duration string and convert it to milliseconds
func parseDuration(s string) (bool, int64) {
	// Common duration formats in goroutine states
	type durationPattern struct {
		regex      *regexp.Regexp
		multiplier int64
	}

	patterns := []durationPattern{
		{regexp.MustCompile(`^(\d+)\s*days?$`), 24 * 60 * 60 * 1000}, // days to ms
		{regexp.MustCompile(`^(\d+)\s*hours?$`), 60 * 60 * 1000},     // hours to ms
		{regexp.MustCompile(`^(\d+)\s*minutes?$`), 60 * 1000},        // minutes to ms
		{regexp.MustCompile(`^(\d+)\s*seconds?$`), 1000},             // seconds to ms
		{regexp.MustCompile(`^(\d+)\s*(milliseconds?|ms)$`), 1},      // ms to ms
		{regexp.MustCompile(`^(\d+)\s*(microseconds?|us|µs)$`), 0},   // µs to ms (effectively 0 for small values)
		{regexp.MustCompile(`^(\d+)\s*(nanoseconds?|ns)$`), 0},       // ns to ms (effectively 0 for small values)
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
