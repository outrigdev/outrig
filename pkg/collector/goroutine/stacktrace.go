package goroutine

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// ParsedGoRoutine represents a parsed goroutine stack trace
type ParsedGoRoutine struct {
	GoId         int64
	RawState     string   // The complete state information
	PrimaryState string   // The first part of the state (before any commas)
	DurationMs   int64    // Duration in milliseconds (if available)
	ExtraStates  []string // Array of additional state information
	Frames       []string
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
					currentFrame = nil
				}

				result = append(result, *currentRoutine)
			}

			// Parse the new goroutine header
			goId, _ := strconv.ParseInt(match[1], 10, 64)
			state := match[2]

			// Create a new routine
			currentRoutine = &ParsedGoRoutine{
				GoId:     goId,
				RawState: state,
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
			currentFrame = nil
		}
	}

	// Add the last routine if there is one
	if currentRoutine != nil {
		// Handle any remaining frame
		if len(currentFrame) > 0 {
			currentRoutine.Frames = append(currentRoutine.Frames, currentFrame...)
		}

		result = append(result, *currentRoutine)
	}

	return result, nil
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
