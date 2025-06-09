// SPDX-License-Identifier: BSD-3-Clause

package closer

import (
	"os"
)

type options struct {
	signals   []os.Signal
	ctxCancel bool
}

type Option func(*options)

// WithContextCancel allows to call closer CloseAll on context cancel implicitly.
func WithContextCancel() Option {
	return func(o *options) {
		o.ctxCancel = true
	}
}

// WithSignals will trigger creation of signal notifiable context.
// The closer CloseAll will be called implicitly when any of the specified signals arrives.
func WithSignals(signals ...os.Signal) Option {
	return func(o *options) {
		o.ctxCancel = true
		if o.signals == nil {
			o.signals = signals
			return
		}
		o.signals = append(o.signals, signals...)
	}
}
