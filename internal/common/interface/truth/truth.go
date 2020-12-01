// Released under an MIT license. See LICENSE.

// Package truth defines the interface for oh types that have a truth value.
package truth

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

// I (truth) is anything that evaluates to a true or false value.
type I interface {
	Bool() bool
}

// Value returns the truth value for a cell.
func Value(c cell.I) bool {
	b, ok := c.(I)
	if !ok {
		return c != pair.Null
	}

	return b.Bool()
}
