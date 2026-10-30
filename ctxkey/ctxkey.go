// SPDX-License-Identifier: BSD-3-Clause

// Package ctxkey is like errors.New() for context keys.
//
// Origin: https://github.com/cstockton/pkg/tree/master/ctxkey
//
// # Invariants
//
// All context keys created with New() will be globally unique during program
// execution for the lifespan of the Key and will not cause additional
// allocations when they are associated with a context value. Uniqueness is
// satisfied by returning pointers to a private key struct, forcing comparison
// operations in context value retrieval to use pointer equality within the
// current Go's specification:
//
// https://golang.org/ref/spec#Comparison_operators
//
// Additional allocations means that you will not allocate for the context key
// as seen with this common idiom:
//
//	type key int
//	var myKey key = 0
//	// 3 allocations, context struct, myKey and someValue
//	context.WithValue(ctx, myKey, SomeValue)
//
// Keys issued by New() fit within an interface{} value, i.e.:
//
//	var myKey = ctxkey.New(`MyKey`)
//	// 2 allocations, context struct and someValue
//	context.WithValue(ctx, myKey, SomeValue)
package ctxkey

import (
	"fmt"
)

// Key is a context key created by [New].
//
// It is an opaque, unique value: two keys created by [New] are never equal,
// even if they were given the same name.
type Key interface {
	fmt.Stringer
}

// key is a pointer-backed context key. The pointer identity is what makes the
// key unique, the name is only used for the [fmt.Stringer] representation.
type key struct {
	name string
}

// New returns a new unique context key with the given name.
//
// The name is only used to make the key human-readable in logs and debug
// output; it does not affect the uniqueness of the key.
func New(name string) Key {
	return &key{name}
}

// String returns the human-readable representation of the key.
func (k key) String() string {
	return `ctxkey "` + k.name + `"`
}
