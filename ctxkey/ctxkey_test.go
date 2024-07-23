package ctxkey_test

import (
	"context"
	"testing"

	. "github.com/nbgrp/pkg/ctxkey"
	"github.com/stretchr/testify/assert"
)

func TestCtxkey(t *testing.T) {
	t.Parallel()

	key := New("testkey")
	val := "testvalue"
	ctx := context.WithValue(context.Background(), key, val)

	assert.Equal(t, ctx.Value(key), val)
	assert.Nil(t, ctx.Value("testkey"))

	assert.Equal(t, `ctxkey "testkey"`, key.String())
}
