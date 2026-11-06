# grpcslog

A [`grpclog.LoggerV2`](https://pkg.go.dev/google.golang.org/grpc/grpclog) implementation
that writes the gRPC messages to a [`log/slog`](https://pkg.go.dev/log/slog) logger.

The logger is installed as the global gRPC logger, so all the messages of gRPC
go to the same place as the rest of the application logs:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

grpcLogger, err := grpcslog.New(logger)
if err != nil {
    return err
}

grpclog.SetLoggerV2(grpcLogger)
```

## Public API

```go
func New(logger *slog.Logger, opts ...Option) (*GrpcLogger, error)

func WithVerbosity(v int) Option
```

`New` returns an error if the `GRPC_GO_LOG_VERBOSITY_LEVEL` environment variable
contains an invalid value.

### WithVerbosity

Sets the verbosity level of the logger: the threshold of the `V` method.

## Configuration

The level and the format of the messages are set in the logger passed to `New`,
so only the verbosity level is configured by the package itself:

| Variable                    | Values                  | Default |
|-----------------------------|-------------------------|---------|
| `GRPC_GO_LOG_VERBOSITY_LEVEL` | a non-negative integer | `0`     |

- `GRPC_GO_LOG_VERBOSITY_LEVEL` — the verbosity level of the logger. It sets
  the threshold of the `V` method, so the verbose messages of gRPC are logged
  only if their level is not greater than the verbosity level of the logger.
  The variable takes precedence over the level passed to `WithVerbosity`.

**`GRPC_GO_LOG_SEVERITY_LEVEL` and `GRPC_GO_LOG_FORMATTER` are not applied.**
They are read by the default gRPC logger only: the level and the format of the
records are defined by the logger, so the environment variables have no effect
on them.

## Verbosity

The verbosity level of gRPC is a threshold: the higher it is, the more verbose
messages are logged. The `V` method reports whether a message of the given
verbose level should be logged:

```go
grpcLogger, err := grpcslog.New(logger, grpcslog.WithVerbosity(2))
if err != nil {
    return err
}

grpcLogger.V(2) // true
grpcLogger.V(3) // false
```

The verbose messages of gRPC are written with the info severity, so they are
only logged if the level of the logger allows the info records.

## Errors

The constructor returns an error if the verbosity level is not a non-negative
integer:

```
invalid gRPC log verbosity level "bogus": strconv.Atoi: parsing "bogus": invalid syntax
```
