// SPDX-License-Identifier: BSD-3-Clause

package slogger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/nbgrp/pkg/slogger"
	"github.com/nbgrp/pkg/slogger/ctxlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestNew_Format(t *testing.T) {
	t.Parallel()

	t.Run("json", func(t *testing.T) {
		t.Parallel()

		buf := &safeBuffer{}
		logger, err := New(context.Background(), Options{
			Leveler: func() string { return "INFO" },
			Format:  "json",
			Writer:  buf,
		})
		require.NoError(t, err)

		logger.Info("Hello", slog.String("user_id", "42"))

		record := map[string]any{}
		require.NoError(t, json.Unmarshal([]byte(buf.String()), &record))
		assert.Equal(t, "INFO", record["level"])
		assert.Equal(t, "Hello", record["msg"])
		assert.Equal(t, "42", record["user_id"])
	})

	t.Run("text", func(t *testing.T) {
		t.Parallel()

		buf := &safeBuffer{}
		logger, err := New(context.Background(), Options{
			Leveler: func() string { return "INFO" },
			Format:  "text",
			Writer:  buf,
		})
		require.NoError(t, err)

		logger.Info("Hello", slog.String("user_id", "42"))

		out := buf.String()
		assert.Contains(t, out, "level=INFO")
		assert.Contains(t, out, "msg=Hello")
		assert.Contains(t, out, "user_id=42")
	})

	t.Run("unknown", func(t *testing.T) {
		t.Parallel()

		_, err := New(context.Background(), Options{
			Leveler: func() string { return "INFO" },
			Format:  "xml",
		})
		require.EqualError(t, err, `unknown log format "xml"`)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		_, err := New(context.Background(), Options{
			Leveler: func() string { return "INFO" },
		})
		require.EqualError(t, err, `unknown log format ""`)
	})
}

func TestNew_Level(t *testing.T) {
	t.Parallel()

	t.Run("filters the records", func(t *testing.T) {
		t.Parallel()

		buf := &safeBuffer{}
		logger, err := New(context.Background(), Options{
			Leveler: func() string { return "WARN" },
			Format:  "text",
			Writer:  buf,
		})
		require.NoError(t, err)

		logger.Info("Info message")
		logger.Warn("Warn message")
		logger.Error("Error message")

		out := buf.String()
		assert.NotContains(t, out, "Info message")
		assert.Contains(t, out, "Warn message")
		assert.Contains(t, out, "Error message")
	})

	t.Run("rejects an invalid level", func(t *testing.T) {
		t.Parallel()

		_, err := New(context.Background(), Options{
			Leveler: func() string { return "bogus" },
			Format:  "text",
		})
		require.ErrorContains(t, err, `slog: level string "bogus"`)
	})
}

func TestNew_CustomWriter(t *testing.T) {
	t.Parallel()

	buf := &safeBuffer{}
	logger, err := New(context.Background(), Options{
		Leveler: func() string { return "INFO" },
		Format:  "text",
		Writer:  buf,
	})
	require.NoError(t, err)

	logger.Info("Custom writer")

	assert.Contains(t, buf.String(), "Custom writer")
}

func TestNew_LevelReload(t *testing.T) {
	t.Run("lowers the level", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		r, cancel := newReloader(t, "INFO")
		defer cancel()

		r.logger.Debug("Dropped message")
		require.NotContains(t, r.buf.String(), "Dropped message")

		r.setLevel("DEBUG")
		requireEventually(t, r.buf, "Log level changed from INFO to DEBUG")

		r.logger.Debug("Accepted message")
		assert.Contains(t, r.buf.String(), "Accepted message")
	})

	t.Run("raises the level", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		r, cancel := newReloader(t, "DEBUG")
		defer cancel()

		r.logger.Debug("Accepted message")
		require.Contains(t, r.buf.String(), "Accepted message")

		r.setLevel("ERROR")
		// The notification is written through the handler, so it is not
		// filtered out by the new level.
		requireEventually(t, r.buf, "Log level changed from DEBUG to ERROR")
		requireEventuallyDropped(t, r.logger, r.buf)

		r.logger.Error("Error message")
		assert.Contains(t, r.buf.String(), "Error message")
	})

	t.Run("reports an invalid level", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		r, cancel := newReloader(t, "INFO")
		defer cancel()

		r.setLevel("bogus")
		requireEventually(t, r.buf, "Failed to parse log level while hot-reload update")

		// The current level is kept.
		r.logger.Info("Accepted message")
		assert.Contains(t, r.buf.String(), "Accepted message")
	})
}

