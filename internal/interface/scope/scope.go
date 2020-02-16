// Released under an MIT license. See LICENSE.

// Package scope defines the interface for oh's first-class environments and objects.
package scope

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/reference"
)

// T (scope) is the interface for oh's first-class environments and objects.
type T interface {
	cell.T

	Clone() T
	Enclosing() T
	Expose() T

	Define(k string, v cell.T)
	Export(k string, v cell.T)
	Lookup(k string) reference.T
	Public(k string) reference.T
	Remove(k string) bool

	Exported() int
	Visible(o T) bool
}
