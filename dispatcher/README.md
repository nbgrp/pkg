# dispatcher

A small, dependency-free Go package for routing keyed events to one or more handlers, with
optional priorities, propagation control, and wildcard matching.

The package is split into a thin contract layer and backend implementation:

- [`github.com/nbgrp/pkg/dispatcher`](./dispatcher.go) — public types and interfaces.
- [`github.com/nbgrp/pkg/dispatcher/trie`](./trie/trie.go) — the current implementation, a
  prefix-tree (trie) keyed by the event path.

## Public API

The root package defines the contracts every backend implements.

### Types

```go
type Handler func(ctx context.Context, payload ...any) error

type Dispatcher interface {
    Dispatch(ctx context.Context, key string, payload ...any) error
}

type Listener interface {
    Listen(keyPattern string, handler Handler) (cancel func(), err error)
}

type PriorityListener interface {
    ListenWithPriority(keyPattern string, handler Handler, priority int) (cancel func(), err error)
}
```

- `Handler` is the user-supplied callback invoked for a matching `Dispatch`.
- `Dispatcher.Dispatch` runs all handlers registered for `key` and returns their joined errors. 
  `payload` is forwarded to every handler.
- `Listener.Listen` and `PriorityListener.ListenWithPriority` register a handler on `key`
  and return a `cancel` function that removes the registration. A backend must return
  an error for invalid keys (empty, leading/trailing/duplicated separators) and a `nil`
  handler.

A concrete dispatcher typically satisfies `Listener` and `PriorityListener` in addition
to `Dispatcher`; users cast or interface-assert to whichever subset they need.

### Stopping the chain

```go
type StopPropagationError struct {
    Inner error
}
```

A handler may return a `*StopPropagationError` to short-circuit dispatch and (optionally)
propagate an inner error. See each backend's documentation for the exact semantics — in
particular, `ModePriority` and `ModeConcurrent` interpret it differently.

## Implementations

### `trie`

[`github.com/nbgrp/pkg/dispatcher/trie`](./trie/trie.go) is a prefix-tree implementation
that supports wildcards and two execution modes (`Priority` and `Concurrent`).

#### Quickstart

```go
import (
    "context"
    "fmt"

    "github.com/nbgrp/pkg/dispatcher/trie"
)

func main() {
    d, err := trie.NewDispatcher() // defaults: ModePriority, '.' separator, '*' wildcard
    if err != nil {
        panic(err)
    }

    cancel, err := d.Listen("user.created", func(_ context.Context, payload ...any) error {
        fmt.Printf("user.created: %v\n", payload)
        return nil
    })
    if err != nil {
        panic(err)
    }
    defer cancel()

    _ = d.Dispatch(context.Background(), "user.created", 42, "alice")
    // Output: user.created: [42 alice]
}
```

