// SPDX-License-Identifier: BSD-3-Clause

package grpcslog

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// record is a logged message.
type record struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
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
			t.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "INFO")
			t.Setenv("GRPC_GO_LOG_FORMATTER", "json")

			buf := &bytes.Buffer{}
			l, err := newGrpcLoggerWithWriter(t.Context(), 0, buf)
			require.NoError(t, err)

			tt.log(l)

			records := parseRecords(t, buf.String())
			require.Len(t, records, 1)
			assert.Equal(t, tt.level, records[0].Level)
			assert.Equal(t, tt.msg, records[0].Msg)
		})
	}
}

// TestFatalMethods checks that the fatal methods log the message and exit the
// process with the code 1.
func TestFatalMethods(t *testing.T) {
	tests := []struct {
		name string
		log  func(l *GrpcLogger)
	}{
		{name: "Fatal", log: func(l *GrpcLogger) { l.Fatal("fatal message") }},
		{name: "Fatalln", log: func(l *GrpcLogger) { l.Fatalln("fatal message") }},
		{name: "Fatalf", log: func(l *GrpcLogger) { l.Fatalf("fatal %s", "message") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "ERROR")
			t.Setenv("GRPC_GO_LOG_FORMATTER", "json")

			orig := exit
			t.Cleanup(func() { exit = orig })

			var code int
			exit = func(c int) { code = c }

			buf := &bytes.Buffer{}
			l, err := newGrpcLoggerWithWriter(t.Context(), 0, buf)
			require.NoError(t, err)

			tt.log(l)

			assert.Equal(t, 1, code)

			records := parseRecords(t, buf.String())
			require.Len(t, records, 1)
			assert.Equal(t, "ERROR", records[0].Level)
			assert.Contains(t, records[0].Msg, "fatal message")
		})
	}
}

// TestSeverityLevel checks that the severity level is taken from the
// environment variable.
func TestSeverityLevel(t *testing.T) {
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

			buf := &bytes.Buffer{}
			l, err := newGrpcLoggerWithWriter(t.Context(), 0, buf)
			require.NoError(t, err)

			l.Info("info message")
			l.Warning("warning message")
			l.Error("error message")

			got := make([]string, 0, len(tt.want))
			for _, record := range parseRecords(t, buf.String()) {
				got = append(got, record.Level)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

// TestLogFormat checks that the format of the messages is taken from the
// environment variable.
func TestLogFormat(t *testing.T) {
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

			buf := &bytes.Buffer{}
			l, err := newGrpcLoggerWithWriter(t.Context(), 0, buf)
			require.NoError(t, err)

			l.Info("info message")

			switch tt.want {
			case "json":
				records := parseRecords(t, buf.String())
				require.Len(t, records, 1)
				assert.Equal(t, "info message", records[0].Msg)
			case "text":
				assert.Contains(t, buf.String(), "level=INFO")
				assert.Contains(t, buf.String(), "msg=\"info message\"")
			}
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
