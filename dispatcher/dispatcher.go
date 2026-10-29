// SPDX-License-Identifier: BSD-3-Clause

package dispatcher

import (
	"context"
)

// StopPropagationError is returned by a handler to stop the dispatch chain.
//
// The handlers that are not called yet are skipped. The Inner error, if it is
// not nil, is returned by the dispatch instead of the StopPropagationError
// itself.
type StopPropagationError struct {
	// Inner is the error the dispatch reports. It may be nil.
	Inner error
}

// Error returns the message of the error: "stop propagation", followed by the
// message of the Inner error, if there is one.
func (e *StopPropagationError) Error() string {
	msg := "stop propagation"
	if e.Inner != nil {
		msg += ": " + e.Inner.Error()
	}
	return msg
}

// Handler is a callback invoked on a matching dispatch.
//
// The payload is the one passed to the dispatch call.
type Handler func(context.Context, ...any) error

// Dispatcher routes the payload to the handlers registered for the key.
type Dispatcher interface {
	// Dispatch calls the handlers registered for the key and returns their joined errors.
	Dispatch(ctx context.Context, key string, payload ...any) error
}

// Listener registers the handlers for a key pattern.
type Listener interface {
	// Listen registers the handler for the key pattern and returns the function
	// that cancels the registration.
	Listen(keyPattern string, handler Handler) (cancel func(), err error)
}

// PriorityListener registers the handlers for a key pattern, with a priority.
type PriorityListener interface {
	// ListenWithPriority registers the handler for the key pattern with the given
	// priority and returns the function that cancels the registration.
	ListenWithPriority(keyPattern string, handler Handler, priority int) (cancel func(), err error)
}
