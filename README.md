# pkg

A collection of small, standalone Go packages.

Every package is a separate Go module with its own dependencies, so importing
one of them does not pull anything else in.

## Modules

| Module                     | Description                                                                                     |
|----------------------------|-------------------------------------------------------------------------------------------------|
| [closer](./closer)         | Collection of cleanup functions, run once — explicitly, or on a context cancel or an OS signal. |
| [ctxkey](./ctxkey)         | Unique, pointer-backed context keys.                                                            |
| [deferred](./deferred)     | Helpers for the `defer` calls.                                                                  |
| [dispatcher](./dispatcher) | Routing of keyed events to handlers, with priorities, propagation control and wildcards.        |
| [protomap](./protomap)     | Deserialization of protobuf messages by their descriptor name.                                  |
| [strcase](./strcase)       | Detection of the case of a string and conversion between the case styles.                       |
| [sync](./sync)             | A `sync.Once` that keeps retrying until the guarded function succeeds.                          |
| [time](./time)             | A duration type that is serialized as a human-readable string.                                  |

Each module has its own README with the description of the API and the
examples of the usage.

## Usage

The packages are imported by their module path:

```go
import (
    "github.com/nbgrp/pkg/closer"
    "github.com/nbgrp/pkg/strcase"
)
```

## License

[BSD 3-Clause](./LICENSE)
