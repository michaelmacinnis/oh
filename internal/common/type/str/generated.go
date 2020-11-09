// Released under an MIT license. See LICENSE.

package str

import "github.com/michaelmacinnis/oh/internal/common/interface/cell"

// Is returns true if c is a *T.
func Is(c cell.I) bool {
	_, ok := c.(*T)

	return ok
}

// To returns a *T if c is a *T; Otherwise it panics.
func To(c cell.I) *T {
	if t, ok := c.(*T); ok {
		return t
	}

	panic("not a " + name)
}
