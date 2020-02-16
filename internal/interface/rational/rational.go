// Released under an MIT license. See LICENSE.

// Package rational defines the interface for oh's numeric types.
package rational

import (
	"math/big"

	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

// T (rational) is anything that can be treated as a rational number in oh.
type T interface {
	Rat() *big.Rat
}

// Number returns the *big.Rat value for a cell, if possible.
func Number(c cell.T) *big.Rat {
	r, ok := c.(T)
	if !ok {
		// Not all cell types can be treated as numbers.
		panic(c.Name() + " cannot be use in a numeric expression")
	}

	return r.Rat()
}