func TestNewToContext(t *testing.T) {
	t.Parallel()

	t.Run("puts the logger into the context", func(t *testing.T) {
		t.Parallel()

		buf := &safeBuffer{}
		ctx, err := NewToContext(t.Context(), Options{
			Leveler: func() string { return "INFO" },
			Format:  "text",
			Writer:  buf,
		})
		require.NoError(t, err)

		logger := ctxlog.FromContext(ctx)
		require.NotNil(t, logger)
		assert.NotSame(t, slog.Default(), logger)

		logger.Info("Hello")

		assert.Contains(t, buf.String(), "Hello")
	})

	t.Run("applies the log level", func(t *testing.T) {
		t.Parallel()

		buf := &safeBuffer{}
		ctx, err := NewToContext(t.Context(), Options{
			Leveler: func() string { return "WARN" },
			Format:  "text",
			Writer:  buf,
		})
		require.NoError(t, err)

		ctxlog.Info(ctx, "Dropped message")
		ctxlog.Warn(ctx, "Accepted message")

		out := buf.String()
		assert.NotContains(t, out, "Dropped message")
		assert.Contains(t, out, "Accepted message")
	})

	t.Run("returns no context on error", func(t *testing.T) {
		t.Parallel()

		got, err := NewToContext(t.Context(), Options{
			Leveler: func() string { return "INFO" },
			Format:  "xml",
		})
		require.EqualError(t, err, `unknown log format "xml"`)
		assert.Nil(t, got)
	})
}

// reloader is a logger with a hot-reloadable log level.
type reloader struct {
	logger *slog.Logger
	cond   *sync.Cond
	buf    *safeBuffer
	// level is protected by the condition variable lock: the watcher reads it
	// while holding the lock, and the test writes it under the same lock.
	level string
}

// newReloader creates a logger with the given initial log level. The returned
// function cancels the context of the logger.
func newReloader(t *testing.T, level string) (*reloader, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	r := &reloader{
		cond:  sync.NewCond(&sync.Mutex{}),
		buf:   &safeBuffer{},
		level: level,
	}

	logger, err := New(ctx, Options{
		Leveler:         func() string { return r.level },
		Format:          "text",
		LevelReloadCond: r.cond,
		Writer:          r.buf,
	})
	require.NoError(t, err)

	r.logger = logger

	return r, cancel
}

// setLevel changes the log level and notifies the logger about it.
func (r *reloader) setLevel(level string) {
	r.cond.L.Lock()
	defer r.cond.L.Unlock()

	r.level = level
	r.cond.Broadcast()
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

// requireEventually waits until the buffer contains the given text.
func requireEventually(t *testing.T, buf *safeBuffer, want string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), want) {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	require.Failf(t, "timeout", "the record %q has not been written, got %q", want, buf.String())
}

// requireEventuallyDropped waits until the logger stops writing the records.
func requireEventuallyDropped(t *testing.T, logger *slog.Logger, buf *safeBuffer) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for i := 0; time.Now().Before(deadline); i++ {
		msg := fmt.Sprintf("Probe %d", i)
		logger.Info(msg)

		if !strings.Contains(buf.String(), msg) {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	require.Fail(t, "timeout", "the log level has not been changed")
}
