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
		searchInput string
		want        []SearchToken
	}{
		{
			name:        "Empty string",
			searchInput: "",
			want:        []SearchToken{},
		},
		{
			name:        "Single token",
			searchInput: "hello",
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
			},
		},
		{
			name:        "Multiple tokens",
			searchInput: "hello world",
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Double quoted token",
			searchInput: `"hello world"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello world"},
			},
		},
		{
			name:        "Single quoted token",
			searchInput: `'hello world'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "hello world"},
			},
		},
		{
			name:        "Mixed tokens",
			searchInput: `hello "world of" code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world of"},
				{Type: "exact", SearchTerm: "code"},
			},
		},
		{
			name:        "Mixed single and double quotes",
			searchInput: `hello 'World Of' code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exactcase", SearchTerm: "World Of"},
				{Type: "exact", SearchTerm: "code"},
			},
		},
		{
			name:        "Unclosed double quote",
			searchInput: `hello "world of code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world of code"},
			},
		},
		{
			name:        "Unclosed single quote",
			searchInput: `hello 'World Of code`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exactcase", SearchTerm: "World Of code"},
			},
		},
		{
			name:        "Multiple quoted tokens",
			searchInput: `"hello world" "of code"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello world"},
				{Type: "exact", SearchTerm: "of code"},
			},
		},
		{
			name:        "Multiple single quoted tokens",
			searchInput: `'Hello World' 'Of Code'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "Hello World"},
				{Type: "exactcase", SearchTerm: "Of Code"},
			},
		},
		{
			name:        "Mixed single and double quoted tokens",
			searchInput: `'Hello World' "of code"`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "Hello World"},
				{Type: "exact", SearchTerm: "of code"},
			},
		},
		{
			name:        "Empty quoted token",
			searchInput: `hello "" world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Empty single quoted token",
			searchInput: `hello '' world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Fuzzy search token",
			searchInput: `hello ~world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "fzf", SearchTerm: "world"},
			},
		},
		{
			name:        "Fuzzy search with double quotes",
			searchInput: `~"hello world"`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "hello world"},
			},
		},
		{
			name:        "Fuzzy search with single quotes",
			searchInput: `~'Hello World'`,
			want: []SearchToken{
				{Type: "fzfcase", SearchTerm: "Hello World"},
			},
		},
		{
			name:        "Double tilde",
			searchInput: `hello ~~world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "fzf", SearchTerm: "~world"},
			},
		},
		{
			name:        "Empty tilde",
			searchInput: `hello ~ world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Mixed fuzzy and regular tokens",
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
			searchInput: `/test\d+/`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: `test\d+`},
			},
		},
		{
			name:        "Regexp token with escaped slashes",
			searchInput: `/path\/to\/file/`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: `path\/to\/file`},
			},
		},
		{
			name:        "Unclosed regexp token",
			searchInput: `/unclosed`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: `unclosed`},
			},
		},
		{
			name:        "Empty regexp token",
			searchInput: `//`,
			want:        []SearchToken{},
		},
		{
			name:        "Mixed regexp and other tokens",
			searchInput: `hello /world\d+/ "quoted text"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "regexp", SearchTerm: `world\d+`},
				{Type: "exact", SearchTerm: "quoted text"},
			},
		},
		{
			name:        "Case-sensitive regexp token",
			searchInput: `c/CaseSensitive/`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `CaseSensitive`},
			},
		},
		{
			name:        "Mixed case-sensitive and case-insensitive regexp tokens",
			searchInput: `c/CaseSensitive/ /caseinsensitive/`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `CaseSensitive`},
				{Type: "regexp", SearchTerm: `caseinsensitive`},
			},
		},
		{
			name:        "Case-sensitive regexp token with escaped slashes",
			searchInput: `c/Path\/To\/File/`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `Path\/To\/File`},
			},
		},
		{
			name:        "Unclosed case-sensitive regexp token",
			searchInput: `c/Unclosed`,
			want: []SearchToken{
				{Type: "regexpcase", SearchTerm: `Unclosed`},
			},
		},
		{
			name:        "Tag token",
			searchInput: `#foo`,
			want: []SearchToken{
				{Type: "tag", SearchTerm: `foo`},
			},
		},
		{
			name:        "Tag token with exact match",
			searchInput: `#foo/`,
			want: []SearchToken{
				{Type: "tag", SearchTerm: `foo/`},
			},
		},
		{
			name:        "Hash marked token",
			searchInput: `#marked`,
			want: []SearchToken{
				{Type: "marked", SearchTerm: ``},
			},
		},
		{
			name:        "Hash marked token case-insensitive",
			searchInput: `#MaRkEd`,
			want: []SearchToken{
				{Type: "marked", SearchTerm: ``},
			},
		},
		{
			name:        "Multiple tag tokens",
			searchInput: `#foo #bar`,
			want: []SearchToken{
				{Type: "tag", SearchTerm: `foo`},
				{Type: "tag", SearchTerm: `bar`},
			},
		},
		{
			name:        "Mixed tag and other tokens",
			searchInput: `hello #foo "quoted text" #marked`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "tag", SearchTerm: "foo"},
				{Type: "exact", SearchTerm: "quoted text"},
				{Type: "marked", SearchTerm: ""},
			},
		},
		{
			name:        "Empty hash token",
			searchInput: `hello # world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello"},
				{Type: "exact", SearchTerm: "#"},
				{Type: "exact", SearchTerm: "world"},
			},
		},
		{
			name:        "Single hash character",
			searchInput: `#`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "#"},
			},
		},
		{
			name:        "Single double quote character",
			searchInput: `"`,
			want:        []SearchToken{},
		},
		{
			name:        "Single single quote character",
			searchInput: `'`,
			want:        []SearchToken{},
		},
		{
			name:        "Simple not token",
			searchInput: `-hello`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello", IsNot: true},
			},
		},
		{
			name:        "Not token with fuzzy search",
			searchInput: `-~hello`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "hello", IsNot: true},
			},
		},
		{
			name:        "Not token with regexp",
			searchInput: `-/hello/`,
			want: []SearchToken{
				{Type: "regexp", SearchTerm: "hello", IsNot: true},
			},
		},
		{
			name:        "Not token with quoted string",
			searchInput: `-"hello world"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello world", IsNot: true},
			},
		},
		{
			name:        "Not token with single quoted string",
			searchInput: `-'Hello World'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "Hello World", IsNot: true},
			},
		},
		{
			name:        "Multiple not tokens",
			searchInput: `-hello -world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello", IsNot: true},
				{Type: "exact", SearchTerm: "world", IsNot: true},
			},
		},
		{
			name:        "Mixed not and regular tokens",
			searchInput: `hello -world`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "hello", IsNot: false},
				{Type: "exact", SearchTerm: "world", IsNot: true},
			},
		},
		{
			name:        "Literal dash in quoted string",
			searchInput: `"-hello"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "-hello", IsNot: false},
			},
		},
		{
			name:        "Dash as a standalone token",
			searchInput: `-`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "-", IsNot: false},
			},
		},
		{
			name:        "Simple OR expression",
			searchInput: `mike | mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "OR expression with multiple tokens on left side",
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
			searchInput: `"mike smith" | "mark johnson"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike smith"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark johnson"},
			},
		},
		{
			name:        "OR expression with fuzzy search",
			searchInput: `~mike | ~mark`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "fzf", SearchTerm: "mark"},
			},
		},
		{
			name:        "OR expression with NOT tokens",
			searchInput: `-mike | -mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike", IsNot: true},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark", IsNot: true},
			},
		},
		{
			name:        "OR expression with mixed token types",
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
			searchInput: `| mike`,
			want: []SearchToken{
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mike"},
			},
		},
		{
			name:        "Pipe character at the end",
			searchInput: `mike |`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
			},
		},
		{
			name:        "Only pipe character",
			searchInput: `|`,
			want: []SearchToken{
				{Type: "or", SearchTerm: "|"},
			},
		},
		{
			name:        "Pipe without whitespace before",
			searchInput: `mike|mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "Pipe without whitespace after",
			searchInput: `mike |mark`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "exact", SearchTerm: "mark"},
			},
		},
		{
			name:        "Multiple pipes without whitespace",
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
			searchInput: `"mike|mark"`,
			want: []SearchToken{
				{Type: "exact", SearchTerm: "mike|mark"},
			},
		},
		{
			name:        "Pipe in single quoted string",
			searchInput: `'mike|mark'`,
			want: []SearchToken{
				{Type: "exactcase", SearchTerm: "mike|mark"},
			},
		},
		{
			name:        "Mixed tokens with pipes without whitespace",
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
			searchInput: `~mike|~mark`,
			want: []SearchToken{
				{Type: "fzf", SearchTerm: "mike"},
				{Type: "or", SearchTerm: "|"},
				{Type: "fzf", SearchTerm: "mark"},
			},
		},
		{
			name:        "NOT tokens with pipe without whitespace",
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
			got := TokenizeSearch(tt.searchInput)
			// For empty slices, just check the length
			if len(got) == 0 && len(tt.want) == 0 {
				// Both are empty, test passes
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TokenizeSearch() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
