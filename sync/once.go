// SPDX-License-Identifier: BSD-3-Clause

package sync

import (
	"sync"
	"sync/atomic"
)

// SuccessfulOnce allows to run user function once successfully.
//
// Unlike the standard library sync.Once, the function is retried as long as it
// fails: the call is considered successful only when the function returns a nil
// error, and every call made after that is a no-op.
type SuccessfulOnce struct {
	done atomic.Uint32
	mu   sync.Mutex
}

// Do runs fn until it succeeds.
//
// The function is called only while it has not succeeded yet. Its error is
// returned to the caller, and the next call retries it. Once fn has succeeded,
// Do returns nil without calling it again.
//
// The calls are serialized: while fn is running, the other callers wait for it
// to finish.
func (o *SuccessfulOnce) Do(fn func() error) error {
	if o.done.Load() == 0 {
		return o.doSlow(fn)
	}

	return nil
}

// doSlow runs fn under the lock, so that only one call is in flight at a time.
func (o *SuccessfulOnce) doSlow(fn func() error) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.done.Load() != 0 {
		return nil
	}

	if err := fn(); err != nil {
		return err
	}

	o.done.Store(1)
	return nil
}
