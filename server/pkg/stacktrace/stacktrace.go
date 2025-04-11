// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package stacktrace

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

type ParsedGoRoutine = rpctypes.ParsedGoRoutine
type StackFrame = rpctypes.StackFrame

// rawStackFrame represents a pair of lines in a stack trace
type rawStackFrame struct {
	FuncLine string // The function call line
	FileLine string // The file location line (may be empty)
}

// PreprocessedGoRoutineLines represents a preprocessed goroutine stack trace
type PreprocessedGoRoutineLines struct {
	StackFrames []rawStackFrame // Stack frames, where each frame has a function line and optional file line
	CreatedBy   rawStackFrame   // The "created by" information
}

// AnnotateFrame sets the IsImportant and IsSys flags on a stack frame
// based on the module name and package information
func AnnotateFrame(frame *StackFrame, moduleName string) {
	if frame == nil {
		return
	}

	// Mark as important if it belongs to the user's module and is not vendored.
	if moduleName != "" && strings.HasPrefix(frame.Package, moduleName) && !strings.Contains(frame.Package, "/vendor/") {
		frame.IsImportant = true
		return
	}
	if frame.Package == "main" {
		// Special case for main package
		frame.IsImportant = true
		return
	}

	// Determine if the frame is from the standard library or extended runtime.
	// Standard library packages have a first segment without a dot. (dot indicates a domain name, like "github.com/...")
	parts := strings.Split(frame.Package, "/")
	if len(parts) > 0 {
		if !strings.Contains(parts[0], ".") || strings.HasPrefix(frame.Package, "golang.org/x/") {
			frame.IsSys = true
		}
	}
}

// ParseGoRoutineStackTrace parses a Go routine stack trace string into a struct
// moduleName is the name of the module that the app belongs to, used to identify important frames
// goId and state are required parameters since the stacktrace no longer includes the goroutine header line
func ParseGoRoutineStackTrace(stackTrace string, moduleName string, goId int64, state string) (ParsedGoRoutine, error) {
	// Create a basic ParsedGoRoutine with the raw data
	routine := ParsedGoRoutine{
		RawStackTrace: stackTrace,
		Parsed:        false, // Default to not parsed
		GoId:          goId,
		RawState:      state,
	}

	// Parse the state components
	primaryState, stateDurationMs, stateDuration, extraStates := parseStateComponents(state)
	routine.PrimaryState = primaryState
	routine.StateDurationMs = stateDurationMs
	routine.StateDuration = stateDuration
	routine.ExtraStates = extraStates

	// Preprocess the stack trace
	preprocessed := preprocessStackTrace(stackTrace)

	// Parse stack frames
	for _, frame := range preprocessed.StackFrames {
		if parsedFrame, ok := parseFrame(frame.FuncLine, frame.FileLine, true); ok {
			// Annotate the frame with IsImportant and IsSys flags
			AnnotateFrame(&parsedFrame, moduleName)
			routine.ParsedFrames = append(routine.ParsedFrames, parsedFrame)
		}
	}

	// Parse created by information
	if preprocessed.CreatedBy.FuncLine != "" {
		frame, goId, ok := parseCreatedByFrame(preprocessed.CreatedBy.FuncLine, preprocessed.CreatedBy.FileLine)
		if ok {
			// Annotate the created by frame
			AnnotateFrame(frame, moduleName)
			routine.CreatedByGoId = int64(goId)
			routine.CreatedByFrame = frame
		}
	}

	// Mark as successfully parsed
	routine.Parsed = true
	return routine, nil
}

// preprocessStackTrace processes a goroutine stack trace and groups the lines into stack frames and created by sections
func preprocessStackTrace(stackTrace string) PreprocessedGoRoutineLines {
	// Split the stack trace into lines (don't trim yet to preserve indentation)
	lines := strings.Split(stackTrace, "\n")

	result := PreprocessedGoRoutineLines{}
	var currentFuncLine string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is a "created by" line
		if strings.HasPrefix(strings.TrimSpace(line), "created by ") {
			// Add any remaining frame
			if currentFuncLine != "" {
				result.StackFrames = append(result.StackFrames, rawStackFrame{
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
				result.StackFrames = append(result.StackFrames, rawStackFrame{
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
				result.StackFrames = append(result.StackFrames, rawStackFrame{
					FuncLine: currentFuncLine,
					FileLine: "",
				})
			}
			currentFuncLine = trimmedLine
		}
	}

	// Add any remaining frame
	if currentFuncLine != "" {
		result.StackFrames = append(result.StackFrames, rawStackFrame{
			FuncLine: currentFuncLine,
			FileLine: "",
		})
	}

	return result
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
func parseFrame(funcLine, fileLine string, argsRequired bool) (StackFrame, bool) {
	frame := StackFrame{}
	var ok bool
	frame.Package, frame.FuncName, frame.FuncArgs, ok = parseFuncLine(funcLine, argsRequired)
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
func parseCreatedByFrame(funcLine string, fileLine string) (*StackFrame, int, bool) {
	// the trick is just removing "created by" off the front and "in goroutine X" off the end
	if !strings.HasPrefix(funcLine, "created by ") {
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
		return nil, 0, false
	}
	// now parse the frame
	frame, ok := parseFrame(funcLine, fileLine, false)
	if !ok {
		return nil, 0, false
	}
	return &frame, goId, true
}

// parseStateComponents parses a raw state string into its components
func parseStateComponents(rawState string) (string, int64, string, []string) {
	// Split the state by commas
	components := strings.Split(rawState, ",")

	// The first component is always the primary state
	primaryState := strings.TrimSpace(components[0])

	// Initialize variables for additional components
	var stateDurationMs int64
	var stateDuration string
	var extraStates []string

	// Process additional components
	if len(components) > 1 {
		extraStates = make([]string, 0, len(components)-1)

		for _, component := range components[1:] {
			component = strings.TrimSpace(component)

			// Check if this component is a duration
			if isDuration, durationMs := parseDuration(component); isDuration {
				stateDurationMs = durationMs
				stateDuration = component
			} else {
				extraStates = append(extraStates, component)
			}
		}
	}

	return primaryState, stateDurationMs, stateDuration, extraStates
}

// parseFuncLine parses a function line from a stack trace and extracts the package, function name, and args
// The function name includes the receiver if present
func parseFuncLine(funcLine string, argsRequired bool) (pkgName string, funcName string, funcArgs string, valid bool) {
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
	} else if argsRequired {
		// If we require args but there are none, return false
		return "", "", "", false
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
