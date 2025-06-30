// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loglineparser

import (
	"encoding/json"
)

// FindFirstJSON finds the first valid JSON object or array in the given string
// Returns nil if no valid JSON is found, otherwise returns the position range
func FindFirstJSON(s string, allowArrays bool) *Position {
	if len(s) == 0 {
		return nil
	}
	
	for i := 0; i < len(s); i++ {
		char := s[i]
		if char == '{' || (allowArrays && char == '[') {
			// Found potential JSON start, try to parse it
			if pos := IsJsonAt(s, i); pos != nil {
				return pos
			}
		}
	}
	
	return nil
}

// IsJsonAt checks if there is a valid JSON object or array starting at the given index
// Returns nil if no valid JSON is found, otherwise returns the position range
func IsJsonAt(s string, index int) *Position {
	if index >= len(s) {
		return nil
	}
	
	char := s[index]
	if char != '{' && char != '[' {
		return nil
	}
	
	if end := findJSONEnd(s, index); end != -1 {
		// Validate that it's actually valid JSON
		candidate := s[index:end]
		if isValidJSON(candidate) {
			return &Position{Start: index, End: end}
		}
	}
	
	return nil
}

// findJSONEnd finds the matching closing brace/bracket for a JSON object/array
// starting at the given position. Returns -1 if no valid structure is found.
func findJSONEnd(s string, start int) int {
	if start >= len(s) {
		return -1
	}

	openChar := s[start]
	if openChar != '{' && openChar != '[' {
		return -1
	}

	braceDepth := 0
	bracketDepth := 0

	if openChar == '{' {
		braceDepth = 1
	} else {
		bracketDepth = 1
	}

	inString := false
	escaped := false

	for i := start + 1; i < len(s); i++ {
		char := s[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' && inString {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch char {
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		}

		if braceDepth == 0 && bracketDepth == 0 {
			return i + 1 // Return position after the closing character
		}

		if braceDepth < 0 || bracketDepth < 0 {
			return -1 // Mismatched brackets/braces
		}
	}

	return -1 // No matching close found
}

// isValidJSON checks if the given string is valid JSON
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
