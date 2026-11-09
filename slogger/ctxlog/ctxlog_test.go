// SPDX-License-Identifier: BSD-3-Clause

package ctxlog_test

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"

	. "github.com/nbgrp/pkg/slogger/ctxlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLogger returns a buffer, a logger writing to it and a context with the
// logger attached.
func setupLogger(t *testing.T) (*bytes.Buffer, *slog.Logger, context.Context) {
	t.Helper()

	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	return buf, logger, ToContext(t.Context(), logger)
}

func TestToContext(t *testing.T) {
	t.Parallel()

	_, logger, ctx := setupLogger(t)

	assert.Same(t, logger, FromContext(ctx))
}

func TestFromContext_Default(t *testing.T) {
	t.Parallel()

	assert.Same(t, slog.Default(), FromContext(t.Context()))
}

func TestFromContext_NilLogger(t *testing.T) {
	t.Parallel()

	ctx := ToContext(t.Context(), nil)

	assert.Same(t, slog.Default(), FromContext(ctx))
}

func TestLogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(ctx context.Context)
		want string
	}{
		{
			name: "debug",
			log:  func(ctx context.Context) { Debug(ctx, "Message") },
			want: "level=DEBUG msg=Message",
		},
		{
			name: "info",
			log:  func(ctx context.Context) { Info(ctx, "Message") },
			want: "level=INFO msg=Message",
		},
		{
			name: "warn",
			log:  func(ctx context.Context) { Warn(ctx, "Message") },
			want: "level=WARN msg=Message",
		},
		{
			name: "error",
			log:  func(ctx context.Context) { Error(ctx, "Message") },
			want: "level=ERROR msg=Message",
		},
		{
			name: "log",
			log:  func(ctx context.Context) { Log(ctx, slog.LevelWarn, "Message", "key", "value") },
			want: "level=WARN msg=Message key=value",
		},
		{
			name: "log_attrs",
			log:  func(ctx context.Context) { LogAttrs(ctx, slog.LevelError, "Message", slog.String("key", "value")) },
			want: "level=ERROR msg=Message key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf, _, ctx := setupLogger(t)

			tt.log(ctx)

			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestAttrs(t *testing.T) {
	t.Parallel()

	buf, _, ctx := setupLogger(t)

	Info(ctx, "Message", "key", "value", "num", 42)

	out := buf.String()
	assert.Contains(t, out, "key=value")
	assert.Contains(t, out, "num=42")
}

func TestSlogAttr(t *testing.T) {
	t.Parallel()

	buf, _, ctx := setupLogger(t)

	Info(ctx, "Message", slog.String("key", "value"))

	assert.Contains(t, buf.String(), "key=value")
}

func TestAttrsMixed(t *testing.T) {
	t.Parallel()

	buf, _, ctx := setupLogger(t)

	Info(ctx, "Message", "key", "value", slog.String("tenant", "acme"))

	out := buf.String()
	assert.Contains(t, out, "key=value")
	assert.Contains(t, out, "tenant=acme")
}

func TestWith(t *testing.T) {
	t.Parallel()

	buf, _, ctx := setupLogger(t)

	ctx = With(ctx, "request_id", "abc")
	Info(ctx, "Message")

	assert.Contains(t, buf.String(), "request_id=abc")
}

func TestWithSlogAttr(t *testing.T) {
	t.Parallel()

	buf, _, ctx := setupLogger(t)

	ctx = With(ctx, slog.String("request_id", "abc"))
	Info(ctx, "Message")

	assert.Contains(t, buf.String(), "request_id=abc")
}

func TestWithMixedAttrs(t *testing.T) {
	t.Parallel()

	buf, _, ctx := setupLogger(t)

	ctx = With(ctx, "request_id", "abc", slog.String("tenant", "acme"))
	Info(ctx, "Message")

	out := buf.String()
	assert.Contains(t, out, "request_id=abc")
	assert.Contains(t, out, "tenant=acme")
}

func TestWith_NoArgs(t *testing.T) {
	t.Parallel()

	_, logger, ctx := setupLogger(t)

	ctx = With(ctx)

	assert.Same(t, logger, FromContext(ctx))
}

func TestWith_Scope(t *testing.T) {
	t.Parallel()

	buf, _, ctx := setupLogger(t)

	scoped := With(ctx, "request_id", "abc")

	Info(scoped, "Message")
	Info(ctx, "Message")

	// The attributes are bound to the derived context only.
	assert.Equal(t, 1, strings.Count(buf.String(), "request_id=abc"))
}

func TestWith_DefaultLogger(t *testing.T) {
	t.Parallel()

	ctx := With(t.Context(), "request_id", "abc")

	assert.NotSame(t, slog.Default(), FromContext(ctx))
}

func TestErrorAttr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  func(ctx context.Context)
		want string
	}{
		{
			name: "value",
			log:  func(ctx context.Context) { Error(ctx, "Message", "error", context.Canceled) },
			want: `error="context canceled"`,
		},
		{
			name: "attr",
			log:  func(ctx context.Context) { Error(ctx, "Message", slog.Any("error", context.Canceled)) },
			want: `error="context canceled"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf, _, ctx := setupLogger(t)

			tt.log(ctx)

			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

// TestContextPropagation checks that the context passed to the logging call
// reaches the handler. The package uses the *Context methods of [slog.Logger],
// so a context-aware handler can take the values from it, e.g. a request ID or
// a trace span.
func TestContextPropagation(t *testing.T) {
	t.Parallel()

	handler := &captureHandler{}
	ctx := ToContext(t.Context(), slog.New(handler))

	type ctxKey struct{}
	ctx = context.WithValue(ctx, ctxKey{}, "value")

	Info(ctx, "Message")

	contexts := handler.contexts()
	require.Len(t, contexts, 1)
	assert.Equal(t, "value", contexts[0].Value(ctxKey{}))
}

// captureHandler is a logger handler that captures the contexts it is called
// with.
type captureHandler struct {
	ctxs []context.Context
	mu   sync.Mutex
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

// Handle implements [slog.Handler].
func (h *captureHandler) Handle(ctx context.Context, _ slog.Record) error { //nolint:gocritic // The signature is fixed by slog.Handler.
	h.mu.Lock()
	defer h.mu.Unlock()

	h.ctxs = append(h.ctxs, ctx)

	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

// contexts returns the captured contexts.
func (h *captureHandler) contexts() []context.Context {
	h.mu.Lock()
	defer h.mu.Unlock()

	return slices.Clone(h.ctxs)
}
