// SPDX-License-Identifier: BSD-3-Clause

package slogger

import (
	"context"
	"log/slog"
)

type levelKey struct{}

// WithLevel returns a copy of ctx carrying the given log level.
//
// The level of the context takes precedence over the level of the logger: the
// records below it are dropped, and the ones above it are written, whatever
// the level of the logger is.
func WithLevel(ctx context.Context, level slog.Level) context.Context {
	return context.WithValue(ctx, levelKey{}, level)
}

// levelFromContext returns the log level carried by ctx and whether it is set.
func levelFromContext(ctx context.Context) (slog.Level, bool) {
	level, ok := ctx.Value(levelKey{}).(slog.Level)

	return level, ok
}

var _ slog.Handler = (*levelHandler)(nil)

// levelHandler is a [slog.Handler] that takes the log level from the context.
//
// If the context carries a log level, it takes precedence over the level of
// the wrapped handler. Otherwise the check is passed to the wrapped handler.
type levelHandler struct {
	next slog.Handler
}

// NewLevelHandler returns a handler that takes the log level from the context
// and passes the other calls to the given handler.
//
// It is the way to apply the level of the context to a logger that is not
// built by [New] or [NewToContext].
func NewLevelHandler(next slog.Handler) slog.Handler {
	return &levelHandler{next: next}
}

// Enabled reports whether the handler handles the records of the given level.
//
// The level of the context takes precedence: if the context carries a log
// level, the record is handled when its level is not lower than the level of
// the context. Otherwise the check is passed to the wrapped handler.
func (h *levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if ctxLevel, ok := levelFromContext(ctx); ok {
		return level >= ctxLevel
	}

	return h.next.Enabled(ctx, level)
}

func (h *levelHandler) Handle(ctx context.Context, record slog.Record) error { //nolint:gocritic
	return h.next.Handle(ctx, record)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{next: h.next.WithAttrs(attrs)}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{next: h.next.WithGroup(name)}
}
