// SPDX-License-Identifier: BSD-3-Clause

package protomap_test

import (
	"testing"

	. "github.com/nbgrp/pkg/protomap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	stringValueName = protoreflect.FullName("google.protobuf.StringValue")
	structName      = protoreflect.FullName("google.protobuf.Struct")
	timestampName   = protoreflect.FullName("google.protobuf.Timestamp")
)

// valueMessage implements proto.Message with a value receiver, which makes it
// unusable as a prototype.
type valueMessage struct{}

func (valueMessage) ProtoReflect() protoreflect.Message {
	return (&wrapperspb.StringValue{}).ProtoReflect()
}

func TestProtomap_From(t *testing.T) {
	t.Parallel()

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		p := From([]proto.Message{})

		assert.NotNil(t, p)
		assert.Empty(t, p)
	})

	t.Run("single message", func(t *testing.T) {
		t.Parallel()

		p := From([]proto.Message{&wrapperspb.StringValue{Value: "value"}})
		require.Len(t, p, 1)

		newMsg, ok := p[stringValueName]
		require.True(t, ok)

		msg := newMsg()
		require.NotNil(t, msg)
		assert.True(t, proto.Equal(&wrapperspb.StringValue{}, msg))
	})

	t.Run("messages of different types", func(t *testing.T) {
		t.Parallel()

		p := From([]proto.Message{
			&wrapperspb.StringValue{Value: "value"},
			&structpb.Struct{Fields: map[string]*structpb.Value{
				"key": structpb.NewStringValue("value"),
			}},
			&timestamppb.Timestamp{Seconds: 1},
		})
		require.Len(t, p, 3)

		assert.Contains(t, p, stringValueName)
		assert.Contains(t, p, structName)
		assert.Contains(t, p, timestampName)
	})

	t.Run("duplicated message type", func(t *testing.T) {
		t.Parallel()

		p := From([]proto.Message{
			&wrapperspb.StringValue{Value: "first"},
			&wrapperspb.StringValue{Value: "second"},
		})
		require.Len(t, p, 1)

		msg := p[stringValueName]()
		require.NotNil(t, msg)
		assert.True(t, proto.Equal(&wrapperspb.StringValue{}, msg))
	})

	t.Run("typed nil message", func(t *testing.T) {
		t.Parallel()

		var msg *wrapperspb.StringValue

		p := From([]proto.Message{msg})
		require.Len(t, p, 1)

		newMsg := p[stringValueName]()
		require.NotNil(t, newMsg)
		assert.True(t, proto.Equal(&wrapperspb.StringValue{}, newMsg))
	})

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			From([]proto.Message{nil})
		})
	})

	t.Run("non-pointer message", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			From([]proto.Message{valueMessage{}})
		})
	})
}

func TestProtomap_Unmarshal(t *testing.T) {
	t.Parallel()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		src := &structpb.Struct{Fields: map[string]*structpb.Value{
			"str":  structpb.NewStringValue("value"),
			"num":  structpb.NewNumberValue(42),
			"bool": structpb.NewBoolValue(true),
		}}
		data, err := proto.Marshal(src)
		require.NoError(t, err)

		p := From([]proto.Message{&structpb.Struct{}})
		msg, err := p.Unmarshal(structName, data)
		require.NoError(t, err)
		assert.True(t, proto.Equal(src, msg))
	})

	t.Run("empty data", func(t *testing.T) {
		t.Parallel()

		p := From([]proto.Message{&wrapperspb.StringValue{}})

		msg, err := p.Unmarshal(stringValueName, nil)
		require.NoError(t, err)
		assert.True(t, proto.Equal(&wrapperspb.StringValue{}, msg))
	})

	t.Run("unknown message", func(t *testing.T) {
		t.Parallel()

		p := From([]proto.Message{&wrapperspb.StringValue{}})

		msg, err := p.Unmarshal("google.protobuf.Unknown", nil)
		require.ErrorIs(t, err, ErrNoPrototype)
		assert.Nil(t, msg)
	})

	t.Run("invalid data", func(t *testing.T) {
		t.Parallel()

		p := From([]proto.Message{&wrapperspb.StringValue{}})

		msg, err := p.Unmarshal(stringValueName, []byte{0x0A, 0x05, 0x01})
		require.Error(t, err)
		require.ErrorIs(t, err, proto.Error)
		require.ErrorContains(t, err, string(stringValueName))
		assert.NotNil(t, msg)
	})

	t.Run("nil map", func(t *testing.T) {
		t.Parallel()

		var p Protomap[proto.Message]

		msg, err := p.Unmarshal(stringValueName, nil)
		require.ErrorIs(t, err, ErrNoPrototype)
		assert.Nil(t, msg)
	})

	t.Run("independent instances", func(t *testing.T) {
		t.Parallel()

		src := &wrapperspb.StringValue{Value: "value"}
		data, err := proto.Marshal(src)
		require.NoError(t, err)

		p := From([]*wrapperspb.StringValue{src})

		first, err := p.Unmarshal(stringValueName, data)
		require.NoError(t, err)

		second, err := p.Unmarshal(stringValueName, data)
		require.NoError(t, err)

		first.Value = "changed"
		assert.Equal(t, "value", second.GetValue())
	})
}
