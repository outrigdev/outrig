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
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Empty single quoted token",
			searchType:  "exact",
			searchInput: `hello '' world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Fuzzy search token",
			searchType:  "exact",
			searchInput: `hello ~world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "fzf", SearchTerm: "world"},
			},
		},
		{
			name:        "Fuzzy search with double quotes",
			searchType:  "exact",
			searchInput: `~"hello world"`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "hello world"},
			},
		},
		{
			name:        "Fuzzy search with single quotes",
			searchType:  "exact",
			searchInput: `~'Hello World'`,
			want: []SearchToken{
				{Type: "fzfcase", SearchTerm: "Hello World"},
			},
		},
		{
			name:        "Double tilde",
			searchType:  "exact",
			searchInput: `hello ~~world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "fzf", SearchTerm: "~world"},
			},
		},
		{
			name:        "Empty tilde",
			searchType:  "exact",
			searchInput: `hello ~ world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Mixed fuzzy and regular tokens",
			searchType:  "exact",
			searchInput: `hello ~world "test" ~"fuzzy search"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "fzf", SearchTerm: "world"},
				{Type: "exact", SearchTerm: "test"},
				{Type: "fzf", SearchTerm: "fuzzy search"},
			},
		},
		{
			name:        "Simple regexp token",
			searchType:  "exact",
			searchInput: `/test\d+/`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: `test\d+`},
			},
		},
		{
			name:        "Regexp token with escaped slashes",
			searchType:  "exact",
			searchInput: `/path\/to\/file/`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: `path\/to\/file`},
			},
		},
		{
			name:        "Unclosed regexp token",
			searchType:  "exact",
			searchInput: `/unclosed`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: `unclosed`},
			},
		},
		{
			name:        "Empty regexp token",
			searchType:  "exact",
			searchInput: `//`,
			want: []SearchToken{},
		},
		{
			name:        "Mixed regexp and other tokens",
			searchType:  "exact",
			searchInput: `hello /world\d+/ "quoted text"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "regexp", SearchTerm: `world\d+`},
				{Type: "exact", SearchTerm: "quoted text"},
			},
		},
		{
			name:        "Case-sensitive regexp token",
			searchType:  "exact",
			searchInput: `c/CaseSensitive/`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `CaseSensitive`},
			},
		},
		{
			name:        "Mixed case-sensitive and case-insensitive regexp tokens",
			searchType:  "exact",
			searchInput: `c/CaseSensitive/ /caseinsensitive/`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `CaseSensitive`},
				{Type: "regexp", SearchTerm: `caseinsensitive`},
			},
		},
		{
			name:        "Case-sensitive regexp token with escaped slashes",
			searchType:  "exact",
			searchInput: `c/Path\/To\/File/`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `Path\/To\/File`},
			},
		},
		{
			name:        "Unclosed case-sensitive regexp token",
			searchType:  "exact",
			searchInput: `c/Unclosed`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `Unclosed`},
			},
		},
		{
			name:        "Hash token",
			searchType:  "exact",
			searchInput: `#foo`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: `#foo`},
			},
		},
		{
			name:        "Hash marked token",
			searchType:  "exact",
			searchInput: `#marked`,
			want: []SearchToken{
				{Type: "marked", SearchTerm: ``},
			},
		},
		{
			name:        "Hash marked token case-insensitive",
			searchType:  "exact",
			searchInput: `#MaRkEd`,
			want: []SearchToken{
				{Type: "marked", SearchTerm: ``},
			},
		},
		{
			name:        "Multiple hash tokens",
			searchType:  "exact",
			searchInput: `#foo #bar`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: `#foo`},
				{Type: "exact", SearchTerm: `#bar`},
			},
		},
		{
			name:        "Mixed hash and other tokens",
			searchType:  "exact",
			searchInput: `hello #foo "quoted text" #marked`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "#foo"},
				{Type: "exact", SearchTerm: "quoted text"},
				{Type: "marked", SearchTerm: ""},
			},
		},
		{
			name:        "Empty hash token",
			searchType:  "exact",
			searchInput: `hello # world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "#"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Single hash character",
			searchType:  "exact",
			searchInput: `#`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "#"},
			},
		},
		{
			name:        "Single double quote character",
			searchType:  "exact",
			searchInput: `"`,
			want: []SearchToken{},
		},
		{
			name:        "Single single quote character",
			searchType:  "exact",
			searchInput: `'`,
			want: []SearchToken{},
		},
		{
			name:        "Simple not token",
			searchType:  "exact",
			searchInput: `-hello`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello", IsNot: true},
			},
		},
		{
			name:        "Not token with fuzzy search",
			searchType:  "exact",
			searchInput: `-~hello`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "hello", IsNot: true},
			},
		},
		{
			name:        "Not token with regexp",
			searchType:  "exact",
			searchInput: `-/hello/`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: "hello", IsNot: true},
			},
		},
		{
			name:        "Not token with quoted string",
			searchType:  "exact",
			searchInput: `-"hello world"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello world", IsNot: true},
			},
		},
		{
			name:        "Not token with single quoted string",
			searchType:  "exact",
			searchInput: `-'Hello World'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "Hello World", IsNot: true},
			},
		},
		{
			name:        "Multiple not tokens",
			searchType:  "exact",
			searchInput: `-hello -world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello", IsNot: true},
				{Type: "exact", SearchTerm: "world", IsNot: true},
			},
		},
		{
			name:        "Mixed not and regular tokens",
			searchType:  "exact",
			searchInput: `hello -world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello", IsNot: false},
				{Type: "exact", SearchTerm: "world", IsNot: true},
			},
		},
		{
			name:        "Literal dash in quoted string",
			searchType:  "exact",
			searchInput: `"-hello"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "-hello", IsNot: false},
			},
		},
		{
			name:        "Dash as a standalone token",
			searchType:  "exact",
			searchInput: `-`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "-", IsNot: false},
			},
		},
		{
			name:        "Simple OR expression",
			searchType:  "exact",
			searchInput: `mike | mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "OR expression with multiple tokens on left side",
			searchType:  "exact",
			searchInput: `mike michelle | mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "exact", SearchTerm: "michelle"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "OR expression with multiple tokens on right side",
			searchType:  "exact",
			searchInput: `mike | mark mary`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
				{Type: "exact", SearchTerm: "mary"},
			},
		},
		{
			name:        "Multiple OR expressions",
			searchType:  "exact",
			searchInput: `mike | mark | mary`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mary"},
			},
		},
		{
			name:        "OR expression with quoted strings",
			searchType:  "exact",
			searchInput: `"mike smith" | "mark johnson"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike smith"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark johnson"},
			},
		},
		{
			name:        "OR expression with fuzzy search",
			searchType:  "exact",
			searchInput: `~mike | ~mark`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "fzf", SearchTerm: "mark"},
			},
		},
		{
			name:        "OR expression with NOT tokens",
			searchType:  "exact",
			searchInput: `-mike | -mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike", IsNot: true},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark", IsNot: true},
			},
		},
		{
			name:        "OR expression with mixed token types",
			searchType:  "exact",
			searchInput: `mike ~johnson | /mark\d+/ | #marked`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "fzf", SearchTerm: "johnson"},
				{Type: "or", SearchTerm: "|"},
				{Type: "regexp", SearchTerm: `mark\d+`},
				{Type: "or", SearchTerm: "|"},
				{Type: "marked", SearchTerm: ""},
			},
		},
		{
			name:        "Empty OR expression segments",
			searchType:  "exact",
			searchInput: `mike | | mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "Pipe character at the beginning",
			searchType:  "exact",
			searchInput: `| mike`,
			want: []SearchToken{
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mike"},
			},
		},
		{
			name:        "Pipe character at the end",
			searchType:  "exact",
			searchInput: `mike |`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
			},
		},
		{
			name:        "Only pipe character",
			searchType:  "exact",
			searchInput: `|`,
			want: []SearchToken{
				{Type: "or", SearchTerm: "|"},
			},
		},
		{
			name:        "Pipe without whitespace before",
			searchType:  "exact",
			searchInput: `mike|mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "Pipe without whitespace after",
			searchType:  "exact",
			searchInput: `mike |mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "Multiple pipes without whitespace",
			searchType:  "exact",
			searchInput: `mike|mark|mary`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mary"},
			},
		},
		{
			name:        "Pipe in quoted string",
			searchType:  "exact",
			searchInput: `"mike|mark"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike|mark"},
			},
		},
		{
			name:        "Pipe in single quoted string",
			searchType:  "exact",
			searchInput: `'mike|mark'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "mike|mark"},
			},
		},
		{
			name:        "Mixed tokens with pipes without whitespace",
			searchType:  "exact",
			searchInput: `mike michelle|mark mary`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "exact", SearchTerm: "michelle"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
				{Type: "exact", SearchTerm: "mary"},
			},
		},
		{
			name:        "Fuzzy search with pipe without whitespace",
			searchType:  "exact",
			searchInput: `~mike|~mark`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "fzf", SearchTerm: "mark"},
			},
		},
		{
			name:        "NOT tokens with pipe without whitespace",
			searchType:  "exact",
			searchInput: `-mike|-mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike", IsNot: true},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark", IsNot: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TokenizeSearch(tt.searchType, tt.searchInput)
			// For empty slices, just check the length
			if len(got) == 0 && len(tt.want) == 0 {
				// Both are empty, test passes
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TokenizeSearch() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
