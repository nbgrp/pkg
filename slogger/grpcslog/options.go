// SPDX-License-Identifier: BSD-3-Clause

package grpcslog

type Option func(*GrpcLogger)

// WithVerbosity sets the verbosity level of the logger: the threshold of the
// [GrpcLogger.V] method.
//
// The GRPC_GO_LOG_VERBOSITY_LEVEL environment variable takes precedence over
// the given level.
func WithVerbosity(v int) Option {
	return func(l *GrpcLogger) {
		l.v = v
	}
}
