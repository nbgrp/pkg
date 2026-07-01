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

type mode int

const (
	ModePriority mode = iota
	ModeConcurrent
)

type options struct {
	keySeparator rune
	wildcardMark rune
	mode         mode
}

type Option func(*options)

func WithKeySeparator(separator rune) Option {
	return func(opts *options) {
		opts.keySeparator = separator
	}
}

func WithWildcardMark(mark rune) Option {
	return func(opts *options) {
		opts.wildcardMark = mark
	}
}

func WithMode(mode mode) Option {
	return func(opts *options) {
		opts.mode = mode
	}
}

type nodeHandler struct {
	handler  pkgdispatcher.Handler
	priority int
	deleted  atomic.Bool
}

type node struct {
	children map[string]*node
	handlers []*nodeHandler

	mu sync.Mutex
}

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

func (d *dispatcher) Listen(keyPattern string, handler pkgdispatcher.Handler) (cancel func(), err error) {
	return d.ListenWithPriority(keyPattern, handler, 0)
}

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
				if stopPropagation.Inner != nil {
					errs[i] = stopPropagation.Inner
				}
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
					if stopPropagation.Inner != nil {
						err = stopPropagation.Inner
					} else {
						err = nil
					}
				}
				errs[i] = err
			})
		}
		wg.Wait()
	}

	return errors.Join(errs...)
}

func (d *dispatcher) splitKey(key string) iter.Seq[string] {
	return strings.FieldsFuncSeq(key, func(r rune) bool {
		return r == d.opts.keySeparator
	})
}
