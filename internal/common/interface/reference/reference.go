// Released under an MIT license. See LICENSE.

// Package reference defines the interface for oh's variable type.
package reference

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// I (reference) is anything that can hold a value.
type I interface {
	Copy() I
	Get() cell.I
	Set(cell.I)
}
