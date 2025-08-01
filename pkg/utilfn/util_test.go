// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilfn

import (
	"reflect"
	"testing"
)

func TestParseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTags []string
	}{
		{
			name:     "tag at beginning",
			input:    "#hello mike",
			wantTags: []string{"hello"},
		},
		{
			name:     "tag in middle",
			input:    "mike #hello bar",
			wantTags: []string{"hello"},
		},
		{
			name:     "tag at end",
			input:    "mike #hello",
			wantTags: []string{"hello"},
		},
		{
			name:     "multiple tags",
			input:    "#hello mike #world",
			wantTags: []string{"hello", "world"},
		},
		{
			name:     "complex tag",
			input:    "mike #hello-world/123_test.abc",
			wantTags: []string{"hello-world/123_test.abc"},
		},
		{
			name:     "no tags",
			input:    "mike",
			wantTags: []string{},
		},
		{
			name:     "only tag",
			input:    "#hello",
			wantTags: []string{"hello"},
		},
		{
			name:     "multiple tags with extra spaces",
			input:    "  #hello   mike   #world  ",
			wantTags: []string{"hello", "world"},
		},
		{
			name:     "valid and invalid tags",
			input:    "mike #hello-world:123_test.abc #foo(hello)",
			wantTags: []string{"hello-world:123_test.abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTags := ParseTags(tt.input)
			if len(gotTags) != 0 || len(tt.wantTags) != 0 {
				if !reflect.DeepEqual(gotTags, tt.wantTags) {
					t.Errorf("ParseTags() tags = %+v, want %+v", gotTags, tt.wantTags)
				}
			}
		})
	}
}

func TestParseTagsSetGr(t *testing.T) {
	input := `2025/06/17 18:48:24 #record #setgr from:Run demo.StdinMonitor (22) &ds.GoDecl{Name:"demo.StdinMonitor", Tags:[]string{"system"}, Pkg:"github.com/outrigdev/outrig", NewLine:"", RunLine:"", NoRecover:false, GoId:22, ParentGoId:1, NumSpawned:0, State:1, StartTs:1750211304225, EndTs:0, FirstPollTs:0, LastPollTs:0, RealCreatedBy:"created by github.com/outrigdev/outrig/server/demo.RunOutrigAcres in goroutine 1\n\t/Users/mike/work/outrig/server/demo/outrigacres.go:782 +0x123"}`

	tags := ParseTags(input)
	expectedTags := []string{"record", "setgr"}

	if len(tags) != len(expectedTags) {
		t.Errorf("ParseTags() returned %d tags, expected %d. Got: %+v", len(tags), len(expectedTags), tags)
		return
	}

	for i, expected := range expectedTags {
		if i >= len(tags) || tags[i] != expected {
			t.Errorf("ParseTags() tag[%d] = %q, expected %q. Full result: %+v", i, tags[i], expected, tags)
			return
		}
	}
}

func TestParseTagsSimple(t *testing.T) {
	input := "test #record #setgr more"

	tags := ParseTags(input)
	expectedTags := []string{"record", "setgr"}

	if len(tags) != len(expectedTags) {
		t.Errorf("ParseTags() returned %d tags, expected %d. Got: %+v", len(tags), len(expectedTags), tags)
		return
	}

	for i, expected := range expectedTags {
		if i >= len(tags) || tags[i] != expected {
			t.Errorf("ParseTags() tag[%d] = %q, expected %q. Full result: %+v", i, tags[i], expected, tags)
			return
		}
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
