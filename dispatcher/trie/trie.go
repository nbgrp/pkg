// SPDX-License-Identifier: BSD-3-Clause

package trie

import (
	"context"
	"errors"
	"iter"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	pkgdispatcher "github.com/nbgrp/pkg/dispatcher"
)

// mode is the way the matched handlers are executed.
type mode int

const (
	// ModePriority runs the handlers sequentially, in the descending order of
	// their priority.
	ModePriority mode = iota
	// ModeConcurrent runs all the matched handlers concurrently and waits for
	// all of them to finish.
	ModeConcurrent
)

type options struct {
	keySeparator rune
	wildcardMark rune
	mode         mode
}

// Option configures a dispatcher created by [NewDispatcher].
type Option func(*options)

// WithKeySeparator sets the separator the keys are split by.
// The default is '.'.
func WithKeySeparator(separator rune) Option {
	return func(opts *options) {
		opts.keySeparator = separator
	}
}

// WithWildcardMark sets the mark that denotes a wildcard part of a key.
// The default is '*'.
func WithWildcardMark(mark rune) Option {
	return func(opts *options) {
		opts.wildcardMark = mark
	}
}

// WithMode sets the mode the matched handlers are executed in.
// The default is [ModePriority].
func WithMode(mode mode) Option {
	return func(opts *options) {
		opts.mode = mode
	}
}

// nodeHandler is a handler registered on a node of the trie.
type nodeHandler struct {
	handler  pkgdispatcher.Handler
	priority int
	// deleted is set by the cancel function returned by the Listen methods.
	// The handler stays in the trie until it is pruned on the next match.
	deleted atomic.Bool
}

// node is a part of the trie. It holds the handlers registered for the key
// ending with this part, and the nodes for the next parts.
type node struct {
	children map[string]*node
	handlers []*nodeHandler

	mu sync.Mutex
}

// purgeDeleted removes the handlers that were cancelled from the node.
//
// It is called on the visited nodes only, so a cancelled handler is pruned
// when a dispatch walks through its node.
func (n *node) purgeDeleted() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.handlers = slices.DeleteFunc(n.handlers, func(h *nodeHandler) bool {
		return h.deleted.Load()
	})
}

func newNode() *node {
	return &node{
		children: make(map[string]*node),
	}
}

type dispatcher struct {
	root *node
	opts options
}

// NewDispatcher returns a dispatcher with the given options.
//
// The default options are [ModePriority], a '.' key separator and a '*'
// wildcard mark.
func NewDispatcher(opts ...Option) (*dispatcher, error) {
	o := options{
		keySeparator: '.',
		wildcardMark: '*',
	}
	for _, opt := range opts {
		opt(&o)
	}

	return &dispatcher{
		opts: o,
		root: newNode(),
	}, nil
}

// Listen registers the handler for the key pattern and returns the function
// that cancels the registration.
//
// The key pattern must not be empty, must not start or end with the separator,
// must not contain two consecutive separators, and the handler must not be nil.
func (d *dispatcher) Listen(keyPattern string, handler pkgdispatcher.Handler) (cancel func(), err error) {
	return d.ListenWithPriority(keyPattern, handler, 0)
}

// ListenWithPriority registers the handler for the key pattern with the given
// priority and returns the function that cancels the registration.
//
// The higher the priority, the earlier the handler is called in [ModePriority].
// The priority is ignored in [ModeConcurrent].
func (d *dispatcher) ListenWithPriority(keyPattern string, handler pkgdispatcher.Handler, priority int) (cancel func(), err error) {
	switch {
	case keyPattern == "":
		return nil, errors.New("key should be non-empty string")
	case strings.HasPrefix(keyPattern, string(d.opts.keySeparator)):
		return nil, errors.New("key should not start with separator")
	case strings.HasSuffix(keyPattern, string(d.opts.keySeparator)):
		return nil, errors.New("key should not end with separator")
	case strings.Contains(keyPattern, string([]rune{d.opts.keySeparator, d.opts.keySeparator})):
		return nil, errors.New("key should not contain empty parts")
	case handler == nil:
		return nil, errors.New("handler should be non-nil")
	}

	node := d.root
	for k := range d.splitKey(keyPattern) {
		if _, ok := node.children[k]; !ok {
			node.mu.Lock()
			node.children[k] = newNode()
			node.mu.Unlock()
		}
		node = node.children[k]
	}

	h := nodeHandler{
		handler:  handler,
		priority: priority,
	}

	node.mu.Lock()
	node.handlers = append(node.handlers, &h)
	node.mu.Unlock()

	return func() {
		h.deleted.Store(true)
	}, nil
}

// Dispatch calls the handlers registered for the key and returns their joined
// errors.
//
// The handlers are called in the order that depends on the mode of the
// dispatcher, and the payload is passed to all of them.
func (d *dispatcher) Dispatch(ctx context.Context, key string, payload ...any) error { //nolint:gocognit
	wm := string(d.opts.wildcardMark)
	var handlers []*nodeHandler

	stack := []*node{d.root}
	for k := range d.splitKey(key) {
		for range stack {
			node := stack[0]
			stack = stack[1:]

			if wild, ok := node.children[wm]; ok {
				wild.purgeDeleted()
				handlers = append(handlers, wild.handlers...)
				if len(wild.children) > 0 {
					stack = append(stack, wild)
				}
			}

			if k == wm {
				continue
			}

			if next, ok := node.children[k]; ok {
				stack = append(stack, next)
			}
		}
	}

	for _, node := range stack {
		if len(node.children) == 0 {
			node.purgeDeleted()
			handlers = append(handlers, node.handlers...)
		}
	}

	handlers = slices.DeleteFunc(handlers, func(h *nodeHandler) bool {
		return h.deleted.Load()
	})

	errs := make([]error, len(handlers))

	switch d.opts.mode {
	case ModePriority:
		slices.SortStableFunc(handlers, func(a, b *nodeHandler) int {
			return b.priority - a.priority
		})

		for i, h := range handlers {
			err := h.handler(ctx, payload...)
			if stopPropagation, ok := errors.AsType[*pkgdispatcher.StopPropagationError](err); ok {
				errs[i] = stopPropagation.Inner
				break
			}
			errs[i] = err
		}

	case ModeConcurrent:
		var wg sync.WaitGroup
		for i, h := range handlers {
			wg.Go(func() {
				err := h.handler(ctx, payload...)
				if stopPropagation, ok := errors.AsType[*pkgdispatcher.StopPropagationError](err); ok {
					err = stopPropagation.Inner
				}
				errs[i] = err
			})
		}
		wg.Wait()
	}

	return errors.Join(errs...)
}

// splitKey splits the key into the parts separated by the key separator.
//
// The empty parts are dropped, so a dispatched key with consecutive, leading
// or trailing separators is split into the same parts as a valid one.
func (d *dispatcher) splitKey(key string) iter.Seq[string] {
	return strings.FieldsFuncSeq(key, func(r rune) bool {
		return r == d.opts.keySeparator
	})
}
