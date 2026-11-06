// SPDX-License-Identifier: BSD-3-Clause

package grpcslog_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/nbgrp/pkg/slogger/grpcslog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fatalMethodEnvVar tells the helper test which fatal method to call.
const fatalMethodEnvVar = "GRPCSLOG_TEST_FATAL_METHOD"

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

			l, err := grpcslog.NewGrpcLoggerWithVerbosity(t.Context(), tt.verbosity)
			require.NoError(t, err)

			assert.Equal(t, tt.want, l.V(tt.level))
		})
	}
}

// TestV_EnabledLevels checks that the enabled verbose levels form a prefix:
// if a level is enabled, all the lower levels are enabled too.
func TestV_EnabledLevels(t *testing.T) {
	for verbosity := range 5 {
		t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "")

		l, err := grpcslog.NewGrpcLoggerWithVerbosity(t.Context(), verbosity)
		require.NoError(t, err)

		for level := range 5 {
			want := level <= verbosity

			assert.Equalf(t, want, l.V(level), "verbosity %d, level %d", verbosity, level)
		}
	}
}

// TestNewGrpcLogger_Verbosity checks that the verbosity level is taken from
// the environment variable.
func TestNewGrpcLogger_Verbosity(t *testing.T) {
	t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "2")

	l, err := grpcslog.NewGrpcLogger(t.Context())
	require.NoError(t, err)

	assert.True(t, l.V(0))
	assert.True(t, l.V(1))
	assert.True(t, l.V(2))
	assert.False(t, l.V(3))
}

// TestNewGrpcLoggerWithVerbosity_EnvPrecedence checks that the environment
// variable takes precedence over the verbosity level passed to the
// constructor.
func TestNewGrpcLoggerWithVerbosity_EnvPrecedence(t *testing.T) {
	t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "3")

	l, err := grpcslog.NewGrpcLoggerWithVerbosity(t.Context(), 1)
	require.NoError(t, err)

	assert.True(t, l.V(3))
	assert.False(t, l.V(4))
}

// TestNewGrpcLoggerWithVerbosity_EnvUnset checks that the verbosity level
// passed to the constructor is used if the environment variable is not set.
func TestNewGrpcLoggerWithVerbosity_EnvUnset(t *testing.T) {
	t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "")

	l, err := grpcslog.NewGrpcLoggerWithVerbosity(t.Context(), 2)
	require.NoError(t, err)

	assert.True(t, l.V(2))
	assert.False(t, l.V(3))
}

// TestNewGrpcLogger_SeverityLevel checks that the severity level is taken
// from the environment variable.
func TestNewGrpcLogger_SeverityLevel(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "unset", value: "", want: []string{"ERROR"}},
		{name: "error", value: "ERROR", want: []string{"ERROR"}},
		{name: "warning", value: "WARNING", want: []string{"WARN", "ERROR"}},
		{name: "info", value: "INFO", want: []string{"INFO", "WARN", "ERROR"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", tt.value)
			t.Setenv("GRPC_GO_LOG_FORMATTER", "json")

			out := captureStderr(t, func() {
				l, err := grpcslog.NewGrpcLogger(t.Context())
				require.NoError(t, err)

				l.Info("info message")
				l.Warning("warning message")
				l.Error("error message")
			})

			var got []string
			for _, record := range parseRecords(t, out) {
				got = append(got, record.Level)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNewGrpcLogger_Format checks that the format of the messages is taken
// from the environment variable.
func TestNewGrpcLogger_Format(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "unset", value: "", want: "text"},
		{name: "text", value: "text", want: "text"},
		{name: "json", value: "json", want: "json"},
		{name: "json in upper case", value: "JSON", want: "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "INFO")
			t.Setenv("GRPC_GO_LOG_FORMATTER", tt.value)

			out := captureStderr(t, func() {
				l, err := grpcslog.NewGrpcLogger(t.Context())
				require.NoError(t, err)

				l.Info("info message")
			})

			switch tt.want {
			case "json":
				records := parseRecords(t, out)
				require.Len(t, records, 1)
				assert.Equal(t, "info message", records[0].Msg)
			case "text":
				assert.Contains(t, out, "level=INFO")
				assert.Contains(t, out, "msg=\"info message\"")
			}
		})
	}
}

