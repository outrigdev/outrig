package goroutine

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// ParsedGoRoutine represents a parsed goroutine stack trace
type ParsedGoRoutine struct {
	GoId      int64
	State     string
	Frames    []string
	CreatedBy string
}

// ParseGoRoutineStackTrace parses a Go routine stack trace string into a struct
func ParseGoRoutineStackTrace(stackTrace string) ([]ParsedGoRoutine, error) {
	var result []ParsedGoRoutine

	// Split the input by goroutine headers
	headerRegex := regexp.MustCompile(`(?m)^goroutine\s+(\d+)\s+\[(.*?)(,\s+(\d+).*?)?\]:$`)

	// Find all goroutine blocks
	blocks := headerRegex.Split(stackTrace, -1)
	headerMatches := headerRegex.FindAllStringSubmatch(stackTrace, -1)

	if len(blocks) <= 1 || len(headerMatches) == 0 {
		return result, nil
	}

	// Start from index 1 as the first split is before any header
	for i := 0; i < len(headerMatches); i++ {
		// Skip the first block which is empty (before the first header)
		blockContent := blocks[i+1]

		headerMatch := headerMatches[i]
		goId, _ := strconv.ParseInt(headerMatch[1], 10, 64)
		state := headerMatch[2]

		// Parse the stack frames and created by info
		routine := ParsedGoRoutine{
			GoId:  goId,
			State: state,
		}

		// Process the block content
		scanner := bufio.NewScanner(strings.NewReader(blockContent))
		var currentFrame []string

		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimSpace(line)

			if line == "" {
				continue
			}

			// Check if this is a "created by" line
			if strings.HasPrefix(line, "created by ") {
				routine.CreatedBy = line
				continue
			}

			// Add to current frame
			currentFrame = append(currentFrame, line)

			// If we have 2 lines in the current frame, add it to frames and reset
			if len(currentFrame) == 2 {
				routine.Frames = append(routine.Frames, currentFrame...)
				currentFrame = nil
			}
		}

		// Handle any remaining frame
		if len(currentFrame) > 0 {
			routine.Frames = append(routine.Frames, currentFrame...)
		}

		result = append(result, routine)
	}

	return result, nil
}