Keys are split by the configured separator (default `.`) and matched segment by segment.
Handlers are registered on concrete keys; use the wildcard segment `*` to subscribe to
"any value at this position" (see [Wildcards](#wildcards-trie)).

#### Modes

The dispatcher can run handlers in one of two modes. The mode is selected at construction
time and cannot be changed afterwards.

##### `trie.ModePriority` (default)

Handlers are sorted in **descending** order of their `priority` and called sequentially.
Handlers registered with the same priority keep their registration order (sort is stable).
Returning a `*dispatcher.StopPropagationError` from a handler stops the chain: handlers
with lower priority are **not** called. If `StopPropagationError.Inner` is non-nil, it is
returned by `Dispatch`; otherwise `Dispatch` returns `nil`.

```go
d, _ := trie.NewDispatcher(trie.WithMode(trie.ModePriority))

_, _ = d.ListenWithPriority("evt", high, 10)
_, _ = d.Listen("evt", mid) // priority 0
_, _ = d.ListenWithPriority("evt", low, -5)

// Dispatch order: high, mid, low
```

##### `trie.ModeConcurrent`

All matching handlers are launched concurrently via `sync.WaitGroup.Go` and the
dispatcher waits for every one of them to return. The dispatch error is the `errors.Join`
of every handler's return value.

`StopPropagationError` has **no interrupting effect** in this mode: every matching
handler has already been scheduled by the time one of them returns it. It is still
treated specially for error reporting: with a non-nil `Inner` the inner error is joined
into the result; without an `Inner` the stop-propagation signal is dropped from the
joined error (i.e. it is treated as success).

```go
d, _ := trie.NewDispatcher(trie.WithMode(trie.ModeConcurrent))

_, _ = d.Listen("evt.a", h1)
_, _ = d.Listen("evt.a", h2)
_, _ = d.Listen("evt.b", h3)

_ = d.Dispatch(ctx, "evt.a") // h1 and h2 run in parallel
_ = d.Dispatch(ctx, "evt.b") // h3 runs alone
```

#### Priorities and cancellation

`ListenWithPriority(key, handler, priority)` accepts an `int` priority — the higher the
number, the earlier the handler runs in `ModePriority`. Priorities are ignored in
`ModeConcurrent`.

`Listen` and `ListenWithPriority` both return a `cancel` function. Calling it marks the
registration as deleted. The handler is not removed from the trie eagerly; instead the
dispatcher filters out deleted handlers on every `Dispatch` and lazily prunes them from
the trie on the next match. This makes `cancel` safe to call concurrently with `Dispatch`.

```go
cancel, err := d.Listen("evt", handler)
if err != nil { /* invalid key, nil handler, ... */ }

cancel() // handler will not be called for any future Dispatch("evt", ...)
```

##### Valid keys

The constructor options `WithKeySeparator` and `WithWildcardMark` default to `.` and `*`
respectively. The following keys are rejected with an error:

- empty string;
- starting or ending with the separator (e.g. `.a`, `a.`);
- containing consecutive separators (e.g. `a..b`);
- a `nil` handler.

#### <a id="wildcards-trie"></a>Wildcards

A key segment equal to the wildcard mark (default `*`) matches **any single segment** of
the dispatched key. A registered handler is invoked when every segment of its key —
wildcards and concrete segments alike — matches the dispatched key.

The wildcard can appear in any position: leading (`*.created`), middle
(`user.*.updated`), or trailing (`user.deleted.*`). Concrete and wildcard handlers on
overlapping paths are both invoked.

```go
d, _ := trie.NewDispatcher()

_, _ = d.Listen("user.*.updated", onUserUpdated)
_, _ = d.Listen("*.created",      onAnythingCreated)
_, _ = d.Listen("user.deleted",   onUserDeleted)

_ = d.Dispatch(ctx, "user.alice.updated") // onUserUpdated
_ = d.Dispatch(ctx, "order.created")      // onAnythingCreated
_ = d.Dispatch(ctx, "user.deleted")       // onUserDeleted
```

In `ModePriority`, handlers contributed by a wildcard and a concrete match on the same
dispatched key are interleaved by their priority. Handlers reached through wildcards are
added to the candidate set in the order the trie walk encounters them (wildcard child is
checked before concrete child at each node), and `slices.SortStableFunc` then reorders by
priority while preserving the relative order of equal-priority handlers.

#### Constructor options

```go
d, err := trie.NewDispatcher(
    trie.WithMode(trie.ModeConcurrent), // default: trie.ModePriority
    trie.WithKeySeparator('/'),         // default: '.'
    trie.WithWildcardMark('#'),         // default: '*'
)
```

The three option functions are the only configuration knobs: mode, key separator, and
wildcard mark. Anything else is fixed at construction time.

#### Errors

`Dispatch` returns `errors.Join(errs...)` of every matched handler's return value (after
unwrapping `StopPropagationError` per the rules above). It returns `nil` only when every
handler returned `nil` (or a `StopPropagationError` without an `Inner`).

A handler may short-circuit the chain in `ModePriority` by returning:

```go
return &dispatcher.StopPropagationError{}                 // stop, no error
return &dispatcher.StopPropagationError{Inner: someError} // stop, propagate error
```

#### Concurrency

`Listen`, `ListenWithPriority`, `Dispatch`, and `cancel` are safe for concurrent use. The
implementation lazily prunes cancelled handlers during dispatch, so the cost of a `cancel`
is constant and independent of trie size.
