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
