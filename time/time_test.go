package time_test

import (
	"testing"
	"time"

	. "github.com/nbgrp/pkg/time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	//nolint:govet
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
			got, err := tt.d.MarshalJSON()
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
			name: "5 millisecond",
			d:    []byte(`"5ms"`),
			want: Duration(5 * time.Millisecond),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got Duration
			err := got.UnmarshalJSON(tt.d)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
