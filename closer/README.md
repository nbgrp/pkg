# closer

A small, dependency-free Go package for collecting cleanup functions and running
them exactly once — either explicitly, or automatically when a context is canceled
or an OS signal arrives.

The package has two layers:

- a set of package-level functions operating on a global closer, useful for
  "process-wide" cleanup;
- the [`Closer`](./closer.go) interface and its implementation, useful when the
  cleanup scope is explicit and must be bound to a particular context.

## Public API

### Types

```go
type CloseFn func(ctx context.Context) error

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

type Adder interface {
    // Add adds close function.
    Add(CloseFn)
}
```

- `CloseFn` is a cleanup function. It receives a context, but not the one the
  closer was bound to — the cancellation is stripped away (see
  [The close context](#the-close-context)).
- `Adder` is the registration half of `Closer`: it only knows how to register
  a close function.
- `Closer` adds the lifecycle to it: setting the close context, waiting for the
  cleanup to finish (`Done`), reading the result (`Err`) and triggering it
  (`CloseAll`).

### The global closer

```go
func SetContext(ctx context.Context)
func Add(f CloseFn)
func Done() <-chan struct{}
func Err() error
func CloseAll()
```

These operate on a package-level closer that is created on initialization. They
are handy for libraries that need to register a cleanup without owning the
application's lifecycle: the application calls `closer.CloseAll()` on shutdown
and inspects `closer.Err()` afterwards.

The global closer is created with `context.Background()`, so
`closer.SetContext(...)` is the way to give the close functions access to the
application context.

### Creating a closer

```go
func New(ctx context.Context, opts ...Option) (context.Context, *closer)
```

`New` returns the closer together with the context it is bound to. The context
is returned because it may be replaced by `New` itself: with `WithSignals` the
returned context is a signal-notifiable one, and it is this returned value —
not the one passed in — that has to be used by the application.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

ctx, cl := closer.New(ctx, closer.WithContextCancel())
defer cl.CloseAll()
```

`New` panics if `ctx` is nil.

## Options

```go
func WithContextCancel() Option
func WithSignals(signals ...os.Signal) Option
```

- `WithContextCancel` closes the closer automatically as soon as the context
  passed to `New` is done.
- `WithSignals` does the same, but the context is first wrapped into a
  signal-notifiable one, so that the closer is closed when any of the given
  signals arrives. It implies `WithContextCancel`.

```go
ctx, cl := closer.New(context.Background(), closer.WithSignals(syscall.SIGINT, syscall.SIGTERM))

cl.Add(func(ctx context.Context) error {
    fmt.Println("shutting down")
    return nil
})

<-ctx.Done() // the context is canceled on SIGINT/SIGTERM
<-cl.Done() // ... and the closer is closed right after that
```

## Closing

`CloseAll` runs every registered close function and then closes the `Done`
channel. It is idempotent: only the first call has an effect, subsequent calls
return immediately.

The close functions run **concurrently**, in an arbitrary order, and each of them
gets its own goroutine. There is no ordering guarantee between them, so a close
function must not depend on another one having finished.

`Done` is closed after all close functions have returned, which makes it the
right place to wait for the cleanup to complete:

```go
cl.CloseAll()
<-cl.Done()

if err := cl.Err(); err != nil {
    // at least one close function failed
}
```

## Errors

`Err` returns the `errors.Join` of all errors returned by the close functions.
It returns `nil` while the closer is not closed yet, and `nil` after it has been
closed if every close function succeeded.

Errors are collected, not short-circuited: a failing close function does not
prevent the others from running.

## The close context

Close functions receive a context that carries the **values** of the context
set on the closer, but is **not** canceled when that context is canceled
(the implementation uses `context.WithoutCancel`).

This is what makes the automatic shutdown work: with `WithContextCancel` the
closer is closed exactly because the context is done, and passing that canceled
context to the cleanup functions would make them abort immediately.

The context can be replaced at any time with `SetContext`:

```go
cl.SetContext(context.WithValue(ctx, key, "value"))
```

The closer reads the context at the moment it is closed, so the last value set
before `CloseAll` is the one the close functions receive.

## Concurrency

`Add`, `SetContext`, `Done`, `Err` and `CloseAll` are safe for concurrent use.

`Add` is lock-free: the functions are stored in an atomically swapped slice, so
registering a cleanup never blocks the caller. Registering a function after the
closer has been closed has no effect — the function is never called.

`CloseAll` runs the close functions concurrently and waits for all of them
before closing the `Done` channel.
