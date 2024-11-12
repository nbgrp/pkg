package closer

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
)

type CloseFn func(ctx context.Context) error

// Closer allows to register functions to clean up resources.
type Closer interface {
	// SetContext sets context which will be passed into the close functions without cancel.
	SetContext(ctx context.Context)
	// Add adds close function.
	Add(CloseFn)
	// Done returns signal channel.
	Done() <-chan struct{}
	// Err returns joint error based on close functions errors.
	Err() error
	// CloseAll runs close functions in arbitrary order.
	CloseAll()
}

type closer struct {
	done  chan struct{}
	ctx   atomic.Pointer[context.Context]
	err   atomic.Value
	funcs atomic.Pointer[[]CloseFn]
	once  sync.Once
}

var global = New(context.Background())

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

// New returns base Closer implementation.
//
// If signals specified, then close functions will trigger when any arrives.
func New(ctx context.Context, signals ...os.Signal) *closer {
	c := &closer{done: make(chan struct{})}
	if ctx == nil {
		panic("cannot create closer with nil context")
	}
	c.ctx.Store(&ctx)

	go func() {
		if len(signals) > 0 {
			var cancel context.CancelFunc
			ctx, cancel = signal.NotifyContext(ctx, signals...)
			defer cancel()
		}

		<-ctx.Done()
		c.CloseAll() //nolint:contextcheck
	}()

	return c
}

func (c *closer) SetContext(ctx context.Context) {
	if ctx == nil {
		panic("cannot set nil context")
	}
	c.ctx.Store(&ctx)
}

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
		old := c.funcs.Load()
		if old != nil {
			funcs = make([]CloseFn, len(*old)+1)
			copy(funcs, *old)
			funcs[len(funcs)-1] = f
		}
	}
}

func (c *closer) Done() <-chan struct{} {
	return c.done
}

func (c *closer) Err() error {
	select {
	case <-c.done:
		err, _ := c.err.Load().(error)
		return err
	default:
		return nil
	}
}

func (c *closer) CloseAll() {
	c.once.Do(func() {
		defer close(c.done)

		ctx := context.WithoutCancel(*c.ctx.Load())
		funcs := *c.funcs.Swap(nil)

		errCh := make(chan error, len(funcs))
		for _, fn := range funcs {
			go func(fn CloseFn) {
				errCh <- fn(ctx)
			}(fn)
		}

		errs := make([]error, 0, len(funcs))
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
