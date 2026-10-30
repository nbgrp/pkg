# ctxkey

A tiny, dependency-free Go package for creating unique context keys.

`context.WithValue` accepts any comparable value as a key, but the [official
advice](https://go.dev/blog/context#Customizing-Context) is to define a dedicated
type for every key, so that keys from different packages cannot collide. This
package does that in one line, without the boilerplate of declaring a type per
key.

## Public API

```go
type Key interface {
    fmt.Stringer
}

func New(name string) Key
```

- `New` returns a new key with the given name.
- `Key` is an opaque interface: the only thing a caller can do with a key is
  compare it with another key or read its string representation.

## Usage

```go
import (
    "context"

    "github.com/nbgrp/pkg/ctxkey"
)

var userIDKey = ctxkey.New("user_id")

func WithUserID(ctx context.Context, id int64) context.Context {
    return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) (int64, bool) {
    id, ok := ctx.Value(userIDKey).(int64)
    return id, ok
}
```

The key is usually declared once, next to the accessor that uses it, and is then
passed around as an ordinary value.

## Uniqueness

Every call to `New` returns a distinct key. Two keys are equal only if they are
the same pointer, so a key created by `New("id")` in one package never matches
a key created by `New("id")` in another one — and it never matches a string
`"id"` either.

```go
a := ctxkey.New("id")
b := ctxkey.New("id")

fmt.Println(a == b)            // false
ctx := context.WithValue(context.Background(), a, 1)
fmt.Println(ctx.Value(b))      // nil
fmt.Println(ctx.Value("id"))   // nil
```

The name is only used for the string representation, so it does not have to be
unique.

## String representation

A key prints itself as `ctxkey "<name>"`, which makes it recognizable in logs
and in `%+v` formatting of wrapped errors.

```go
key := ctxkey.New("user_id")
fmt.Println(key) // ctxkey "user_id"
```

The `Key` interface embeds `fmt.Stringer`, so a key can be passed to any
formatting function directly.

## Allocations

A key is backed by a pointer to a private struct, so it fits into an `any`
without an extra allocation for the interface conversion:

```go
var myKey = ctxkey.New("MyKey")
// 2 allocations: the context struct and the value
context.WithValue(ctx, myKey, someValue)
```

The common alternative — declaring a named integer type for the key — allocates
one extra time, because the key itself has to be boxed into an `any`:

```go
type key int
var myKey key = 0
// 3 allocations: the context struct, the key and the value
context.WithValue(ctx, myKey, someValue)
```
