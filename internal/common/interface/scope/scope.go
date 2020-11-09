// Released under an MIT license. See LICENSE.

// Package scope defines the interface for oh's first-class environments and objects.
package scope

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/reference"
	"github.com/michaelmacinnis/oh/internal/common/struct/hash"
)

// I (scope) is the interface for oh's first-class environments and objects.
type I interface {
	cell.I

	Clone() I
	Enclosing() I
	Expose() I

	Define(k string, v cell.I)
	Export(k string, v cell.I)
	Lookup(k string) reference.I
	Public() *hash.T
	Remove(k string) bool

	Exported() int
	Visible(o I) bool
}

type scope = I

// Is returns true if c is a scope.
func Is(c cell.I) bool {
	_, ok := c.(scope)

	return ok
}

// To returns a scope if c is a scope; Otherwise it panics.
func To(c cell.I) scope {
	if t, ok := c.(scope); ok {
		return t
	}

	panic(c.Name() + " cannot be used in an object context")
}
