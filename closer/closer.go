// SPDX-License-Identifier: BSD-3-Clause

package closer

import (
	"context"
	"errors"
	"os/signal"
	"sync"
	"sync/atomic"
)

// CloseFn is a cleanup function registered in a Closer.
//
// The context passed to it carries the values of the context set on the
// Closer, but is detached from its cancellation, so that the cleanup is able
// to finish its work even after the context has been canceled.
type CloseFn func(ctx context.Context) error

// Closer cleans up resources.
type Closer interface {
	Adder
	// SetContext sets context which will be passed into the close functions without cancel.
	SetContext(ctx context.Context)
	// Done returns signal channel.
	Done() <-chan struct{}
	// Err returns a joint error based on close functions errors.
	Err() error
	// CloseAll runs close functions in arbitrary order.
	CloseAll()
}

// Adder allows registering functions to clean up resources.
type Adder interface {
	// Add adds close function.
	Add(CloseFn)
}

type closer struct {
	done  chan struct{}
	ctx   atomic.Pointer[context.Context]
	err   atomic.Value
	funcs atomic.Pointer[[]CloseFn]
	once  sync.Once
}

// global is the Closer used by the package-level functions.
var _, global = New(context.Background())

// SetContext sets context into the global Closer.
func SetContext(ctx context.Context) {
	global.SetContext(ctx)
}

// Add adds close function into the global Closer.
func Add(f CloseFn) {
	global.Add(f)
}

// Done returns signal channel of the global Closer.
func Done() <-chan struct{} {
	return global.Done()
}

// Err returns joint error based on close functions errors of the global Closer.
func Err() error {
	return global.Err()
}

// CloseAll runs close functions of the global Closer.
func CloseAll() {
	global.CloseAll()
}

// New returns a Closer implementation together with the context it is bound to.
//
// The returned context is the one passed in, wrapped into a signal-notifiable
// context when [WithSignals] is used. When [WithContextCancel] or
// [WithSignals] is used, the closer is closed automatically as soon as that
// context is done.
//
// New panics if ctx is nil.
func New(ctx context.Context, opts ...Option) (context.Context, *closer) {
	o := options{}
	for _, opt := range opts {
		o = opt(o)
	}

	var cancel context.CancelFunc
	if len(o.signals) > 0 {
		ctx, cancel = signal.NotifyContext(ctx, o.signals...)
	}

	c := &closer{done: make(chan struct{})}
	if ctx == nil {
		panic("cannot create closer with nil context")
	}
	c.ctx.Store(&ctx)

	if o.ctxCancel {
		go func() {
			if cancel != nil {
				defer cancel()
			}

			<-ctx.Done()
			c.CloseAll() //nolint:contextcheck
		}()
	}

	return ctx, c
}

// SetContext sets the context passed to the close functions.
func (c *closer) SetContext(ctx context.Context) {
	if ctx == nil {
		panic("cannot set nil context")
	}
	c.ctx.Store(&ctx)
}

// Add registers a close function.
//
// The function is called exactly once, when the closer is closed. Registering
// a function in an already closed closer has no effect.
func (c *closer) Add(f CloseFn) {
	var funcs []CloseFn
	old := c.funcs.Load()
	if old != nil {
		funcs = make([]CloseFn, len(*old)+1)
		copy(funcs, *old)
		funcs[len(funcs)-1] = f
	} else {
		funcs = []CloseFn{f}
	}

	for !c.funcs.CompareAndSwap(old, &funcs) {
		old = c.funcs.Load()
		if old != nil {
			funcs = make([]CloseFn, len(*old)+1)
			copy(funcs, *old)
			funcs[len(funcs)-1] = f
		}
	}
}

// Done returns a channel that is closed after all close functions have finished.
func (c *closer) Done() <-chan struct{} {
	return c.done
}

// Err returns the joined error of the close functions.
//
// It returns nil until the closer is closed.
func (c *closer) Err() error {
	select {
	case <-c.done:
		err, _ := c.err.Load().(error)
		return err
	default:
		return nil
	}
}

// CloseAll runs all registered close functions concurrently and closes the done
// channel.
//
// It is safe to call it multiple times: only the first call has an effect. The
// close functions receive a context detached from the cancellation of the
// context set on the closer, so that they are able to complete their work.
func (c *closer) CloseAll() { //nolint:gocognit
	c.once.Do(func() {
		defer close(c.done)

		ctx := context.WithoutCancel(*c.ctx.Load())
		funcs := c.funcs.Swap(nil)
		if funcs == nil {
			return
		}

		errCh := make(chan error, len(*funcs))
		for _, fn := range *funcs {
			go func(fn CloseFn) {
				errCh <- fn(ctx)
			}(fn)
		}

		errs := make([]error, 0, len(*funcs))
		for i := 0; i < cap(errs); i++ {
			if err := <-errCh; err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			c.err.Store(errors.Join(errs...))
		}
	})
}
