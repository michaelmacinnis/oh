// Released under an MIT license. See LICENSE.

package sym

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// Is returns true if c is a sym or sym plus.
func Is(c cell.I) bool {
	switch c.(type) {
	case *sym, *Plus:
		return true
	}

	return false
}

// To returns a sym if c is a sym or sym plus; Otherwise it panics.
func To(c cell.I) *sym {
	switch t := c.(type) {
	case *sym:
		return t
	case *Plus:
		return t.sym
	}

	panic("not a " + name)
}
