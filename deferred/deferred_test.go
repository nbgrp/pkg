// SPDX-License-Identifier: BSD-3-Clause

package deferred_test

import (
	"errors"
	"testing"

	. "github.com/nbgrp/pkg/deferred"
	"github.com/stretchr/testify/require"
)

var (
	errExternal = errors.New("external")
	errInternal = errors.New("internal")
)

type closer struct {
	err    error
	called bool
}

func (c *closer) Close() error {
	c.called = true
	return c.err
}

func errorCloser() *closer {
	return &closer{err: errExternal}
}

func noErrorCloser() *closer {
	return &closer{}
}

func TestJoin(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		fn := func() (err error) {
			defer Close(&err, noErrorCloser())
			return errInternal
		}

		err := fn()

		require.NotErrorIs(t, err, errExternal)
		require.ErrorIs(t, err, errInternal)
	})

	t.Run("external error", func(t *testing.T) {
		fn := func() (err error) {
			defer Close(&err, errorCloser())
			return nil
		}

		err := fn()

		require.ErrorIs(t, err, errExternal)
		require.NotErrorIs(t, err, errInternal)
	})

	t.Run("joint error", func(t *testing.T) {
		fn := func() (err error) {
			defer Close(&err, errorCloser())
			return errInternal
		}

		err := fn()

		require.ErrorIs(t, err, errExternal)
		require.ErrorIs(t, err, errInternal)
	})

	t.Run("no errors", func(t *testing.T) {
		fn := func() (err error) {
			defer Close(&err, noErrorCloser())
			return err
		}

		err := fn()

		require.NoError(t, err)
	})

	t.Run("no panic with nil error reference", func(t *testing.T) {
		c := noErrorCloser()
		Close(nil, c)

		require.True(t, c.called)
	})
}
