// SPDX-License-Identifier: BSD-3-Clause

package derrors_test

import (
	"errors"
	"testing"

	. "github.com/nbgrp/pkg/derrors"
	"github.com/stretchr/testify/require"
)

var (
	errExternal = errors.New("external")
	errInternal = errors.New("internal")
)

func errorFunc() error {
	return errExternal
}

func noErrorFunc() error {
	return nil
}

func TestJoin(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, noErrorFunc())
			return errInternal
		}

		err := fn()

		require.NotErrorIs(t, err, errExternal)
		require.ErrorIs(t, err, errInternal)
	})

	t.Run("external error", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, errorFunc())
			return nil
		}

		err := fn()

		require.ErrorIs(t, err, errExternal)
		require.NotErrorIs(t, err, errInternal)
	})

	t.Run("joint error", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, errorFunc())
			return errInternal
		}

		err := fn()

		require.ErrorIs(t, err, errExternal)
		require.ErrorIs(t, err, errInternal)
	})

	t.Run("no errors", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, noErrorFunc())
			return err
		}

		err := fn()

		require.NoError(t, err)
	})

	t.Run("no panic with nil error reference", func(t *testing.T) {
		Join(nil, errInternal)
	})
}
