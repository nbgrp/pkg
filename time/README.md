# time

A tiny, dependency-free Go package with a duration type that is serialized as a
human-readable string instead of a number of nanoseconds.

`time.Duration` is a plain `int64`, so it is marshaled to JSON as a bare
number — `1800000000000` instead of `30m0s`. This package provides a type that
keeps the readable form in both JSON and YAML.

## Public API

```go
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error)
func (d *Duration) UnmarshalJSON(b []byte) error
func (d Duration) MarshalYAML() (interface{}, error)
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error
```

`Duration` implements `json.Marshaler`, `json.Unmarshaler` and the YAML
marshaler and unmarshaler interfaces, so it works with `encoding/json` and with
`gopkg.in/yaml.v3` out of the box.

## Usage

```go
import (
    "encoding/json"
    "fmt"

    pkgTime "github.com/nbgrp/pkg/time"
)

type Config struct {
    Timeout pkgTime.Duration `json:"timeout"`
}

data, _ := json.Marshal(Config{Timeout: pkgTime.Duration(30 * time.Second)})
// data == {"timeout":"30s"}
```

The package is named `time`, just like the standard library one, so it usually
has to be imported under an alias.

## Serialization

A duration is always written as a string, in the format of
[`time.Duration.String`](https://pkg.go.dev/time#Duration.String):

```go
json.Marshal(pkgTime.Duration(0))                  // "0s"
json.Marshal(pkgTime.Duration(30 * time.Second))  // "30s"
json.Marshal(pkgTime.Duration(5 * time.Hour))      // "5h0m0s"
```

The same holds for YAML:

```go
yaml.Marshal(pkgTime.Duration(30 * time.Second))  // "30s\n"
```

## Deserialization

Both a duration string and a number are accepted:

| value      | JSON                          | YAML              |
|------------|-------------------------------|-------------------|
| `"30s"`    | 30 seconds                    | 30 seconds        |
| `30`       | 30 nanoseconds                | error             |
| `30.5`     | 30 nanoseconds                | 30 nanoseconds    |
| `"1m30s"`  | 1 minute 30 seconds           | 1 minute 30 seconds |
| `true`     | error                         | error             |

The number is always a number of **nanoseconds**, the same unit
`time.Duration` uses internally.

The JSON and the YAML formats differ in the way they decode the numbers: JSON
decodes every number as a `float64`, while YAML decodes an integer as an `int`.
The YAML decoder only accepts the floating point numbers, so a whole number in
a YAML document is rejected:

```yaml
timeout: 30s    # ok
timeout: 30.5   # ok, 30 nanoseconds
timeout: 30     # error: invalid duration
```

When in doubt, use the string form: it is the one that is always accepted and
the one that is always written.

## Errors

An invalid value is reported as an error, and the duration is left unchanged:

- a string that is not a valid duration produces the error of
  [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration);
- a value of an unsupported type produces `invalid duration`.

## Comparison with `time.Duration`

`Duration` is a defined type, not an alias, so it does not inherit the methods
of `time.Duration`. Convert it back when they are needed:

```go
d := pkgTime.Duration(30 * time.Second)

fmt.Println(time.Duration(d).Seconds()) // 30
```

`Duration` is meant to be used in the configuration structures that are
serialized, not as a general-purpose replacement for `time.Duration`.
