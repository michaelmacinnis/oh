// Released under an MIT license. See LICENSE.

// Package reference defines the interface for oh's variable type.
package reference

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

// T (reference) is anything that can hold a value.
type T interface {
	Copy() T
	Get() cell.T
	Set(cell.T)
}
