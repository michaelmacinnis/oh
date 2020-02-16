// Released under an MIT license. See LICENSE.

// Package boolean defines the interface for oh's boolean types.
package boolean

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

// T (boolean) is anything that evaluates to a true or false value.
type T interface {
	Bool() bool
}

// Value returns the bool value for a cell, if possible.
func Value(c cell.T) bool {
	b, ok := c.(T)
	if !ok {
		panic(c.Name() + " cannot be used in a boolean expression")
	}

	return b.Bool()
}