// TestNewGrpcLogger_InvalidEnv checks that the invalid values of the
// environment variables are reported.
func TestNewGrpcLogger_InvalidEnv(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{
			name:  "unknown severity level",
			key:   "GRPC_GO_LOG_SEVERITY_LEVEL",
			value: "bogus",
			want:  `unknown gRPC log severity level "bogus"`,
		},
		{
			name:  "invalid verbosity level",
			key:   "GRPC_GO_LOG_VERBOSITY_LEVEL",
			value: "bogus",
			want:  `invalid gRPC log verbosity level "bogus"`,
		},
		{
			name:  "negative verbosity level",
			key:   "GRPC_GO_LOG_VERBOSITY_LEVEL",
			value: "-1",
			want:  `invalid gRPC log verbosity level "-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "")
			t.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "")
			t.Setenv(tt.key, tt.value)

			_, err := grpcslog.NewGrpcLogger(t.Context())
			require.ErrorContains(t, err, tt.want)
		})
	}
}

// TestLogMethods checks that the messages are logged with the expected
// severity levels and contents.
func TestLogMethods(t *testing.T) {
	tests := []struct {
		name  string
		log   func(l *grpcslog.GrpcLogger)
		level string
		msg   string
	}{
		{
			name:  "Info",
			log:   func(l *grpcslog.GrpcLogger) { l.Info("message") },
			level: "INFO",
			msg:   "message",
		},
		{
			name:  "Infoln",
			log:   func(l *grpcslog.GrpcLogger) { l.Infoln("message") },
			level: "INFO",
			msg:   "message\n",
		},
		{
			name:  "Infof",
			log:   func(l *grpcslog.GrpcLogger) { l.Infof("mes%s", "sage") },
			level: "INFO",
			msg:   "message",
		},
		{
			name:  "Warning",
			log:   func(l *grpcslog.GrpcLogger) { l.Warning("message") },
			level: "WARN",
			msg:   "message",
		},
		{
			name:  "Warningln",
			log:   func(l *grpcslog.GrpcLogger) { l.Warningln("message") },
			level: "WARN",
			msg:   "message\n",
		},
		{
			name:  "Warningf",
			log:   func(l *grpcslog.GrpcLogger) { l.Warningf("mes%s", "sage") },
			level: "WARN",
			msg:   "message",
		},
		{
			name:  "Error",
			log:   func(l *grpcslog.GrpcLogger) { l.Error("message") },
			level: "ERROR",
			msg:   "message",
		},
		{
			name:  "Errorln",
			log:   func(l *grpcslog.GrpcLogger) { l.Errorln("message") },
			level: "ERROR",
			msg:   "message\n",
		},
		{
			name:  "Errorf",
			log:   func(l *grpcslog.GrpcLogger) { l.Errorf("mes%s", "sage") },
			level: "ERROR",
			msg:   "message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "INFO")
			t.Setenv("GRPC_GO_LOG_FORMATTER", "json")

			out := captureStderr(t, func() {
				l, err := grpcslog.NewGrpcLogger(t.Context())
				require.NoError(t, err)

				tt.log(l)
			})

			records := parseRecords(t, out)
			require.Len(t, records, 1)
			assert.Equal(t, tt.level, records[0].Level)
			assert.Equal(t, tt.msg, records[0].Msg)
		})
	}
}

// TestFatalMethods checks that the fatal methods log the message and exit the
// process with the code 1.
//
// The methods are called in a subprocess, because they exit the process.
func TestFatalMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{name: "Fatal", method: "Fatal"},
		{name: "Fatalln", method: "Fatalln"},
		{name: "Fatalf", method: "Fatalf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer

			cmd := exec.Command(os.Args[0], "-test.run=^TestFatalHelper$")
			cmd.Env = append(os.Environ(), fatalMethodEnvVar+"="+tt.method)
			cmd.Stderr = &stderr

			err := cmd.Run()

			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, 1, exitErr.ExitCode())
			assert.Contains(t, stderr.String(), "fatal message")
		})
	}
}

// TestFatalHelper calls the fatal method selected by the environment
// variable. It is run in a subprocess by TestFatalMethods.
func TestFatalHelper(t *testing.T) {
	method := os.Getenv(fatalMethodEnvVar)
	if method == "" {
		t.Skip("not a helper process")
	}

	l, err := grpcslog.NewGrpcLogger(t.Context())
	require.NoError(t, err)

	switch method {
	case "Fatal":
		l.Fatal("fatal message")
	case "Fatalln":
		l.Fatalln("fatal message")
	case "Fatalf":
		l.Fatalf("fatal %s", "message")
	default:
		t.Fatalf("unknown method %q", method)
	}
}

// record is a logged message.
type record struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

// parseRecords parses the JSON messages logged to the standard error output.
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

// captureStderr redirects the standard error output to a pipe for the
// duration of the function and returns the captured output.
func captureStderr(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	f()

	require.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	return string(out)
}
