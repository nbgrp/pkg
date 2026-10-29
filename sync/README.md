# sync

A small, dependency-free Go package with a `sync.Once` that keeps retrying until
the guarded function succeeds.

## Public API

```go
type SuccessfulOnce struct {
    // contains unexported fields
}

func (o *SuccessfulOnce) Do(fn func() error) error
```

`SuccessfulOnce` is a zero-value ready value: it needs no initialization and
must not be copied after the first use.

## Usage

```go
import (
    pkgSync "github.com/nbgrp/pkg/sync"
)

var once pkgSync.SuccessfulOnce

err := once.Do(func() error {
    return connect() // retried until it succeeds
})
```

The package is named `sync`, just like the standard library one, so it usually
has to be imported under an alias:

```go
import (
    stdsync "sync"

    pkgSync "github.com/nbgrp/pkg/sync"
)
```

## Semantics

`Do` calls `fn` until it returns a nil error:

- while `fn` fails, its error is returned to the caller and `fn` is called
  again by the next `Do` call;
- the first successful call is remembered, and every `Do` call made after it
  returns nil without calling `fn` again.

```go
var once sync.SuccessfulOnce

// the first two calls fail, the third one succeeds
err := once.Do(func() error { return errors.New("boom") }) // boom
err = once.Do(func() error { return errors.New("boom") })  // boom
err = once.Do(func() error { return nil })                // nil
err = once.Do(func() error { return errors.New("boom") }) // nil, fn is not called
```

The function is only ever called with the arguments of the caller that won the
race to run it — it takes no arguments, so the state it needs is usually closed
over.

## Comparison with `sync.Once`

|                         | `sync.Once` | `SuccessfulOnce`      |
|-------------------------|-------------|-----------------------|
| The function may fail   | no          | yes                   |
| Retries after a failure | no          | yes, on the next call |
| Returns an error        | no          | yes                   |
| The function is called  | exactly once| until it succeeds     |

`SuccessfulOnce` is the right choice when the guarded operation is idempotent
but may fail for transient reasons — connecting to a service, acquiring a lock
through an external system, initializing a resource.

## Concurrency

`Do` is safe for concurrent use.

The calls are serialized: only one of them runs `fn` at a time, and the others
wait for it. If `fn` fails, the next waiter runs it in turn, so a failing
function is retried by the concurrent callers as well — one attempt per caller.

The success flag is read without the lock on the fast path, so a `Do` call made
after a success returns immediately, without blocking.
