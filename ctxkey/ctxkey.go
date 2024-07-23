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
//  https://golang.org/ref/spec#Comparison_operators
//
// Additional allocations means that you will not allocate for the context key
// as seen with this common idiom:
//
//  type key int
//  var myKey key = 0
//  // 3 allocations, context struct, myKey and someValue
//  context.WithValue(ctx, myKey, SomeValue)
//
// Keys issued by New() fit within an interface{} value, i.e.:
//
//  var myKey = ctxkey.New(`MyKey`)
//  // 2 allocations, context struct and someValue
//  context.WithValue(ctx, myKey, SomeValue)
package ctxkey

import (
	"fmt"
)

type Key interface {
	fmt.Stringer
}

type key struct {
	name string
}

func New(name string) Key {
	return &key{name}
}

func (k key) String() string {
	return `ctxkey "` + k.name + `"`
}
