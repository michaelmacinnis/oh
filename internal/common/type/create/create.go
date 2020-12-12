// Released under an MIT license. See LICENSE.

// Package create provides helper functions for creating oh types.
package create

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
)

// Bool returns the oh value corresponding to the value of the boolean a.
func Bool(a bool) cell.I {
	if a {
		return sym.True
	}

	return pair.Null
}
