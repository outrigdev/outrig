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
			input:    "mike #hello-world:123_test.abc",
			wantName: "mike",
			wantTags: []string{"hello-world:123_test.abc"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotTags := ParseNameAndTags(tt.input)
			if gotName != tt.wantName {
				t.Errorf("ParseNameAndTags() name = %v, want %v", gotName, tt.wantName)
			}
			if !reflect.DeepEqual(gotTags, tt.wantTags) {
				t.Errorf("ParseNameAndTags() tags = %v, want %v", gotTags, tt.wantTags)
			}
		})
	}
}
