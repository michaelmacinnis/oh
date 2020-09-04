// Released under an MIT license. See LICENSE.

// Package integer converts an oh cell to an int64 value, if possible.
package integer

import (
	"strconv"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
)

// Value returns the int64 value for a cell, if possible.
func Value(c cell.I) int64 {
	r, isRational := c.(rational.I)
	if isRational {
		br := r.Rat()
		if br.IsInt() {
			bi := br.Num()
			if bi.IsInt64() {
				return bi.Int64()
			}
		}

		panic(c.Name() + " does not have an integer value")
	}

	s, isSym := c.(*sym.T)
	if isSym {
		i, err := strconv.ParseInt(s.String(), 10, 64)
		if err != nil {
			return i
		}
	}

	panic(c.Name() + " cannot be converted to an integer value")
}
