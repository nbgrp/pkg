// SPDX-License-Identifier: BSD-3-Clause

package time_test

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/nbgrp/pkg/time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    Duration
		want []byte
	}{
		{
			name: "zero",
			d:    Duration(0),
			want: []byte(`"0s"`),
		},
		{
			name: "seconds",
			d:    Duration(30 * time.Second),
			want: []byte(`"30s"`),
		},
		{
			name: "hours",
			d:    Duration(5 * time.Hour),
			want: []byte(`"5h0m0s"`),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.d)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    []byte
		want Duration
	}{
		{
			name: "zero",
			d:    []byte(`"0s"`),
			want: Duration(0),
		},
		{
			name: "4 minutes and 30 second",
			d:    []byte(`"4m30s"`),
			want: Duration(4*time.Minute + 30*time.Second),
		},
		{
			name: "5 milliseconds",
			d:    []byte(`"5ms"`),
			want: Duration(5 * time.Millisecond),
		},
		{
			name: "600 nanoseconds",
			d:    []byte(`600`),
			want: Duration(600 * time.Nanosecond),
		},
		{
			name: "float 10 nanoseconds",
			d:    []byte(`10.5`),
			want: Duration(10 * time.Nanosecond),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got Duration
			err := json.Unmarshal(tt.d, &got)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDuration_UnmarshalJSON_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		d       []byte
		wantErr string
	}{
		{
			name:    "invalid duration string",
			d:       []byte(`"bogus"`),
			wantErr: `time: invalid duration "bogus"`,
		},
		{
			name:    "empty string",
			d:       []byte(`""`),
			wantErr: `time: invalid duration ""`,
		},
		{
			name:    "boolean",
			d:       []byte(`true`),
			wantErr: "invalid duration: true (bool)",
		},
		{
			name:    "object",
			d:       []byte(`{}`),
			wantErr: "invalid duration: map[] (map[string]interface {})",
		},
		{
			name:    "array",
			d:       []byte(`[]`),
			wantErr: "invalid duration: [] ([]interface {})",
		},
		{
			name:    "null",
			d:       []byte(`null`),
			wantErr: "invalid duration: <nil> (<nil>)",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Duration(time.Second)
			err := json.Unmarshal(tt.d, &got)
			require.EqualError(t, err, tt.wantErr)
			assert.Equal(t, Duration(time.Second), got)
		})
	}
}

func TestDuration_MarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    Duration
		want []byte
	}{
		{
			name: "zero",
			d:    Duration(0),
			want: []byte("0s\n"),
		},
		{
			name: "seconds",
			d:    Duration(30 * time.Second),
			want: []byte("30s\n"),
		},
		{
			name: "hours",
			d:    Duration(5 * time.Hour),
			want: []byte("5h0m0s\n"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := yaml.Marshal(tt.d)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    []byte
		want Duration
	}{
		{
			name: "zero with quotes",
			d:    []byte(`"0s"`),
			want: Duration(0),
		},
		{
			name: "4 minutes and 30 second",
			d:    []byte(`4m30s`),
			want: Duration(4*time.Minute + 30*time.Second),
		},
		{
			name: "5 milliseconds",
			d:    []byte(`5ms`),
			want: Duration(5 * time.Millisecond),
		},
		{
			name: "600 nanoseconds",
			d:    []byte(`600`),
			want: Duration(600 * time.Nanosecond),
		},
		{
			name: "float 10 nanoseconds",
			d:    []byte(`10.5`),
			want: Duration(10 * time.Nanosecond),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got Duration
			err := yaml.Unmarshal(tt.d, &got)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDuration_UnmarshalYAML_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		d       []byte
		wantErr string
	}{
		{
			name:    "invalid duration string",
			d:       []byte(`"bogus"`),
			wantErr: `time: invalid duration "bogus"`,
		},
		{
			name:    "empty string",
			d:       []byte(`""`),
			wantErr: `time: invalid duration ""`,
		},
		{
			name:    "boolean",
			d:       []byte(`true`),
			wantErr: "invalid duration: true (bool)",
		},
		{
			name:    "object",
			d:       []byte(`{}`),
			wantErr: "invalid duration: map[] (map[string]interface {})",
		},
		{
			name:    "array",
			d:       []byte(`[]`),
			wantErr: "invalid duration: [] ([]interface {})",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Duration(time.Second)
			err := yaml.Unmarshal(tt.d, &got)
			require.EqualError(t, err, tt.wantErr)
			assert.Equal(t, Duration(time.Second), got)
		})
	}
}
