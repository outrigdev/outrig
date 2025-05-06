// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilfn

import (
	"reflect"
	"testing"
)

func TestParseNameAndTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantTags []string
	}{
		{
			name:     "tag at beginning",
			input:    "#hello mike",
			wantName: "mike",
			wantTags: []string{"hello"},
		},
		{
			name:     "tag in middle",
			input:    "mike #hello bar",
			wantName: "mike bar",
			wantTags: []string{"hello"},
		},
		{
			name:     "tag at end",
			input:    "mike #hello",
			wantName: "mike",
			wantTags: []string{"hello"},
		},
		{
			name:     "multiple tags",
			input:    "#hello mike #world",
			wantName: "mike",
			wantTags: []string{"hello", "world"},
		},
		{
			name:     "complex tag",
			input:    "mike #hello-world/123_test.abc",
			wantName: "mike",
			wantTags: []string{"hello-world/123_test.abc"},
		},
		{
			name:     "no tags",
			input:    "mike",
			wantName: "mike",
			wantTags: []string{},
		},
		{
			name:     "only tag",
			input:    "#hello",
			wantName: "",
			wantTags: []string{"hello"},
		},
		{
			name:     "multiple tags with extra spaces",
			input:    "  #hello   mike   #world  ",
			wantName: "mike",
			wantTags: []string{"hello", "world"},
		},
		{
			name:     "non-terminated tag",
			input:    "mike #hello-world:123_test.abc #foo(hello)",
			wantName: "mike #hello-world:123_test.abc #foo(hello)",
			wantTags: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotTags := ParseNameAndTags(tt.input)
			if gotName != tt.wantName {
				t.Errorf("ParseNameAndTags() name = %q, want %q", gotName, tt.wantName)
			}
			if len(gotTags) != 0 || len(tt.wantTags) != 0 {
				if !reflect.DeepEqual(gotTags, tt.wantTags) {
					t.Errorf("ParseNameAndTags() tags = %+v, want %+v", gotTags, tt.wantTags)
				}
			}
			justTags := ParseTags(tt.input)
			if len(justTags) != 0 || len(tt.wantTags) != 0 {
				if !reflect.DeepEqual(justTags, tt.wantTags) {
					t.Errorf("ParseTags() tags = %+v, want %+v", justTags, tt.wantTags)
				}
			}
		})
	}
}

func TestSemVer(t *testing.T) {
	cmp, err := CompareSemVerCore("v0.4.0-dev", "0.4.0")
	if err != nil {
		t.Errorf("Error comparing versions: %v", err)
		return
	}
	if cmp != 0 {
		t.Errorf("Expected 0, got %d", cmp)
		return
	}

	cmp, err = CompareSemVerCore("v0.4.0+76xcsd", "0.4.1")
	if err != nil {
		t.Errorf("Error comparing versions: %v", err)
		return
	}
	if cmp != -1 {
		t.Errorf("Expected -1, got %d", cmp)
		return
	}

	_, err = CompareSemVerCore("", "v0.4.1")
	if err == nil {
		t.Errorf("Expected error, got nil")
		return
	}
}
