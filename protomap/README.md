# protomap

A small Go package for deserializing protobuf messages dynamically — by their
full descriptor name, without knowing their concrete types at compile time.

It is useful when the message type is not known statically: a plugin registry, a
configuration format, a proxy that relays messages of arbitrary types, or a test
harness that decodes payloads produced elsewhere.

## Public API

```go
var ErrNoPrototype = errors.New("no prototype for message")

type Protomap[T proto.Message] map[protoreflect.FullName]func() T

func From[T proto.Message](messages []T) Protomap[T]
func (p Protomap[T]) Unmarshal(fullName protoreflect.FullName, data []byte) (T, error)
```

- `Protomap` maps the full name of a protobuf message descriptor
  (`protoreflect.FullName`) to a constructor of new instances of that message.
- `From` builds a `Protomap` from a slice of messages, using each message only
  as a prototype: the registered constructor always returns a **new, empty**
  instance of the same type.
- `Unmarshal` creates a new instance of the registered type and decodes the
  given bytes into it.

## Usage

```go
package main

import (
    "fmt"

    "github.com/nbgrp/pkg/protomap"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/structpb"
    "google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
    p := protomap.From([]proto.Message{
        &timestamppb.Timestamp{},
        &structpb.Struct{},
    })

    data, err := proto.Marshal(&timestamppb.Timestamp{Seconds: 42})
    if err != nil {
        panic(err)
    }

    msg, err := p.Unmarshal("google.protobuf.Timestamp", data)
    if err != nil {
        panic(err)
    }

    fmt.Println(msg.(*timestamppb.Timestamp).GetSeconds()) // 42
}
```

`From` is normally called once, on startup, with the list of the message types
the application is able to handle. `Unmarshal` is then called for every incoming
payload, with the name of the message it is expected to contain.

## Prototypes

A prototype is used for two things: to obtain the full name of the message
descriptor, and to know the concrete Go type to instantiate. The prototype
itself is never returned to the caller — `Unmarshal` always creates a fresh
instance, so the decoded messages never share state.

```go
src := &timestamppb.Timestamp{Seconds: 42}
p := protomap.From([]proto.Message{src})

first, _ := p.Unmarshal("google.protobuf.Timestamp", data)
second, _ := p.Unmarshal("google.protobuf.Timestamp", data)

first.Seconds = 100
fmt.Println(second.GetSeconds()) // 42
```

## Building the map

`From` accepts messages of different types in the same slice, as long as they
all implement the type parameter `T`. If the same message type occurs several
times, only a single prototype is registered for it.

The map is usually built with `T = proto.Message`, which gives access to every
registered message type. A narrower type parameter gives a map that only
contains the messages of that type:

```go
p := protomap.From([]*timestamppb.Timestamp{
    &timestamppb.Timestamp{},
})

msg, err := p.Unmarshal("google.protobuf.Timestamp", data)
// msg is a *timestamppb.Timestamp, no type assertion needed
```

`From` panics if the slice contains a nil message or a message that is not used
as a pointer — a prototype that cannot be instantiated is a programming error,
and it is better to fail on startup than on the first incoming payload. The
prototype is also created eagerly for the same reason: a broken prototype is
reported when the map is built, not later, when a message is unmarshaled.

## Errors

`Unmarshal` returns an error wrapping `ErrNoPrototype` when the map has no
prototype for the requested name:

```go
_, err := p.Unmarshal("google.protobuf.Unknown", data)
if errors.Is(err, protomap.ErrNoPrototype) {
    // the message type is not registered
}
```

A decoding error is returned as is, wrapped with the name of the message for
context. In that case the message is returned too — it may be partially filled
with the data that was successfully parsed.

## Concurrency

`Protomap` is a plain map, so it must not be modified concurrently with
`Unmarshal`. Build it once, before it is used, and it is safe to read from
multiple goroutines.
