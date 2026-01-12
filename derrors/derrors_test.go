// SPDX-License-Identifier: BSD-3-Clause

package derrors_test

import (
	"errors"
	"testing"

	. "github.com/nbgrp/pkg/derrors"
	"github.com/stretchr/testify/require"
)

var (
	externalErr = errors.New("external")
	internalErr = errors.New("internal")
)

func errorFunc() error {
	return externalErr
}

func noErrorFunc() error {
	return nil
}

func TestJoin(t *testing.T) {
	t.Run("internal error", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, noErrorFunc())
			return internalErr
		}

		err := fn()

		require.NotErrorIs(t, err, externalErr)
		require.ErrorIs(t, err, internalErr)
	})

	t.Run("external error", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, errorFunc())
			return nil
		}

		err := fn()

		require.ErrorIs(t, err, externalErr)
		require.NotErrorIs(t, err, internalErr)
	})

	t.Run("joint error", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, errorFunc())
			return internalErr
		}

		err := fn()

		require.ErrorIs(t, err, externalErr)
		require.ErrorIs(t, err, internalErr)
	})

	t.Run("no errors", func(t *testing.T) {
		fn := func() (err error) {
			defer Join(&err, noErrorFunc())
			return err
		}

		err := fn()

		require.NoError(t, err)
	})
}
