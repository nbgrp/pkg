// SPDX-License-Identifier: BSD-3-Clause

// Package grpcslog provides a [grpclog.LoggerV2] implementation that writes
// the gRPC messages to a [log/slog.Logger].
package grpcslog

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"google.golang.org/grpc/grpclog"
)

var _ grpclog.LoggerV2 = (*GrpcLogger)(nil)

// exit terminates the process after a fatal message. It is a variable to make
// it possible to test the fatal methods.
var exit = os.Exit

// GrpcLogger writes the gRPC messages to a [slog.Logger].
type GrpcLogger struct {
	l *slog.Logger
	v int
}

// New initializes and returns a [GrpcLogger] that writes the gRPC messages to
// the given logger.
//
// The GRPC_GO_LOG_VERBOSITY_LEVEL environment variable takes precedence over
// the verbosity level set with [WithVerbosity].
//
// It returns an error if the environment variable contains an invalid value.
func New(logger *slog.Logger, opts ...Option) (*GrpcLogger, error) {
	l := &GrpcLogger{l: logger}

	for _, opt := range opts {
		opt(l)
	}

	verbosity, err := verbosityLevel(l.v)
	if err != nil {
		return nil, err
	}
	l.v = verbosity

	return l, nil
}

func (l *GrpcLogger) Info(args ...any) {
	l.l.Info(fmt.Sprint(args...))
}

func (l *GrpcLogger) Infoln(args ...any) {
	l.l.Info(fmt.Sprintln(args...))
}

func (l *GrpcLogger) Infof(format string, args ...any) {
	l.l.Info(fmt.Sprintf(format, args...))
}

func (l *GrpcLogger) Warning(args ...any) {
	l.l.Warn(fmt.Sprint(args...))
}

func (l *GrpcLogger) Warningln(args ...any) {
	l.l.Warn(fmt.Sprintln(args...))
}

func (l *GrpcLogger) Warningf(format string, args ...any) {
	l.l.Warn(fmt.Sprintf(format, args...))
}

func (l *GrpcLogger) Error(args ...any) {
	l.l.Error(fmt.Sprint(args...))
}

func (l *GrpcLogger) Errorln(args ...any) {
	l.l.Error(fmt.Sprintln(args...))
}

func (l *GrpcLogger) Errorf(format string, args ...any) {
	l.l.Error(fmt.Sprintf(format, args...))
}

func (l *GrpcLogger) Fatal(args ...any) {
	l.l.Error(fmt.Sprint(args...))
	exit(1)
}

func (l *GrpcLogger) Fatalln(args ...any) {
	l.l.Error(fmt.Sprintln(args...))
	exit(1)
}

func (l *GrpcLogger) Fatalf(format string, args ...any) {
	l.l.Error(fmt.Sprintf(format, args...))
	exit(1)
}

// V reports whether the verbosity level is at least the requested verbose
// level, like the default gRPC logger does.
//
// The verbose messages of gRPC are written with the info severity, so they are
// only logged if the level of the logger allows the info records.
func (l *GrpcLogger) V(level int) bool {
	return level <= l.v
}

// verbosityLevel returns the verbosity level of the logger based on the
// GRPC_GO_LOG_VERBOSITY_LEVEL environment variable. The environment variable
// takes precedence over the given level.
func verbosityLevel(v int) (int, error) {
	value := os.Getenv("GRPC_GO_LOG_VERBOSITY_LEVEL")
	if value == "" {
		return v, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid gRPC log verbosity level %q: %w", value, err)
	}

	if parsed < 0 {
		return 0, fmt.Errorf("invalid gRPC log verbosity level %q: the level must not be negative", value)
	}

	return parsed, nil
}
