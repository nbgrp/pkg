# strcase

A small, dependency-free Go package for detecting the case of a string and
converting it between the common case styles.

## Public API

```go
type Mode int

const (
    _ Mode = iota
    FlatCase
    CamelCase
    SnakeCase
    KebabCase
)

func Detect(s string) Mode
func Convert(s string) Converter
func (c Converter) To(target Mode) string
func (c Mode) ConvertTo(target Mode, s string) string
```

- `Mode` is a case style. `FlatCase` is a lowercase string without separators
  (`helloworld`), `CamelCase` capitalizes the word boundaries
  (`helloWorld`), `SnakeCase` separates the words with underscores
  (`hello_world`), `KebabCase` separates them with dashes (`hello-world`).
- `Detect` returns the case of the given string.
- `Convert` detects the case and returns a `Converter` bound to that string.
- `Converter.To` converts the string to the target case.
- `Mode.ConvertTo` converts a string of a known case to the target case, for the
  cases where the detection is not needed.

## Usage

```go
import (
    "fmt"

    "github.com/nbgrp/pkg/strcase"
)

func main() {
    fmt.Println(strcase.Convert("hello_world").To(strcase.CamelCase)) // helloWorld
    fmt.Println(strcase.Convert("helloWorld").To(strcase.SnakeCase)) // hello_world
    fmt.Println(strcase.Convert("hello-world").To(strcase.CamelCase)) // helloWorld
}
```

When the case of the string is already known, `Mode.ConvertTo` can be used
directly, skipping the detection:

```go
fmt.Println(strcase.SnakeCase.ConvertTo(strcase.KebabCase, "hello_world")) // hello-world
```

## Detection

`Detect` looks at the separators first, and then at the letters:

- a string containing an underscore is a `SnakeCase` one;
- a string containing a dash is a `KebabCase` one;
- a string containing both lower and upper case letters is a `CamelCase` one;
- everything else is a `FlatCase` one.

`FlatCase` is also the fallback for an empty string, and for a string without
any case marker at all.

```go
strcase.Detect("hello_world")   // strcase.SnakeCase
strcase.Detect("hello-world")   // strcase.KebabCase
strcase.Detect("helloWorld")    // strcase.CamelCase
strcase.Detect("HELLO_WORLD")   // strcase.SnakeCase
strcase.Detect("helloworld")    // strcase.FlatCase
strcase.Detect("")              // strcase.FlatCase
```

Only the first 100 bytes of the string are inspected, so that the detection
stays cheap on long strings. A separator that appears later than that is not
taken into account.

## Conversion

The conversion is driven by the detected case of the source string:

| source                        | `FlatCase`  | `CamelCase` | `SnakeCase` | `KebabCase` |
|-------------------------------|-------------|-------------|-------------|-------------|
| `helloWorld` (`CamelCase`)    | `helloworld` | `helloWorld` | `hello_world` | `hello-world` |
| `hello_world` (`SnakeCase`)   | `helloworld` | `helloWorld` | `hello_world` | `hello-world` |
| `hello-world` (`KebabCase`)   | `helloworld` | `helloWorld` | `hello_world` | `hello-world` |
| `helloworld` (`FlatCase`)     | `helloworld` | `helloworld` | `helloworld` | `helloworld` |

Two rules deserve a note:

- Converting **from** `FlatCase` is a no-op: the case of such a string is
  unknown, so there is nothing to split or to join.
- Converting **to** `CamelCase` from a separated case removes the separator and
  capitalizes the following letter; converting from `CamelCase` splits the
  string into words.

## Abbreviations

When a camel case string is split into words, the abbreviations are recognized
and kept together:

```go
strcase.Convert("ReadJSONData").To(strcase.SnakeCase) // read_json_data
strcase.Convert("JSONData").To(strcase.SnakeCase)    // json_data
strcase.Convert("PlanBDone").To(strcase.SnakeCase)    // plan_b_done
```

A single letter is treated as a word of its own, and the conversion between two
separated cases (`SnakeCase` and `KebabCase`) only replaces the separator, so
the words themselves are left untouched.

## Limitations

The detection is a heuristic: it looks at the shape of the string, not at the
words it contains. A string without any separator and without any upper case
letter is indistinguishable from a `FlatCase` one, and is treated as such.
