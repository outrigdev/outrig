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

// GoRoutineSection represents a preprocessed section of a goroutine stack trace
type GoRoutineSection struct {
	HeaderLine  string     // The goroutine header line (e.g., "goroutine 1 [running]:")
	StackFrames [][]string // Stack frames, where each frame is 1 or 2 lines
	CreatedBy   []string   // The "created by" information (can be multiple lines)
}

// preprocessStackTrace splits a stack trace into sections for each goroutine
// and groups the lines into header, stack frames, and created by sections
func preprocessStackTrace(stackTrace string) []GoRoutineSection {
	// Split the stack trace into lines (don't trim yet to preserve indentation)
	lines := strings.Split(stackTrace, "\n")

	var sections []GoRoutineSection
	var currentSection *GoRoutineSection
	var currentFrame []string

	headerRegex := regexp.MustCompile(`^goroutine\s+\d+\s+\[.*\]:$`)

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is a goroutine header line
		if headerRegex.MatchString(line) {
			// If we have a current section, add it to the result
			if currentSection != nil {
				// Add any remaining frame to the current section
				if len(currentFrame) > 0 {
					currentSection.StackFrames = append(currentSection.StackFrames, currentFrame)
				}
				sections = append(sections, *currentSection)
			}

			// Start a new section
			currentSection = &GoRoutineSection{
				HeaderLine:  strings.TrimSpace(line),
				StackFrames: [][]string{},
				CreatedBy:   []string{},
			}
			currentFrame = nil
			continue
		}

		// Skip if we haven't found a goroutine header yet
		if currentSection == nil {
			continue
		}

		// Check if this is a "created by" line
		if strings.HasPrefix(strings.TrimSpace(line), "created by ") {
			// Add any remaining frame to the current section
			if len(currentFrame) > 0 {
				currentSection.StackFrames = append(currentSection.StackFrames, currentFrame)
				currentFrame = nil
			}

			// Add this line to the created by section
			currentSection.CreatedBy = append(currentSection.CreatedBy, strings.TrimSpace(line))

			// Check if the next line is indented (part of the created by frame)
			if i+1 < len(lines) && len(lines[i+1]) > 0 && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
				currentSection.CreatedBy = append(currentSection.CreatedBy, strings.TrimSpace(lines[i+1]))
				i++ // Skip the next line since we've processed it
			}

			continue
		}

		// Check if this line is indented (part of a stack frame)
		trimmedLine := strings.TrimSpace(line)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// This is the second line of a frame
			if len(currentFrame) == 1 {
				currentFrame = append(currentFrame, trimmedLine)
				currentSection.StackFrames = append(currentSection.StackFrames, currentFrame)
				currentFrame = nil
			} else {
				// This shouldn't happen in a well-formed stack trace, but handle it anyway
				currentFrame = []string{trimmedLine}
			}
		} else {
			// This is the first line of a new frame
			if len(currentFrame) > 0 {
				// Add the previous frame (which only had one line)
				currentSection.StackFrames = append(currentSection.StackFrames, currentFrame)
			}
			currentFrame = []string{trimmedLine}
		}
	}

	// Add the last section if there is one
	if currentSection != nil {
		// Add any remaining frame
		if len(currentFrame) > 0 {
			currentSection.StackFrames = append(currentSection.StackFrames, currentFrame)
		}
		sections = append(sections, *currentSection)
	}

	return sections
}

// ParseGoRoutineStackTrace parses a Go routine stack trace string into a struct
func ParseGoRoutineStackTrace(stackTrace string) ([]ParsedGoRoutine, error) {
	var result []ParsedGoRoutine

	// Preprocess the stack trace into sections
	sections := preprocessStackTrace(stackTrace)

	headerRegex := regexp.MustCompile(`^goroutine\s+(\d+)\s+\[(.*)\]:$`)

	for _, section := range sections {
		// Parse the goroutine header
		match := headerRegex.FindStringSubmatch(section.HeaderLine)
		if match == nil {
			continue // Skip if header doesn't match expected format
		}

		// Parse the goroutine ID and state
		goId, _ := strconv.ParseInt(match[1], 10, 64)
		state := match[2]

		// Create a new routine
		routine := &ParsedGoRoutine{
			GoId:         goId,
			RawState:     state,
			ParsedFrames: []Frame{},
		}

		// Parse the state components
		parseStateComponents(routine)

		// Parse stack frames
		for _, frameLines := range section.StackFrames {
			if len(frameLines) == 2 {
				if frame, ok := parseFrame(frameLines[0], frameLines[1]); ok {
					routine.ParsedFrames = append(routine.ParsedFrames, frame)
				}
			} else if len(frameLines) == 1 {
				// Handle single-line frames if they exist
				// This is just a safeguard; well-formed stack traces should have 2 lines per frame
			}
		}

		// Parse created by information
		if len(section.CreatedBy) > 0 {
			parseCreatedBySection(section.CreatedBy, routine)
		}

		result = append(result, *routine)
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

// parseCreatedBySection parses the "created by" section of a goroutine stack trace
// It extracts the goroutine ID that created the current goroutine and the function/file information
func parseCreatedBySection(createdByLines []string, routine *ParsedGoRoutine) {
	if len(createdByLines) == 0 {
		return
	}

	line := createdByLines[0]

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
	if len(createdByLines) > 1 && strings.Contains(createdByLines[1], ".go:") {
		// Use the existing parseFrame function to parse the created by frame
		if frame, ok := parseFrame(funcName, createdByLines[1]); ok {
			routine.CreatedByFrame = &frame
		}
	}
}

// parseCreatedBy parses the "created by" information in a goroutine stack trace
// It extracts the goroutine ID that created the current goroutine and the function/file information
// Returns the updated currentFrame slice and the next line index to process
func parseCreatedBy(line string, lines []string, lineIndex int, currentRoutine *ParsedGoRoutine, currentFrame []string) ([]string, int) {
	// Extract the goroutine ID
	goIdRegex := regexp.MustCompile(`in goroutine (\d+)`)
	if match := goIdRegex.FindStringSubmatch(line); match != nil {
		if goId, err := strconv.ParseInt(match[1], 10, 64); err == nil {
			currentRoutine.CreatedByGoId = goId
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

	// The next line might be the file location for the created by frame
	if lineIndex < len(lines) {
		fileLine := lines[lineIndex]
		lineIndex++ // Move to the next line

		// If this looks like a file line, parse it as part of the created by frame
		if strings.Contains(fileLine, ".go:") {
			// Use the existing parseFrame function to parse the created by frame
			if frame, ok := parseFrame(funcName, fileLine); ok {
				currentRoutine.CreatedByFrame = &frame
			}
		} else {
			// If it's not a file line, we need to process it as a normal line
			// by putting it back into processing
			currentFrame = append(currentFrame, fileLine)
		}
	}

	return currentFrame, lineIndex
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
