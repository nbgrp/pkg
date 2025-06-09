// SPDX-License-Identifier: BSD-3-Clause

package closer_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	. "github.com/nbgrp/pkg/closer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestCloser(t *testing.T) {
	key := struct{ k string }{k: "testkey"}

	t.Run("global", func(t *testing.T) {
		goleak.VerifyNone(t, goleak.IgnoreCurrent())

		SetContext(context.WithValue(context.Background(), key, "testvalue"))
		Add(func(ctx context.Context) error {
			assert.NoError(t, ctx.Err())
			assert.Equal(t, ctx.Value(key), "testvalue")
			return nil
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			<-Done()
			assert.NoError(t, Err())
			wg.Done()
		}()

		CloseAll()
		wg.Wait()
	})

	t.Run("with context cancel", func(t *testing.T) {
		goleak.VerifyNone(t, goleak.IgnoreCurrent())

		ctx, cancel := context.WithCancel(context.Background())
		_, c := New(ctx, WithContextCancel())

		var cnt atomic.Uint32
		c.Add(func(context.Context) error {
			cnt.Add(1)
			return errors.New("test error #1")
		})
		c.Add(func(context.Context) error {
			cnt.Add(1)
			return errors.New("test error #2")
		})

		c.SetContext(context.WithValue(ctx, key, "testvalue"))
		c.Add(func(ctx context.Context) error {
			assert.Equal(t, ctx.Value(key), "testvalue")
			return nil
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			<-c.Done()
			err := c.Err()
			assert.ErrorContains(t, err, "test error #1")
			assert.ErrorContains(t, err, "test error #2")
			wg.Done()
		}()

		cancel()
		wg.Wait()

		assert.Equal(t, uint32(2), cnt.Load())
	})

	t.Run("with signals", func(t *testing.T) {
		goleak.VerifyNone(t, goleak.IgnoreCurrent())

		ctx := context.Background()
		_, c := New(ctx, WithSignals(syscall.SIGINT, syscall.SIGTERM))

		var cnt atomic.Uint32
		c.Add(func(context.Context) error {
			cnt.Add(1)
			return nil
		})
		c.Add(func(context.Context) error {
			cnt.Add(1)
			return nil
		})

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			<-c.Done()
			wg.Done()
		}()

		err := syscall.Kill(os.Getpid(), syscall.SIGTERM)
		assert.NoError(t, err)

		wg.Wait()

		assert.Equal(t, uint32(2), cnt.Load())
	})
}
