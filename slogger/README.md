# slogger

A `log/slog` logger with a configurable format, a hot-reloadable log level and a
custom writer.

The module is split into two packages:

- [`github.com/nbgrp/pkg/slogger`](./slogger.go) — the logger constructor.
- [`github.com/nbgrp/pkg/slogger/ctxlog`](./ctxlog/ctxlog.go) — the helpers that
  pass the logger and its attributes through the
  [context](https://pkg.go.dev/context).

There is also a separate module next to it:

- [`github.com/nbgrp/pkg/slogger/grpcslog`](./grpcslog/) — the
  [`grpclog.LoggerV2`](https://pkg.go.dev/google.golang.org/grpc/grpclog)
  implementation that writes the gRPC messages to the logger.

## Public API

The `slogger` package provides:

```go
type Options struct {
    // Leveler returns the current log level.
    Leveler func() string
    // LevelReloadCond is the condition variable that notifies the logger about
    // a log level change.
    LevelReloadCond *sync.Cond
    // Writer is an optional writer the logs are written to.
    // If it is not set, the logs are written to os.Stdout.
    Writer io.Writer
    // Format is the log output format. Allowed values: "json" and "text".
    Format string
}

func New(ctx context.Context, opts Options) (*slog.Logger, error)
func NewToContext(ctx context.Context, opts Options) (context.Context, error)
```

- `Leveler` — the current log level: `"DEBUG"`, `"INFO"`, `"WARN"` or `"ERROR"`
  (case-insensitive, with an optional offset, e.g. `"INFO+2"`).
- `Format` — `"json"` or `"text"`; any other value is an error.
- `LevelReloadCond` — the condition variable that notifies the logger about a level
  change. Without it the level is set once, at startup.
- `Writer` — the destination of the records; `os.Stdout` by default.

`NewToContext` is the same as `New`, but it puts the logger into the returned
context, so it can be used with the [`ctxlog`](#context-logger) helpers. The
returned context is nil if the logger cannot be initialized.

## Usage

```go
logger, err := slogger.New(ctx, slogger.Options{
    Leveler: func() string { return "INFO" },
    Format:  "json",
})
if err != nil {
    return err
}

logger.Info("Server started", slog.String("addr", ":8080"))
// {"time":"...","level":"INFO","msg":"Server started","addr":":8080"}
```

A custom writer replaces `os.Stdout`:

```go
file, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
if err != nil {
    return err
}
defer file.Close()

logger, err := slogger.New(ctx, slogger.Options{
    Leveler: func() string { return "INFO" },
    Format:  "text",
    Writer:  file,
})
```

The level is applied by the handler through a `slog.LevelVar`: the records below it
are dropped before they are formatted.

## Log level reload

With a `*sync.Cond` in `LevelReloadCond` the logger re-reads the level on every
notification:

```go
var (
    levelMu sync.Mutex
    level   = "INFO"

    cond = sync.NewCond(&sync.Mutex{})
)

func setLevel(l string) {
    levelMu.Lock()
    level = l
    levelMu.Unlock()

    cond.L.Lock()
    cond.Broadcast()
    cond.L.Unlock()
}

logger, err := slogger.New(ctx, slogger.Options{
    Leveler: func() string {
        levelMu.Lock()
        defer levelMu.Unlock()

        return level
    },
    Format:          "text",
    LevelReloadCond: cond,
})
```

Two rules:

- the level is changed before the notification is sent, and the notification is sent
  under the lock of the condition variable: the logger reads the level and waits
  atomically under that lock, so a notification sent outside it can be missed;
- the `Leveler` is called while the logger holds the lock of the condition variable,
  so it must not acquire that lock itself. Use a separate mutex for the level, as
  above.

The logger stops watching the level when the context passed to `New` is canceled.

On a level change the logger writes a notification record:

```
time=... level=INFO msg="Log level changed from INFO to DEBUG"
```

It is written directly through the handler, bypassing the level check, so it is
written even when the new level is `WARN` or `ERROR`.

If the level cannot be parsed, the error is written as an `ERROR` record and the
current level is kept:

```
time=... level=ERROR msg="Failed to parse log level while hot-reload update" error="slog: level string \"bogus\": unknown name"
```

## Errors

`New` returns an error if the level returned by `Leveler` cannot be parsed
(`slog: level string "bogus": unknown name`) or if the format is unknown
(`unknown log format "xml"`).

## Concurrency

The logger is safe for concurrent use, as any `slog.Logger` is. The `Leveler` is
called from the watcher goroutine, so it must be safe for concurrent use as well.

## Context logger

The `ctxlog` package passes the logger through the
[context](https://pkg.go.dev/context): the logger is put into the context once,
and the package-level functions take it from there. The code that writes the log
stays independent of the way the logger is built and of the attributes of the
current request.

### Public API

```go
func ToContext(ctx context.Context, logger *slog.Logger) context.Context
func FromContext(ctx context.Context) *slog.Logger
func Debug(ctx context.Context, msg string, args ...any)
func Info(ctx context.Context, msg string, args ...any)
func Warn(ctx context.Context, msg string, args ...any)
func Error(ctx context.Context, msg string, args ...any)
func With(ctx context.Context, args ...any) context.Context
func Log(ctx context.Context, level slog.Level, msg string, args ...any)
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
```

- `ToContext` puts the logger into the context, `FromContext` takes it from
  there. If the context has no logger, `slog.Default()` is used.
- `With` returns a derived context with the given attributes bound to it: they
  are added to every record written with that context. Without the arguments it
  returns the context unchanged.
- `Log` and `LogAttrs` write a record with an arbitrary level; `Debug`, `Info`,
  `Warn` and `Error` are the shortcuts for `slog.LevelDebug`, `slog.LevelInfo`,
  `slog.LevelWarn` and `slog.LevelError`.

### Usage

```go
ctx, err := slogger.NewToContext(ctx, slogger.Options{
    Leveler: func() string { return "INFO" },
    Format:  "json",
})
if err != nil {
    return err
}

ctxlog.Info(ctx, "Server started", slog.String("addr", ":8080"))
```

The logger can also be put into the context explicitly:

```go
logger, err := slogger.New(ctx, slogger.Options{
    Leveler: func() string { return "INFO" },
    Format:  "json",
})
if err != nil {
    return err
}

ctx = ctxlog.ToContext(ctx, logger)
```

The arguments are passed to `slog` as is, so all the forms the standard library
understands are supported:

```go
ctxlog.Info(ctx, "Server started", "addr", ":8080")              // a key-value pair
ctxlog.Info(ctx, "Server started", slog.String("addr", ":8080")) // a slog.Attr
```

The attributes common to all the records of the scope are bound to the context:

```go
ctx = ctxlog.With(ctx, "request_id", requestID)

ctxlog.Info(ctx, "Started")
```

The context passed to the logging call is passed to the handler as well, so a
custom handler can take the values from it.
