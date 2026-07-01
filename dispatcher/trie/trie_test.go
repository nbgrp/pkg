// SPDX-License-Identifier: BSD-3-Clause

package trie_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	pkgdispatcher "github.com/nbgrp/pkg/dispatcher"
	"github.com/nbgrp/pkg/dispatcher/trie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDispatcher(t *testing.T, opts ...trie.Option) interface {
	pkgdispatcher.Dispatcher
	pkgdispatcher.Listener
	pkgdispatcher.PriorityListener
} {
	t.Helper()

	d, err := trie.NewDispatcher(opts...)
	require.NoError(t, err)

	return d
}

func recordingHandler(calls *[]string, id string) pkgdispatcher.Handler {
	return func(_ context.Context, _ ...any) error {
		*calls = append(*calls, id)

		return nil
	}
}

func TestDispatcher_PriorityMode_NoHandlers(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	err := d.Dispatch(t.Context(), "a.b.c")
	require.NoError(t, err)
}

func TestDispatcher_PriorityMode_SingleHandler(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	var calls []string
	cancel, err := d.Listen("a.b.c", recordingHandler(&calls, "h1"))
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, d.Dispatch(t.Context(), "a.b.c"))
	assert.Equal(t, []string{"h1"}, calls)

	require.NoError(t, d.Dispatch(t.Context(), "a.b.c"))
	assert.Equal(t, []string{"h1", "h1"}, calls)
}

func TestDispatcher_PriorityMode_MultipleHandlers(t *testing.T) {
	t.Parallel()

	type entry struct {
		id       string
		priority int
	}

	tests := []struct {
		name         string
		priorities   []entry
		expectedRuns []string
	}{
		{
			name:         "all zero priority",
			priorities:   []entry{{"a", 0}, {"b", 0}, {"c", 0}},
			expectedRuns: []string{"a", "b", "c"},
		},
		{
			name:         "ascending priorities",
			priorities:   []entry{{"a", 1}, {"b", 2}, {"c", 3}},
			expectedRuns: []string{"c", "b", "a"},
		},
		{
			name:         "descending priorities",
			priorities:   []entry{{"a", 3}, {"b", 2}, {"c", 1}},
			expectedRuns: []string{"a", "b", "c"},
		},
		{
			name:         "mixed priorities",
			priorities:   []entry{{"a", 5}, {"b", -1}, {"c", 0}, {"d", 10}, {"e", 5}},
			expectedRuns: []string{"d", "a", "e", "c", "b"},
		},
		{
			name:         "all same priority",
			priorities:   []entry{{"a", 7}, {"b", 7}, {"c", 7}},
			expectedRuns: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := newDispatcher(t)

			var calls []string
			for _, e := range tt.priorities {
				cancel, err := d.ListenWithPriority("evt", recordingHandler(&calls, e.id), e.priority)
				require.NoError(t, err)
				t.Cleanup(cancel)
			}

			require.NoError(t, d.Dispatch(t.Context(), "evt"))
			assert.Equal(t, tt.expectedRuns, calls)
		})
	}
}

func TestDispatcher_PriorityMode_HandlerErrors(t *testing.T) {
	t.Parallel()

	t.Run("all handlers return nil", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t)

		var calls []string
		for i := range 3 {
			cancel, err := d.ListenWithPriority("evt", recordingHandler(&calls, strconv.Itoa(i)), i)
			require.NoError(t, err)
			t.Cleanup(cancel)
		}

		require.NoError(t, d.Dispatch(t.Context(), "evt"))
		assert.Equal(t, []string{"2", "1", "0"}, calls)
	})

	t.Run("stop propagation with nil inner", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t)

		var calls []string
		cancel, err := d.ListenWithPriority("evt", recordingHandler(&calls, "first"), 10)
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", func(_ context.Context, _ ...any) error {
			calls = append(calls, "stop")

			return &pkgdispatcher.StopPropagationError{}
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.ListenWithPriority("evt", recordingHandler(&calls, "last"), -5)
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.NoError(t, d.Dispatch(t.Context(), "evt"))
		assert.Equal(t, []string{"first", "stop"}, calls)
	})

	t.Run("stop propagation with inner error", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t)

		innerErr := errors.New("boom")

		cancel, err := d.Listen("evt", func(_ context.Context, _ ...any) error {
			return &pkgdispatcher.StopPropagationError{Inner: innerErr}
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", func(_ context.Context, _ ...any) error {
			t.Fatal("handler must not be called after StopPropagationError")

			return nil
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.ErrorIs(t, d.Dispatch(t.Context(), "evt"), innerErr)
	})

	t.Run("handler errors are joined", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t)

		err1 := errors.New("first")
		err2 := errors.New("second")

		cancel, err := d.ListenWithPriority("evt", func(_ context.Context, _ ...any) error {
			return err1
		}, 10)
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.ListenWithPriority("evt", func(_ context.Context, _ ...any) error {
			return err2
		}, 5)
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.ListenWithPriority("evt", func(_ context.Context, _ ...any) error {
			return nil
		}, 1)
		require.NoError(t, err)
		t.Cleanup(cancel)

		got := d.Dispatch(t.Context(), "evt")
		require.Error(t, got)
		require.ErrorIs(t, got, err1)
		require.ErrorIs(t, got, err2)
	})
}

