// SPDX-License-Identifier: BSD-3-Clause

package slogger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/nbgrp/pkg/slogger/ctxlog"
)

// Options configures the logger.
type Options struct {
	// Leveler returns the current log level.
	Leveler func() string
	// LevelReloadCond is the condition variable that notifies the logger about
	// a log level change.
	LevelReloadCond *sync.Cond
	// Writer is an optional writer the logs are written to.
	// If it is not set, the logs are written to [os.Stdout].
	Writer io.Writer
	// Format is the log output format. Allowed values: "json" and "text".
	Format string
}

// New initializes and returns a [slog.Logger] or an error.
func New(ctx context.Context, opts Options) (*slog.Logger, error) {
	level := &slog.LevelVar{}
	err := level.UnmarshalText([]byte(opts.Leveler()))
	if err != nil {
		return nil, err
	}

	var (
		handler slog.Handler
		options = &slog.HandlerOptions{Level: level}
	)

	var writer io.Writer = os.Stdout
	if opts.Writer != nil {
		writer = opts.Writer
	}

	logFormat := opts.Format
	switch logFormat {
	case "json":
		handler = slog.NewJSONHandler(writer, options)
	case "text":
		handler = slog.NewTextHandler(writer, options)
	default:
		return nil, fmt.Errorf("unknown log format %q", logFormat)
	}

	l := slog.New(handler)
	if opts.LevelReloadCond != nil {
		go watchLogLevel(ctx, l, opts.LevelReloadCond, level, opts)
	}

	return l, nil
}

// NewToContext initializes a logger and puts it into the returned context, so
// it can be used with the [ctxlog] helpers.
//
// The returned context is nil if the logger cannot be initialized.
func NewToContext(ctx context.Context, opts Options) (context.Context, error) {
	logger, err := New(ctx, opts)
	if err != nil {
		return nil, err
	}

	return ctxlog.ToContext(ctx, logger), nil
}

// watchLogLevel reloads the log level on every notification of the condition
// variable until the context is canceled.
func watchLogLevel(ctx context.Context, logger *slog.Logger, cond *sync.Cond, currentLevel *slog.LevelVar, opts Options) {
	// A [sync.Cond] has no context support, so the watcher has to be woken up
	// from the outside when the context is canceled.
	//
	// The lock is held during the broadcast: the watcher registers itself as a
	// waiter before it releases the lock, so a broadcast sent without the lock
	// could be lost and leave the watcher sleeping on a canceled context.
	stop := context.AfterFunc(ctx, func() {
		cond.L.Lock()
		defer cond.L.Unlock()

		cond.Broadcast()
	})
	defer stop()

	cond.L.Lock()
	defer cond.L.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var newLevel slog.Level
		err := newLevel.UnmarshalText([]byte(opts.Leveler()))

		switch {
		case err != nil:
			logger.ErrorContext(ctx, "Failed to parse log level while hot-reload update", slog.Any("error", err))
		case newLevel != currentLevel.Level():
			oldLevel := currentLevel.Level().String()
			currentLevel.Set(newLevel)
			_ = logger.Handler().Handle(ctx, slog.NewRecord(
				time.Now(),
				slog.LevelInfo,
				fmt.Sprintf("Log level changed from %s to %s", oldLevel, newLevel.String()),
				0,
			))
		}

		cond.Wait()
	}
}
