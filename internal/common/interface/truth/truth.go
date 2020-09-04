// Released under an MIT license. See LICENSE.

// Package truth defines the interface for oh types that have a truth value.
package truth

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// I (truth) is anything that evaluates to a true or false value.
type I interface {
	Bool() bool
}

// Value returns the truth value for a cell, if possible.
func Value(c cell.I) bool {
	b, ok := c.(I)
	if !ok {
		panic(c.Name() + " cannot be used in a boolean context")
	}

	return b.Bool()
}
