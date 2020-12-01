// Released under an MIT license. See LICENSE.

package sym

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// Is returns true if c is a *T or *Plus.
func Is(c cell.I) bool {
	switch c.(type) {
	case *T, *Plus:
		return true
	}

	return false
}

// To returns a *T if c is a *T or *Plus; Otherwise it panics.
func To(c cell.I) *T {
	switch t := c.(type) {
	case *T:
		return t
	case *Plus:
		return t.sym
	}

	panic("not a " + name)
}
