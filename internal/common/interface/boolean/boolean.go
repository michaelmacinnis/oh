// Released under an MIT license. See LICENSE.

// Package boolean defines the interface for oh types that have a boolean value.
package boolean

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

// I (boolean) is anything that evaluates to a true or false value.
type I interface {
	Bool() bool
}

// Value returns the boolean value for a cell.
func Value(c cell.I) bool {
	b, ok := c.(I)
	if !ok {
		return c != pair.Null
	}

	return b.Bool()
}
