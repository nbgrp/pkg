// SPDX-License-Identifier: BSD-3-Clause

// Package deferred provides a helper to close a resource in a deferred call
// without losing the close error.
package deferred

import (
	"errors"
	"io"
)

// Close closes the closer and joins the close error with the error pointed to by err.
// It is intended to be deferred, so that the close error is not lost.
// The closer is always closed; a nil err discards the close error.
func Close(err *error, closer io.Closer) { //nolint:gocritic // ptrToRefParam here is OK
	closeErr := closer.Close()
	if err == nil {
		return
	}
	*err = errors.Join(*err, closeErr)
}
