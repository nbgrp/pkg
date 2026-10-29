# deferred

A tiny, dependency-free Go package with a single helper for closing resources
in a deferred call without losing the close error.

## Public API

```go
func Close(err *error, closer io.Closer)
```

`Close` closes the `closer` and joins the error it returns with the error
pointed to by `err`.

It is meant to be used with `defer`, right after the resource is opened:

```go
func process(path string) (err error) {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer deferred.Close(&err, f)

    return doSomething(f)
}
```

Without the helper the close error would have to be handled explicitly, and the
usual `defer f.Close()` swallows it:

```go
defer f.Close() // the error is dropped
```

## Semantics

- The closer is **always** closed, even if `err` is nil.
- If `err` is nil, the close error is discarded — the closer is still closed,
  but the result is thrown away.
- Otherwise the close error is joined with the current value of `*err` using
  `errors.Join`, so both errors are reported.

```go
deferred.Close(&err, closer)
// err == nil            -> err stays nil, the close error is discarded
// err == errInternal    -> err becomes the join of errInternal and the close error
```

Since the error is joined, the result is never equal to one of its parts. Use
`errors.Is` to check for a particular error:

```go
defer deferred.Close(&err, closer)
// ...
if errors.Is(err, os.ErrNotExist) {
    // ...
}
```

## Ordering

`defer` runs the deferred calls in reverse order, so the resources are closed in
the opposite order to the one in which they were opened. This is the behavior
you usually want, and the helper does not change it.

The helper is also safe to use with a closer that returns an error and a
function that returns one: both errors end up in the same joined error, and
neither of them is lost.
