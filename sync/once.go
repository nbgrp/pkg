// SPDX-License-Identifier: BSD-3-Clause

package sync

import (
	"sync"
	"sync/atomic"
)

// SuccessfulOnce allows to run user function once successfully.
type SuccessfulOnce struct {
	done atomic.Uint32
	m    sync.Mutex
}

// Do runs fn if previous fn call returned an error.
func (o *SuccessfulOnce) Do(fn func() error) error {
	if o.done.Load() == 0 {
		return o.doSlow(fn)
	}

	return nil
}

func (o *SuccessfulOnce) doSlow(fn func() error) error {
	o.m.Lock()
	defer o.m.Unlock()

	if o.done.Load() != 0 {
		return nil
	}

	if err := fn(); err != nil {
		return err
	}

	o.done.Store(1)
	return nil
}
