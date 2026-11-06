// SPDX-License-Identifier: BSD-3-Clause

package grpcslog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			original := exit
			t.Cleanup(func() { exit = original })

			var code int
			exit = func(c int) { code = c }

			buf := &bytes.Buffer{}
			l, err := New(slog.New(slog.NewJSONHandler(buf, nil)))
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

// record is a logged message.
type record struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
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
