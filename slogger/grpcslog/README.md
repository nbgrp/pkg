# grpcslog

A [`grpclog.LoggerV2`](https://pkg.go.dev/google.golang.org/grpc/grpclog) implementation
that writes the gRPC messages to a [`log/slog`](https://pkg.go.dev/log/slog) logger.

The logger is installed as the global gRPC logger, so all the messages of gRPC
go to the same place as the rest of the application logs:

```go
logger, err := grpcslog.NewGrpcLogger(ctx)
if err != nil {
    return err
}

grpclog.SetLoggerV2(logger)
```

## Public API

```go
func NewGrpcLogger(ctx context.Context) (*GrpcLogger, error)
func NewGrpcLoggerWithVerbosity(ctx context.Context, v int) (*GrpcLogger, error)
```

Both constructors return an error if the environment variables contain invalid
values.

## Configuration

The logger is configured with the same environment variables as the default
gRPC logger:

| Variable                    | Values                     | Default |
|-----------------------------|----------------------------|---------|
| `GRPC_GO_LOG_SEVERITY_LEVEL` | `INFO`, `WARNING`, `ERROR` | `ERROR` |
| `GRPC_GO_LOG_VERBOSITY_LEVEL` | a non-negative integer    | `0`     |
| `GRPC_GO_LOG_FORMATTER`      | `json`, `text`             | `text`  |

- `GRPC_GO_LOG_SEVERITY_LEVEL` — the severity level of the logged messages. The
  messages below the level are dropped.
- `GRPC_GO_LOG_VERBOSITY_LEVEL` — the verbosity level of the logger. It sets
  the threshold of the `V` method, so the verbose messages of gRPC are logged
  only if their level is not greater than the verbosity level of the logger.
  The variable takes precedence over the level passed to
  `NewGrpcLoggerWithVerbosity`.
- `GRPC_GO_LOG_FORMATTER` — the format of the logged messages: `json` or
  `text`. The value is case-insensitive.

The messages are written to `os.Stdout`, like the messages of the
[`slogger`](../) package.

## Verbosity

The verbosity level of gRPC is a threshold: the higher it is, the more verbose
messages are logged. The `V` method reports whether a message of the given
verbose level should be logged:

```go
logger, err := grpcslog.NewGrpcLoggerWithVerbosity(ctx, 2)
if err != nil {
    return err
}

logger.V(2) // true
logger.V(3) // false
```

The verbose messages of gRPC are written with the info severity, so they are
only logged if the severity level allows the info records.

## Errors

The constructors return an error if the severity level is unknown or if the
verbosity level is not a non-negative integer:

```
unknown gRPC log severity level "bogus"
invalid gRPC log verbosity level "bogus": strconv.Atoi: parsing "bogus": invalid syntax
```
