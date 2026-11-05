# slogger

A `log/slog` logger with a configurable format, a hot-reloadable log level and a
custom writer.

## Public API

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
```

- `Leveler` — the current log level: `"DEBUG"`, `"INFO"`, `"WARN"` or `"ERROR"`
  (case-insensitive, with an optional offset, e.g. `"INFO+2"`).
- `Format` — `"json"` or `"text"`; any other value is an error.
- `LevelReloadCond` — the condition variable that notifies the logger about a level
  change. Without it the level is set once, at startup.
- `Writer` — the destination of the records; `os.Stdout` by default.

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