func TestDispatcher_Listen_InvalidKey(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	t.Run("empty key", func(t *testing.T) {
		t.Parallel()

		cancel, err := d.Listen("", recordingHandler(&[]string{}, "h"))
		assert.Nil(t, cancel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-empty")
	})

	t.Run("consecutive separators", func(t *testing.T) {
		t.Parallel()

		cancel, err := d.Listen("a..b", recordingHandler(&[]string{}, "h"))
		assert.Nil(t, cancel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty parts")
	})

	t.Run("leading separator", func(t *testing.T) {
		t.Parallel()

		cancel, err := d.Listen(".a", recordingHandler(&[]string{}, "h"))
		assert.Nil(t, cancel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "should not start with separator")
	})

	t.Run("trailing separator", func(t *testing.T) {
		t.Parallel()

		cancel, err := d.Listen("a.", recordingHandler(&[]string{}, "h"))
		assert.Nil(t, cancel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "should not end with separator")
	})

	t.Run("nil handler", func(t *testing.T) {
		t.Parallel()

		cancel, err := d.Listen("a.b", nil)
		assert.Nil(t, cancel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-nil")
	})

	t.Run("ListenWithPriority nil handler", func(t *testing.T) {
		t.Parallel()

		cancel, err := d.ListenWithPriority("a.b", nil, 1)
		assert.Nil(t, cancel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-nil")
	})
}

func TestDispatcher_Dispatch_Payload(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	var got []any
	cancel, err := d.Listen("evt", func(_ context.Context, payload ...any) error {
		got = payload

		return nil
	})
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, d.Dispatch(t.Context(), "evt", 42, "hello"))
	assert.Equal(t, []any{42, "hello"}, got)

	require.NoError(t, d.Dispatch(t.Context(), "evt"))
	assert.Empty(t, got)
}

func TestDispatcher_Cancel(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	var calls []string

	cancelA, err := d.Listen("evt", recordingHandler(&calls, "a"))
	require.NoError(t, err)

	cancelB, err := d.Listen("evt", recordingHandler(&calls, "b"))
	require.NoError(t, err)

	require.NoError(t, d.Dispatch(t.Context(), "evt"))
	assert.ElementsMatch(t, []string{"a", "b"}, calls)

	cancelA()

	require.NoError(t, d.Dispatch(t.Context(), "evt"))
	assert.Equal(t, []string{"a", "b", "b"}, calls)

	cancelB()

	require.NoError(t, d.Dispatch(t.Context(), "evt"))
	assert.Equal(t, []string{"a", "b", "b"}, calls)
}

func TestDispatcher_ConcurrentMode_NoHandlers(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t, trie.WithMode(trie.ModeConcurrent))

	err := d.Dispatch(t.Context(), "a.b.c")
	require.NoError(t, err)
}

func TestDispatcher_ConcurrentMode_SingleHandler(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t, trie.WithMode(trie.ModeConcurrent))

	var calls []string
	cancel, err := d.Listen("a.b.c", recordingHandler(&calls, "h1"))
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, d.Dispatch(t.Context(), "a.b.c"))
	assert.Equal(t, []string{"h1"}, calls)

	require.NoError(t, d.Dispatch(t.Context(), "a.b.c"))
	assert.Equal(t, []string{"h1", "h1"}, calls)
}

