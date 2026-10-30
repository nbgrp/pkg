// SPDX-License-Identifier: BSD-3-Clause

package protomap

import (
	"errors"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ErrNoPrototype is returned by [Protomap.Unmarshal] when the map has no
// prototype registered for the requested message name.
var ErrNoPrototype = errors.New("no prototype for message")

// Protomap maps the full names of protobuf message descriptors to constructors
// of new instances of those messages.
//
// The key is the full name of the protobuf message descriptor
// (protoreflect.FullName), the value is a function that creates a new instance
// of the message.
//
// It is used to deserialize protobuf messages dynamically, by their name, which
// makes it possible to create message instances without knowing their concrete
// types at compile time.
type Protomap[T proto.Message] map[protoreflect.FullName]func() T

// From builds a Protomap from a slice of protobuf messages.
//
// For every message in the slice a prototype is registered under the full name
// of its descriptor and can later be used to create new instances of that
// message type.
//
// Messages of different types may be mixed in the slice as long as they all
// implement T. If the same message type occurs in the slice more than once,
// only a single prototype is registered for it.
//
// From panics if the slice contains a nil message or a message that is not
// used as a pointer.
func From[T proto.Message](messages []T) Protomap[T] {
	p := make(Protomap[T], len(messages))
	for _, msg := range messages {
		if any(msg) == nil {
			panic("protomap: nil message")
		}

		msgType := reflect.TypeOf(msg)
		if msgType.Kind() != reflect.Pointer {
			panic(fmt.Sprintf("protomap: message type %s must be a pointer", msgType))
		}
		typ := msgType.Elem()

		fullName := msg.ProtoReflect().Descriptor().FullName()

		protoFunc := func() T {
			v, ok := reflect.TypeAssert[T](reflect.New(typ))
			if !ok {
				panic(fmt.Sprintf("protomap: %s does not implement %s", typ, reflect.TypeFor[T]()))
			}
			return v
		}

		// The prototype is created eagerly, so that a broken prototype is
		// reported when the map is built (on application start), and not later,
		// when a message is unmarshaled.
		_ = protoFunc()

		p[fullName] = protoFunc
	}

	return p
}

// Unmarshal deserializes protobuf data into a new message of the type
// registered under the given full descriptor name.
//
// It returns the new message instance and an error, if any. If the map has no
// prototype for the given name, the returned error wraps [ErrNoPrototype] and
// the message is the zero value.
//
// On a decoding error the message is returned as well: it may be partially
// filled with the data that was successfully parsed.
func (p Protomap[T]) Unmarshal(fullName protoreflect.FullName, data []byte) (msg T, err error) {
	protoFunc, ok := p[fullName]
	if !ok {
		return msg, fmt.Errorf("%w: %s", ErrNoPrototype, fullName)
	}

	msg = protoFunc()
	if err := proto.Unmarshal(data, msg); err != nil {
		return msg, fmt.Errorf("unmarshal %s: %w", fullName, err)
	}

	return msg, nil
}
