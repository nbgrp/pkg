// SPDX-License-Identifier: BSD-3-Clause

package slogger_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	. "github.com/nbgrp/pkg/slogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevelHandler_Enabled(t *testing.T) {
	t.Parallel()

	t.Run("passes the check to the nested handler", func(t *testing.T) {
		t.Parallel()

		handler := NewLevelHandler(&stubHandler{enabled: false})

		assert.False(t, handler.Enabled(t.Context(), slog.LevelInfo))
	})

	t.Run("passes the enabled check to the nested handler", func(t *testing.T) {
		t.Parallel()

		handler := NewLevelHandler(&stubHandler{enabled: true})

		assert.True(t, handler.Enabled(t.Context(), slog.LevelInfo))
	})

	t.Run("uses the level of the context", func(t *testing.T) {
		t.Parallel()

		handler := NewLevelHandler(&stubHandler{enabled: false})
		ctx := WithLevel(t.Context(), slog.LevelWarn)

		assert.False(t, handler.Enabled(ctx, slog.LevelInfo))
		assert.True(t, handler.Enabled(ctx, slog.LevelWarn))
		assert.True(t, handler.Enabled(ctx, slog.LevelError))
	})

	t.Run("the level of the context takes precedence over the nested handler", func(t *testing.T) {
		t.Parallel()

		handler := NewLevelHandler(&stubHandler{enabled: false})
		ctx := WithLevel(t.Context(), slog.LevelDebug)

		assert.True(t, handler.Enabled(ctx, slog.LevelDebug))
	})
}

func TestLevelHandler_Handle(t *testing.T) {
	t.Parallel()

	stub := &stubHandler{enabled: true}
	handler := NewLevelHandler(stub)

	require.NoError(t, handler.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "Message", 0)))

	require.Len(t, stub.records, 1)
	assert.Equal(t, "Message", stub.records[0].Message)
}

func TestLevelHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	stub := &stubHandler{enabled: true}
	handler := NewLevelHandler(stub).WithAttrs([]slog.Attr{slog.String("key", "value")})

	require.Len(t, stub.attrs, 1)
	assert.Equal(t, "key", stub.attrs[0].Key)

	// The level of the context is still taken into account.
	ctx := WithLevel(t.Context(), slog.LevelError)
	assert.False(t, handler.Enabled(ctx, slog.LevelInfo))
}

func TestLevelHandler_WithGroup(t *testing.T) {
	t.Parallel()

	stub := &stubHandler{enabled: true}
	handler := NewLevelHandler(stub).WithGroup("group")

	require.Len(t, stub.groups, 1)
	assert.Equal(t, "group", stub.groups[0])

	// The level of the context is still taken into account.
	ctx := WithLevel(t.Context(), slog.LevelError)
	assert.False(t, handler.Enabled(ctx, slog.LevelInfo))
}

func TestWithLevel(t *testing.T) {
	t.Parallel()

	t.Run("the innermost level wins", func(t *testing.T) {
		t.Parallel()

		handler := NewLevelHandler(&stubHandler{enabled: false})

		outer := WithLevel(t.Context(), slog.LevelError)
		assert.False(t, handler.Enabled(outer, slog.LevelInfo))

		inner := WithLevel(outer, slog.LevelDebug)
		assert.True(t, handler.Enabled(inner, slog.LevelInfo))
	})
}

// stubHandler is a handler that records the calls it receives and reports the
// records of the enabled level as handled.
type stubHandler struct {
	enabled bool
	records []slog.Record
	attrs   []slog.Attr
	groups  []string
}

// Enabled implements [slog.Handler].
func (h *stubHandler) Enabled(context.Context, slog.Level) bool {
	return h.enabled
}

// Handle implements [slog.Handler].
func (h *stubHandler) Handle(_ context.Context, record slog.Record) error { //nolint:gocritic // The signature is fixed by slog.Handler.
	h.records = append(h.records, record)

	return nil
}

// WithAttrs implements [slog.Handler].
func (h *stubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs, attrs...)

	return h
}

// WithGroup implements [slog.Handler].
func (h *stubHandler) WithGroup(name string) slog.Handler {
	h.groups = append(h.groups, name)

	return h
}
