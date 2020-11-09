// Released under an MIT license. See LICENSE.

// Package common defines common interfaces
package common

import (
	"fmt"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

type Stringer = fmt.Stringer

// String returns the string value for a cell, if possible.
func String(c cell.I) string {
	b, ok := c.(Stringer)
	if !ok {
		panic(c.Name() + " cannot be used in a string context")
	}

	return b.String()
}
