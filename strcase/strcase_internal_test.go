// SPDX-License-Identifier: BSD-3-Clause

package strcase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_toCamel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		sep   rune
	}{
		{
			name:  "empty string",
			input: "",
			sep:   '_',
			want:  "",
		},
		{
			name:  "word and word snake_cased",
			input: "one_two",
			sep:   '_',
			want:  "oneTwo",
		},
		{
			name:  "word and word kebab-cased",
			input: "one-two",
			sep:   '-',
			want:  "oneTwo",
		},
		{
			name:  "abbrev and word",
			input: "json_data",
			sep:   '_',
			want:  "jsonData",
		},
		{
			name:  "word and abbrev and word",
			input: "read_json_data",
			sep:   '_',
			want:  "readJsonData",
		},
		{
			name:  "letter and word",
			input: "a_bee",
			sep:   '_',
			want:  "aBee",
		},
		{
			name:  "letter and letter",
			input: "a_b",
			sep:   '_',
			want:  "aB",
		},
		{
			name:  "word and letter",
			input: "plan_b",
			sep:   '_',
			want:  "planB",
		},
		{
			name:  "letter and digit and word",
			input: "v1_name",
			sep:   '_',
			want:  "v1Name",
		},
		{
			name:  "capital letter and digit and word",
			input: "v1_name",
			sep:   '_',
			want:  "v1Name",
		},
		{
			name:  "the leading separators are kept",
			input: "__one_two",
			sep:   '_',
			want:  "__oneTwo",
		},
		{
			name:  "the trailing separators are kept",
			input: "one_two__",
			sep:   '_',
			want:  "oneTwo__",
		},
		{
			name:  "the separators at both edges are kept",
			input: "__one_two__",
			sep:   '_',
			want:  "__oneTwo__",
		},
		{
			name:  "the leading separators are not a word boundary",
			input: "__one",
			sep:   '_',
			want:  "__one",
		},
		{
			name:  "the inner separators are a word boundary",
			input: "one__two",
			sep:   '_',
			want:  "oneTwo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCamel(tt.input, tt.sep)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_fromCamel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		sep   rune
		want  string
	}{
		{
			name:  "empty input string",
			input: "",
			sep:   '_',
			want:  "",
		},
		{
			name:  "word and word snake_cased",
			input: "oneTwo",
			sep:   '_',
			want:  "one_two",
		},
		{
			name:  "word and word kebab-cased",
			input: "oneTwo",
			sep:   '-',
			want:  "one-two",
		},
		{
			name:  "capitalized",
			input: "OneTwo",
			sep:   '_',
			want:  "one_two",
		},
		{
			name:  "word and abbrev",
			input: "ReadJSON",
			sep:   '_',
			want:  "read_json",
		},
		{
			name:  "abbrev and word",
			input: "JSONData",
			sep:   '_',
			want:  "json_data",
		},
		{
			name:  "word and abbrev and word",
			input: "ReadJSONData",
			sep:   '_',
			want:  "read_json_data",
		},
		{
			name:  "letter and word",
			input: "aBee",
			sep:   '_',
			want:  "a_bee",
		},
		{
			name:  "letter and letter",
			input: "aB",
			sep:   '_',
			want:  "a_b",
		},
		{
			name:  "word and letter",
			input: "PlanB",
			sep:   '_',
			want:  "plan_b",
		},
		{
			name:  "word and letter and word",
			input: "PlanBDone",
			sep:   '_',
			want:  "plan_b_done",
		},
		{
			name:  "word and two letters and word",
			input: "PlanZiDone",
			sep:   '_',
			want:  "plan_zi_done",
		},
		{
			name:  "letter and digit and word",
			input: "v1Name",
			sep:   '_',
			want:  "v1_name",
		},
		{
			name:  "capital letter and digit and word",
			input: "V1Name",
			sep:   '_',
			want:  "v1_name",
		},
		{
			name:  "the leading separators are kept",
			input: "_oneTwo",
			sep:   '_',
			want:  "_one_two",
		},
		{
			name:  "the trailing separators are kept",
			input: "oneTwo__",
			sep:   '_',
			want:  "one_two__",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fromCamel(tt.input, tt.sep)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_replaceSep(t *testing.T) {
	tests := []struct {
		name     string
		inputStr string
		fromSep  rune
		toSep    rune
		want     string
	}{
		{
			name:     "empty input string",
			inputStr: "bonjour-le-monde",
			fromSep:  '-',
			toSep:    noSep,
			want:     "bonjourlemonde",
		},
		{
			name:     "hyphen to underscore",
			inputStr: "bonjour-le-monde",
			fromSep:  '-',
			toSep:    '_',
			want:     "bonjour_le_monde",
		},
		{
			name:     "hyphen to noSep",
			inputStr: "bonjour-le-monde",
			fromSep:  '-',
			toSep:    noSep,
			want:     "bonjourlemonde",
		},
		{
			name:     "noSep to anything",
			inputStr: "bonjour-le-monde",
			fromSep:  noSep,
			toSep:    '#',
			want:     "bonjour-le-monde",
		},
		{
			name:     "the leading separators are not replaced",
			inputStr: "__bonjour_le_monde",
			fromSep:  '_',
			toSep:    '-',
			want:     "__bonjour-le-monde",
		},
		{
			name:     "the trailing separators are not replaced",
			inputStr: "bonjour_le_monde__",
			fromSep:  '_',
			toSep:    '-',
			want:     "bonjour-le-monde__",
		},
		{
			name:     "the leading separators are not removed",
			inputStr: "__bonjour_le_monde",
			fromSep:  '_',
			toSep:    noSep,
			want:     "__bonjourlemonde",
		},
		{
			name:     "the trailing separators are not removed",
			inputStr: "bonjour_le_monde__",
			fromSep:  '_',
			toSep:    noSep,
			want:     "bonjourlemonde__",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceSep(tt.inputStr, tt.fromSep, tt.toSep)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_splitEdges(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPrefix string
		wantCore   string
		wantSuffix string
	}{
		{
			name:       "empty string",
			input:      "",
			wantPrefix: "",
			wantCore:   "",
			wantSuffix: "",
		},
		{
			name:       "no separators",
			input:      "hello",
			wantPrefix: "",
			wantCore:   "hello",
			wantSuffix: "",
		},
		{
			name:       "the leading separators",
			input:      "__hello",
			wantPrefix: "__",
			wantCore:   "hello",
			wantSuffix: "",
		},
		{
			name:       "the trailing separators",
			input:      "hello__",
			wantPrefix: "",
			wantCore:   "hello",
			wantSuffix: "__",
		},
		{
			name:       "the separators at both edges",
			input:      "__hello__",
			wantPrefix: "__",
			wantCore:   "hello",
			wantSuffix: "__",
		},
		{
			name:       "the inner separators",
			input:      "hello_world",
			wantPrefix: "",
			wantCore:   "hello_world",
			wantSuffix: "",
		},
		{
			name:       "the separators only",
			input:      "__",
			wantPrefix: "__",
			wantCore:   "",
			wantSuffix: "",
		},
		{
			name:       "the mixed separators",
			input:      "__hello--world--",
			wantPrefix: "__",
			wantCore:   "hello--world",
			wantSuffix: "--",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrefix, gotCore, gotSuffix := splitEdges(tt.input)
			assert.Equal(t, tt.wantPrefix, gotPrefix)
			assert.Equal(t, tt.wantCore, gotCore)
			assert.Equal(t, tt.wantSuffix, gotSuffix)
		})
	}
}