func TestDispatcher_ConcurrentMode_MultipleHandlers(t *testing.T) {
	t.Parallel()

	t.Run("all handlers are called", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithMode(trie.ModeConcurrent))

		var (
			mu    sync.Mutex
			calls []string
		)
		record := func(id string) pkgdispatcher.Handler {
			return func(_ context.Context, _ ...any) error {
				mu.Lock()
				calls = append(calls, id)
				mu.Unlock()

				return nil
			}
		}

		for _, id := range []string{"a", "b", "c"} {
			cancel, err := d.Listen("evt", record(id))
			require.NoError(t, err)
			t.Cleanup(cancel)
		}

		require.NoError(t, d.Dispatch(t.Context(), "evt"))
		assert.ElementsMatch(t, []string{"a", "b", "c"}, calls)
	})

	t.Run("stop propagation does not interrupt others", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithMode(trie.ModeConcurrent))

		var (
			mu    sync.Mutex
			calls []string
		)
		record := func(id string) pkgdispatcher.Handler {
			return func(_ context.Context, _ ...any) error {
				mu.Lock()
				calls = append(calls, id)
				mu.Unlock()

				return nil
			}
		}

		cancel, err := d.Listen("evt", record("a"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", func(_ context.Context, _ ...any) error {
			mu.Lock()
			calls = append(calls, "stop")
			mu.Unlock()

			return &pkgdispatcher.StopPropagationError{}
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", record("b"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", record("c"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		// In ModeConcurrent StopPropagationError has no interrupting
		// effect: every handler is launched via wg.Go, so all of them
		// run regardless. The StopPropagationError itself is dropped
		// from the joined result (only its Inner, if any, is kept).
		require.NoError(t, d.Dispatch(t.Context(), "evt"))

		mu.Lock()
		gotCalls := append([]string(nil), calls...)
		mu.Unlock()
		assert.ElementsMatch(t, []string{"a", "b", "c", "stop"}, gotCalls)
	})

	t.Run("stop propagation with inner error still propagates inner", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithMode(trie.ModeConcurrent))

		innerErr := errors.New("boom")

		cancel, err := d.Listen("evt", func(_ context.Context, _ ...any) error {
			return &pkgdispatcher.StopPropagationError{Inner: innerErr}
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", func(_ context.Context, _ ...any) error {
			return nil
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		got := d.Dispatch(t.Context(), "evt")
		require.ErrorIs(t, got, innerErr)
	})

	t.Run("handler errors are joined", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithMode(trie.ModeConcurrent))

		err1 := errors.New("first")
		err2 := errors.New("second")
		err3 := errors.New("third")

		var called atomic.Int32
		cancel, err := d.Listen("evt", func(_ context.Context, _ ...any) error {
			called.Add(1)

			return err1
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", func(_ context.Context, _ ...any) error {
			called.Add(1)

			return err2
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("evt", func(_ context.Context, _ ...any) error {
			called.Add(1)

			return err3
		})
		require.NoError(t, err)
		t.Cleanup(cancel)

		got := d.Dispatch(t.Context(), "evt")
		require.Error(t, got)
		require.ErrorIs(t, got, err1)
		require.ErrorIs(t, got, err2)
		require.ErrorIs(t, got, err3)
		assert.Equal(t, int32(3), called.Load())
	})
}

func TestDispatcher_Wildcard_TrailingSegment(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	var calls []string
	cancel, err := d.Listen("a.*", recordingHandler(&calls, "h"))
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, d.Dispatch(t.Context(), "a.b"))
	assert.Equal(t, []string{"h"}, calls)

	require.NoError(t, d.Dispatch(t.Context(), "a.c"))
	assert.Equal(t, []string{"h", "h"}, calls)
}

func TestDispatcher_Wildcard_LeadingSegment(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	var calls []string
	cancel, err := d.Listen("*.a", recordingHandler(&calls, "h"))
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, d.Dispatch(t.Context(), "b.a"))
	assert.Equal(t, []string{"h"}, calls)

	require.NoError(t, d.Dispatch(t.Context(), "anything.a"))
	assert.Equal(t, []string{"h", "h"}, calls)
}

