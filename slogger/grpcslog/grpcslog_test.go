// SPDX-License-Identifier: BSD-3-Clause

package grpcslog_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"

	. "github.com/nbgrp/pkg/slogger/grpcslog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// record is a logged message.
type record struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

// safeBuffer is a buffer that is safe for concurrent use.
type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// newLogger returns a logger that writes the JSON records to the given buffer.
func newLogger(buf *safeBuffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, nil))
}

// TestV checks the verbosity semantics of the logger.
//
// The verbosity level of gRPC is a threshold: the higher it is, the more
// verbose messages are logged. A message of the verbose level is logged if
// its level is not greater than the verbosity level of the logger.
func TestV(t *testing.T) {
	tests := []struct {
		name      string
		verbosity int
		level     int
		want      bool
	}{
		{name: "level 0 with zero verbosity", verbosity: 0, level: 0, want: true},
		{name: "level 1 with zero verbosity", verbosity: 0, level: 1, want: false},
		{name: "level 2 with zero verbosity", verbosity: 0, level: 2, want: false},
		{name: "level 0 with verbosity 2", verbosity: 2, level: 0, want: true},
		{name: "level 1 with verbosity 2", verbosity: 2, level: 1, want: true},
		{name: "level 2 with verbosity 2", verbosity: 2, level: 2, want: true},
		{name: "level 3 with verbosity 2", verbosity: 2, level: 3, want: false},
		{name: "level 2 with verbosity 3", verbosity: 3, level: 2, want: true},
		{name: "level 4 with verbosity 3", verbosity: 3, level: 4, want: false},
		{name: "negative level with zero verbosity", verbosity: 0, level: -1, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "")

			l, err := New(newLogger(&safeBuffer{}), WithVerbosity(tt.verbosity))
			require.NoError(t, err)

			assert.Equal(t, tt.want, l.V(tt.level))
		})
	}
}

// TestNew_Verbosity checks that the verbosity level is taken from the
// environment variable.
func TestNew_Verbosity(t *testing.T) {
	t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "2")

	l, err := New(newLogger(&safeBuffer{}))
	require.NoError(t, err)

	assert.True(t, l.V(0))
	assert.True(t, l.V(1))
	assert.True(t, l.V(2))
	assert.False(t, l.V(3))
}

// TestNew_VerbosityEnvPrecedence checks that the environment variable takes
// precedence over the verbosity level passed to the constructor.
func TestNew_VerbosityEnvPrecedence(t *testing.T) {
	t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "3")

	l, err := New(newLogger(&safeBuffer{}), WithVerbosity(1))
	require.NoError(t, err)

	assert.True(t, l.V(3))
	assert.False(t, l.V(4))
}

// TestNew_InvalidEnv checks that the invalid values of the environment
// variables are reported.
func TestNew_InvalidEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "invalid verbosity level",
			value: "bogus",
			want:  `invalid gRPC log verbosity level "bogus"`,
		},
		{
			name:  "negative verbosity level",
			value: "-1",
			want:  `invalid gRPC log verbosity level "-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", tt.value)

			_, err := New(newLogger(&safeBuffer{}))
			require.ErrorContains(t, err, tt.want)
		})
	}
}

// TestLogMethods checks that the messages are logged with the expected
// severity levels and contents.
func TestLogMethods(t *testing.T) {
	tests := []struct {
		name  string
		log   func(l *GrpcLogger)
		level string
		msg   string
	}{
		{
			name:  "Info",
			log:   func(l *GrpcLogger) { l.Info("message") },
			level: "INFO",
			msg:   "message",
		},
		{
			name:  "Infoln",
			log:   func(l *GrpcLogger) { l.Infoln("message") },
			level: "INFO",
			msg:   "message\n",
		},
		{
			name:  "Infof",
			log:   func(l *GrpcLogger) { l.Infof("mes%s", "sage") },
			level: "INFO",
			msg:   "message",
		},
		{
			name:  "Warning",
			log:   func(l *GrpcLogger) { l.Warning("message") },
			level: "WARN",
			msg:   "message",
		},
		{
			name:  "Warningln",
			log:   func(l *GrpcLogger) { l.Warningln("message") },
			level: "WARN",
			msg:   "message\n",
		},
		{
			name:  "Warningf",
			log:   func(l *GrpcLogger) { l.Warningf("mes%s", "sage") },
			level: "WARN",
			msg:   "message",
		},
		{
			name:  "Error",
			log:   func(l *GrpcLogger) { l.Error("message") },
			level: "ERROR",
			msg:   "message",
		},
		{
			name:  "Errorln",
			log:   func(l *GrpcLogger) { l.Errorln("message") },
			level: "ERROR",
			msg:   "message\n",
		},
		{
			name:  "Errorf",
			log:   func(l *GrpcLogger) { l.Errorf("mes%s", "sage") },
			level: "ERROR",
			msg:   "message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &safeBuffer{}
			l, err := New(newLogger(buf))
			require.NoError(t, err)

			tt.log(l)

			records := parseRecords(t, buf.String())
			require.Len(t, records, 1)
			assert.Equal(t, tt.level, records[0].Level)
			assert.Equal(t, tt.msg, records[0].Msg)
		})
	}
}

// parseRecords parses the JSON messages written to the buffer.
func parseRecords(t *testing.T, out string) []record {
	t.Helper()

	var records []record

	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}

		var r record
		require.NoError(t, json.Unmarshal([]byte(line), &r))

		records = append(records, r)
	}

	return records
}
