// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loglineparser

import (
	"regexp"
	"strconv"
	"strings"
)

// LogSpan represents a segment of parsed log text with styling information
type LogSpan struct {
	Text      string   `json:"text"`
	ClassName []string `json:"classname,omitempty"`
}

// ANSI code to Tailwind CSS class mapping
var ansiTailwindMap = map[int]string{
	// Reset and modifiers
	0: "reset", // special: clear state
	1: "font-bold",
	2: "opacity-75",
	3: "italic",
	4: "underline",
	8: "invisible",
	9: "line-through",

	// Foreground standard colors
	30: "text-ansi-black",
	31: "text-ansi-red",
	32: "text-ansi-green",
	33: "text-ansi-yellow",
	34: "text-ansi-blue",
	35: "text-ansi-magenta",
	36: "text-ansi-cyan",
	37: "text-ansi-white",

	// Foreground bright colors
	90: "text-ansi-brightblack",
	91: "text-ansi-brightred",
	92: "text-ansi-brightgreen",
	93: "text-ansi-brightyellow",
	94: "text-ansi-brightblue",
	95: "text-ansi-brightmagenta",
	96: "text-ansi-brightcyan",
	97: "text-ansi-brightwhite",

	// Background standard colors
	40: "bg-ansi-black",
	41: "bg-ansi-red",
	42: "bg-ansi-green",
	43: "bg-ansi-yellow",
	44: "bg-ansi-blue",
	45: "bg-ansi-magenta",
	46: "bg-ansi-cyan",
	47: "bg-ansi-white",

	// Background bright colors
	100: "bg-ansi-brightblack",
	101: "bg-ansi-brightred",
	102: "bg-ansi-brightgreen",
	103: "bg-ansi-brightyellow",
	104: "bg-ansi-brightblue",
	105: "bg-ansi-brightmagenta",
	106: "bg-ansi-brightcyan",
	107: "bg-ansi-brightwhite",
}

// internalState tracks the current ANSI formatting state
type internalState struct {
	modifiers map[string]bool
	textColor string
	bgColor   string
	reverse   bool
}

// makeInitialState creates a new initial state
func makeInitialState() *internalState {
	return &internalState{
		modifiers: make(map[string]bool),
		textColor: "",
		bgColor:   "",
		reverse:   false,
	}
}

// updateStateWithCodes updates the state based on ANSI codes
func updateStateWithCodes(state *internalState, codes []int) {
	for _, code := range codes {
		if code == 0 {
			// Reset all
			state.modifiers = make(map[string]bool)
			state.textColor = ""
			state.bgColor = ""
			state.reverse = false
			continue
		}
		if code == 7 {
			state.reverse = true
			continue
		}
		
		tailwindClass, exists := ansiTailwindMap[code]
		if exists && tailwindClass != "reset" {
			if strings.HasPrefix(tailwindClass, "text-") {
				state.textColor = tailwindClass
			} else if strings.HasPrefix(tailwindClass, "bg-") {
				state.bgColor = tailwindClass
			} else {
				state.modifiers[tailwindClass] = true
			}
		}
	}
}

// stateToClasses converts the current state to a slice of class strings
func stateToClasses(state *internalState) []string {
	var classes []string
	
	// Add modifiers
	for modifier := range state.modifiers {
		classes = append(classes, modifier)
	}
	
	textColor := state.textColor
	bgColor := state.bgColor
	
	// Handle reverse video
	if state.reverse {
		// Convert colors to their opposite types when reversing
		if textColor != "" && strings.HasPrefix(textColor, "text-") {
			textColor = "bg-" + textColor[5:] // Convert text-ansi-red to bg-ansi-red
		}
		if bgColor != "" && strings.HasPrefix(bgColor, "bg-") {
			bgColor = "text-" + bgColor[3:] // Convert bg-ansi-green to text-ansi-green
		}
		textColor, bgColor = bgColor, textColor
	}
	
	if textColor != "" {
		classes = append(classes, textColor)
	}
	if bgColor != "" {
		classes = append(classes, bgColor)
	}
	
	return classes
}

// ANSI escape sequence regex
var ansiRegex = regexp.MustCompile(`\x1b\[([0-9;]+)m`)

// ParseLine parses a line with ANSI escape sequences and returns LogSpan segments
func ParseLine(line string) []LogSpan {
	// Fast path: if no ANSI escapes are found, return single span
	if !strings.Contains(line, "\x1b[") {
		return []LogSpan{{Text: line, ClassName: nil}}
	}
	
	var segments []LogSpan
	lastIndex := 0
	currentState := makeInitialState()
	
	matches := ansiRegex.FindAllStringSubmatchIndex(line, -1)
	
	for _, match := range matches {
		matchStart := match[0]
		matchEnd := match[1]
		codeStart := match[2]
		codeEnd := match[3]
		
		// Add text before this ANSI sequence
		if matchStart > lastIndex {
			text := line[lastIndex:matchStart]
			segments = append(segments, LogSpan{
				Text:      text,
				ClassName: stateToClasses(currentState),
			})
		}
		
		// Parse the ANSI codes
		codeStr := line[codeStart:codeEnd]
		codeStrs := strings.Split(codeStr, ";")
		var codes []int
		for _, cs := range codeStrs {
			if code, err := strconv.Atoi(cs); err == nil {
				codes = append(codes, code)
			}
		}
		
		updateStateWithCodes(currentState, codes)
		lastIndex = matchEnd
	}
	
	// Add remaining text after last ANSI sequence
	if lastIndex < len(line) {
		text := line[lastIndex:]
		segments = append(segments, LogSpan{
			Text:      text,
			ClassName: stateToClasses(currentState),
		})
	}
	
	return segments
}