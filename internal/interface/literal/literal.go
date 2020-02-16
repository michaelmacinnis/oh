// Released under an MIT license. See LICENSE.

// Package literal defines the interface for oh types that can be expressed as literals.
package literal

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

// T (literal) is any type that can be expressed as a literal.
type T interface {
	Literal() string
}

// String returns the literal string representaition for a cell, if possible.
func String(c cell.T) string {
	l, ok := c.(T)
	if !ok {
		// Not all cell types can be expressed as literals.
		panic(c.Name() + " does not have a literal representation")
	}
	return l.Literal()
}
