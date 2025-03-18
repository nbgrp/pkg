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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCamel(tt.input, tt.sep)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_fromCamel(t *testing.T) {
	tests := []struct { //nolint:govet
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fromCamel(tt.input, tt.sep)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_replaceSep(t *testing.T) {
	tests := []struct { //nolint:govet
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
			inputStr: "bonjour_le-monde",
			fromSep:  '-',
			toSep:    noSep,
			want:     "bonjour_lemonde",
		},
		{
			name:     "noSep to anything",
			inputStr: "bonjour-le-monde",
			fromSep:  noSep,
			toSep:    '#',
			want:     "bonjour-le-monde",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceSep(tt.inputStr, tt.fromSep, tt.toSep)
			assert.Equal(t, tt.want, got)
		})
	}
}
