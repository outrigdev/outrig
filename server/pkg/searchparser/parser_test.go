// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package searchparser

import (
	"reflect"
	"testing"
)

func TestTokenizeSearch(t *testing.T) {
	tests := []struct {
		name        string
		searchType  string
		searchInput string
		want        []SearchToken
	}{
		{
			name:        "Empty string",
			searchType:  "exact",
			searchInput: "",
			want:        []SearchToken{},
		},
		{
			name:        "Single token",
			searchType:  "exact",
			searchInput: "hello",
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
			},
		},
		{
			name:        "Multiple tokens",
			searchType:  "exact",
			searchInput: "hello world",
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Double quoted token",
			searchType:  "exact",
			searchInput: `"hello world"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello world"},
			},
		},
		{
			name:        "Single quoted token",
			searchType:  "exact",
			searchInput: `'hello world'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "hello world"},
			},
		},
		{
			name:        "Mixed tokens",
			searchType:  "exact",
			searchInput: `hello "world of" code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world of"},
				{Type: "exact", SearchTerm: "code"},
			},
		},
		{
			name:        "Mixed single and double quotes",
			searchType:  "exact",
			searchInput: `hello 'World Of' code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exactcase", SearchTerm: "World Of"},
				{Type: "exact", SearchTerm: "code"},
			},
		},
		{
			name:        "Unclosed double quote",
			searchType:  "exact",
			searchInput: `hello "world of code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world of code"},
			},
		},
		{
			name:        "Unclosed single quote",
			searchType:  "exact",
			searchInput: `hello 'World Of code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exactcase", SearchTerm: "World Of code"},
			},
		},
		{
			name:        "Multiple quoted tokens",
			searchType:  "exact",
			searchInput: `"hello world" "of code"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello world"},
				{Type: "exact", SearchTerm: "of code"},
			},
		},
		{
			name:        "Multiple single quoted tokens",
			searchType:  "exact",
			searchInput: `'Hello World' 'Of Code'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "Hello World"},
				{Type: "exactcase", SearchTerm: "Of Code"},
			},
		},
		{
			name:        "Mixed single and double quoted tokens",
			searchType:  "exact",
			searchInput: `'Hello World' "of code"`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "Hello World"},
				{Type: "exact", SearchTerm: "of code"},
			},
		},
		{
			name:        "Empty quoted token",
			searchType:  "exact",
			searchInput: `hello "" world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: ""},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Empty single quoted token",
			searchType:  "exact",
			searchInput: `hello '' world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exactcase", SearchTerm: ""},
				{Type: "exact", SearchTerm: "world"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TokenizeSearch(tt.searchType, tt.searchInput)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TokenizeSearch() = %v, want %v", got, tt.want)
			}
		})
	}
}