func TestDispatcher_Wildcard_MiddleSegment(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	var calls []string
	cancel, err := d.Listen("a.*.b", recordingHandler(&calls, "h"))
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, d.Dispatch(t.Context(), "a.x.b"))
	assert.Equal(t, []string{"h"}, calls)

	require.NoError(t, d.Dispatch(t.Context(), "a.zzz.b"))
	assert.Equal(t, []string{"h", "h"}, calls)
}

func TestDispatcher_Wildcard_MultipleHandlers(t *testing.T) {
	t.Parallel()

	d := newDispatcher(t)

	var calls []string
	cancel, err := d.Listen("a.*", recordingHandler(&calls, "h1"))
	require.NoError(t, err)
	t.Cleanup(cancel)

	cancel, err = d.Listen("a.*", recordingHandler(&calls, "h2"))
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, d.Dispatch(t.Context(), "a.x"))
	assert.Equal(t, []string{"h1", "h2"}, calls)
}

func TestDispatcher_Wildcard_WithExactHandler(t *testing.T) {
	t.Parallel()

	t.Run("both exact and wildcard fire", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t)

		var calls []string
		cancel, err := d.Listen("a.b", recordingHandler(&calls, "exact"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("a.*", recordingHandler(&calls, "wild"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.NoError(t, d.Dispatch(t.Context(), "a.b"))
		assert.Equal(t, []string{"wild", "exact"}, calls)
	})

	t.Run("wildcard fires when exact does not", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t)

		var calls []string
		cancel, err := d.Listen("a.b", recordingHandler(&calls, "exact"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("a.*", recordingHandler(&calls, "wild"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.NoError(t, d.Dispatch(t.Context(), "a.c"))
		assert.Equal(t, []string{"wild"}, calls)
	})

	t.Run("exact fires when no wildcard matches", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t)

		var calls []string
		cancel, err := d.Listen("a.b", recordingHandler(&calls, "exact"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		cancel, err = d.Listen("x.*", recordingHandler(&calls, "wild"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.NoError(t, d.Dispatch(t.Context(), "a.b"))
		assert.Equal(t, []string{"exact"}, calls)
	})
}

func TestDispatcher_WithKeySeparator(t *testing.T) {
	t.Parallel()

	t.Run("custom separator splits keys", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithKeySeparator('/'))

		var calls []string
		cancel, err := d.Listen("a/b/c", recordingHandler(&calls, "h"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.NoError(t, d.Dispatch(t.Context(), "a/b/c"))
		assert.Equal(t, []string{"h"}, calls)
	})

	t.Run("custom separator does not match default", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithKeySeparator('/'))

		var calls []string
		cancel, err := d.Listen("a/b", recordingHandler(&calls, "h"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		// "a.b" is a single segment under '/' separator, so the handler
		// registered for "a/b" must not fire.
		require.NoError(t, d.Dispatch(t.Context(), "a.b"))
		assert.Empty(t, calls)
	})
}

func TestDispatcher_WithWildcardMark(t *testing.T) {
	t.Parallel()

	t.Run("custom mark acts as wildcard", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithWildcardMark('#'))

		var calls []string
		cancel, err := d.Listen("a.#.c", recordingHandler(&calls, "h"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.NoError(t, d.Dispatch(t.Context(), "a.b.c"))
		assert.Equal(t, []string{"h"}, calls)
	})

	t.Run("default mark loses wildcard meaning", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithWildcardMark('#'))

		var calls []string
		cancel, err := d.Listen("a.*.c", recordingHandler(&calls, "h"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		// "a.*.c" is registered with '*' as a literal segment under the
		// new wildcard mark '#'; "a.b.c" must not match it.
		require.NoError(t, d.Dispatch(t.Context(), "a.b.c"))
		assert.Empty(t, calls)
	})

	t.Run("custom mark matches a literal segment", func(t *testing.T) {
		t.Parallel()

		d := newDispatcher(t, trie.WithWildcardMark('#'))

		var calls []string
		cancel, err := d.Listen("a.#.c", recordingHandler(&calls, "h"))
		require.NoError(t, err)
		t.Cleanup(cancel)

		require.NoError(t, d.Dispatch(t.Context(), "a.#.c"))
		assert.Equal(t, []string{"h"}, calls)
	})
}
