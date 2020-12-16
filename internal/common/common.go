// Released under an MIT license. See LICENSE.

// Package common defines common interfaces
package common

import (
	"fmt"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// Error wraps a string as an error.
type Error string

// Error returns the error string for the Error e.
func (e Error) Error() string {
	return string(e)
}

// String returns the string value for a cell, if possible.
func String(c cell.I) string {
	b, ok := c.(fmt.Stringer)
	if !ok {
		panic(c.Name() + " cannot be used in a string context")
	}

	return b.String()
}
