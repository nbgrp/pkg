// SPDX-License-Identifier: BSD-3-Clause

package dispatcher

import (
	"context"
)

type StopPropagationError struct {
	Inner error
}

func (e *StopPropagationError) Error() string {
	msg := "stop propagation"
	if e.Inner != nil {
		msg += ": " + e.Inner.Error()
	}
	return msg
}

type Handler func(context.Context, ...any) error

type Dispatcher interface {
	Dispatch(ctx context.Context, key string, payload ...any) error
}

type Listener interface {
	Listen(keyPattern string, handler Handler) (cancel func(), err error)
}

type PriorityListener interface {
	ListenWithPriority(keyPattern string, handler Handler, priority int) (cancel func(), err error)
}
