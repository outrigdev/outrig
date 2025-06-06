// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package loglineparser

import (
	"testing"
)

func TestParseLineNoAnsi(t *testing.T) {
	line := "Hello, world!"
	spans := ParseLine(line)

	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Text != "Hello, world!" {
		t.Errorf("Expected text 'Hello, world!', got '%s'", spans[0].Text)
	}

	if len(spans[0].ClassName) != 0 {
		t.Errorf("Expected empty class name, got %v", spans[0].ClassName)
	}
}

func TestParseLineWithAnsi(t *testing.T) {
	line := "\x1b[31mRed text\x1b[0m normal text"
	spans := ParseLine(line)

	if len(spans) != 2 {
		t.Errorf("Expected 2 spans, got %d", len(spans))
	}

	if spans[0].Text != "Red text" {
		t.Errorf("Expected text 'Red text', got '%s'", spans[0].Text)
	}

	if !stringSliceEqual(spans[0].ClassName, []string{"text-ansi-red"}) {
		t.Errorf("Expected class ['text-ansi-red'], got %v", spans[0].ClassName)
	}
	
	if spans[1].Text != " normal text" {
		t.Errorf("Expected text ' normal text', got '%s'", spans[1].Text)
	}
	
	if len(spans[1].ClassName) != 0 {
		t.Errorf("Expected empty class name, got %v", spans[1].ClassName)
	}
}

func TestParseLineWithBoldAndColor(t *testing.T) {
	line := "\x1b[1;31mBold red text\x1b[0m"
	spans := ParseLine(line)

	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}

	if spans[0].Text != "Bold red text" {
		t.Errorf("Expected text 'Bold red text', got '%s'", spans[0].Text)
	}

	// Should contain both font-bold and text-ansi-red
	if !stringSliceContainsAll(spans[0].ClassName, "font-bold", "text-ansi-red") {
		t.Errorf("Expected class to contain 'font-bold' and 'text-ansi-red', got %v", spans[0].ClassName)
	}
}

func TestParseLineWithBackground(t *testing.T) {
	line := "\x1b[42mGreen background\x1b[0m"
	spans := ParseLine(line)

	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}

	if !stringSliceEqual(spans[0].ClassName, []string{"bg-ansi-green"}) {
		t.Errorf("Expected class ['bg-ansi-green'], got %v", spans[0].ClassName)
	}
}

func TestParseLineWithReverse(t *testing.T) {
	line := "\x1b[31;42;7mReverse text\x1b[0m"
	spans := ParseLine(line)

	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}

	// With reverse, text and background colors should be swapped
	// Original: red text (31) + green background (42)
	// Reversed: green text + red background
	if !stringSliceContainsAll(spans[0].ClassName, "text-ansi-green", "bg-ansi-red") {
		t.Errorf("Expected reversed colors (text-ansi-green bg-ansi-red), got %v", spans[0].ClassName)
	}
}

// Helper functions for testing string slices

// stringSliceEqual checks if two string slices contain the same elements (order doesn't matter)
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	
	// Create maps to count occurrences
	countA := make(map[string]int)
	countB := make(map[string]int)
	
	for _, s := range a {
		countA[s]++
	}
	for _, s := range b {
		countB[s]++
	}
	
	// Compare maps
	for k, v := range countA {
		if countB[k] != v {
			return false
		}
	}
	
	return true
}

// stringSliceContains checks if a string slice contains a specific string
func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// stringSliceContainsAll checks if a string slice contains all specified strings
func stringSliceContainsAll(slice []string, items ...string) bool {
	for _, item := range items {
		if !stringSliceContains(slice, item) {
			return false
		}
	}
	return true
}
