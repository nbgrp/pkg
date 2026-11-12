// SPDX-License-Identifier: BSD-3-Clause

package strcase_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/nbgrp/pkg/strcase"
	"github.com/stretchr/testify/assert"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Mode
	}{
		{
			name:  "lowerCamelCase",
			input: "lowerCamelCase",
			want:  CamelCase,
		},
		{
			name:  "UpperCamelCase",
			input: "UpperCamelCase",
			want:  CamelCase,
		},
		{
			name:  "lower_snake_case",
			input: "lower_snake_case",
			want:  SnakeCase,
		},
		{
			name:  "SCREAMING_SNAKE_CASE",
			input: "SCREAMING_SNAKE_CASE",
			want:  SnakeCase,
		},
		{
			name:  "MixEd_snaKe_CaSe",
			input: "MixEd_snaKe_CaSe",
			want:  SnakeCase,
		},
		{
			name:  "lower-kebab-case",
			input: "lower-kebab-case",
			want:  KebabCase,
		},
		{
			name:  "UPPER-KEBAB-CASE",
			input: "UPPER-KEBAB-CASE",
			want:  KebabCase,
		},
		{
			name:  "MixEd-kebab-CaSe",
			input: "MixEd-kebab-CaSe",
			want:  KebabCase,
		},
		{
			name:  "flatcase",
			input: "flatcase",
			want:  FlatCase,
		},
		{
			name:  "FLATCASE",
			input: "FLATCASE",
			want:  FlatCase,
		},
		{
			name:  "empty input string",
			input: "",
			want:  FlatCase, // fallback
		},
		{
			name:  "loOoo...ong_string",
			input: fmt.Sprintf("loOoo%song_string", strings.Repeat("o", 100)),
			want:  CamelCase, // underscore is far from the beginning of the string
		},
		{
			name:  "snake_case with the leading separators",
			input: "__lower_snake_case",
			want:  SnakeCase,
		},
		{
			name:  "camelCase with the leading separators",
			input: "__lowerCamelCase",
			want:  CamelCase,
		},
		{
			name:  "kebab-case with the trailing separators",
			input: "lower-kebab-case--",
			want:  KebabCase,
		},
		{
			name:  "a single word with the leading separators",
			input: "__flatcase",
			want:  FlatCase,
		},
		{
			name:  "the separators only",
			input: "__",
			want:  FlatCase,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMode_ConvertTo(t *testing.T) {
	inputs := map[string]Mode{
		"bonjourLeMonde":   CamelCase,
		"bonjour_le_monde": SnakeCase,
		"bonjour-le-monde": KebabCase,
	}
	targets := map[Mode]string{
		CamelCase: "bonjourLeMonde",
		SnakeCase: "bonjour_le_monde",
		KebabCase: "bonjour-le-monde",
		FlatCase:  "bonjourlemonde",
	}

	for inputString, inputCase := range inputs {
		for targetCase, wantString := range targets {
			t.Run(fmt.Sprintf("%s to %s", inputString, wantString), func(t *testing.T) {
				got := inputCase.ConvertTo(targetCase, inputString)
				assert.Equal(t, wantString, got)
			})
		}
	}
}

func TestCase_To(t *testing.T) {
	inputs := []string{
		"bonjourLeMonde",
		"bonjour_le_monde",
		"bonjour-le-monde",
	}
	targets := map[Mode]string{
		CamelCase: "bonjourLeMonde",
		SnakeCase: "bonjour_le_monde",
		KebabCase: "bonjour-le-monde",
		FlatCase:  "bonjourlemonde",
	}

	for _, inputString := range inputs {
		for targetCase, wantString := range targets {
			t.Run(fmt.Sprintf("%s to %s", inputString, wantString), func(t *testing.T) {
				got := Convert(inputString).To(targetCase)
				assert.Equal(t, wantString, got)
			})
		}
	}
}

func TestCase_To_EdgeSeparators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target Mode
		want   string
	}{
		{
			name:   "the leading separators are kept",
			input:  "__bonjour_le_monde",
			target: CamelCase,
			want:   "__bonjourLeMonde",
		},
		{
			name:   "the trailing separators are kept",
			input:  "bonjour_le_monde__",
			target: CamelCase,
			want:   "bonjourLeMonde__",
		},
		{
			name:   "the leading separators are not replaced",
			input:  "__bonjour_le_monde",
			target: KebabCase,
			want:   "__bonjour-le-monde",
		},
		{
			name:   "the trailing separators are not replaced",
			input:  "bonjour_le_monde__",
			target: KebabCase,
			want:   "bonjour-le-monde__",
		},
		{
			name:   "the leading separators are not removed",
			input:  "__bonjour_le_monde",
			target: FlatCase,
			want:   "__bonjourlemonde",
		},
		{
			name:   "the camel case is split into the words",
			input:  "__bonjourLeMonde",
			target: SnakeCase,
			want:   "__bonjour_le_monde",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input).To(tt.target)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCase_To_EdgeSeparatorsRoundTrip(t *testing.T) {
	got := Convert("__bonjour_le_monde").To(CamelCase)
	assert.Equal(t, "__bonjourLeMonde", got)

	got = Convert(got).To(SnakeCase)
	assert.Equal(t, "__bonjour_le_monde", got)
}
