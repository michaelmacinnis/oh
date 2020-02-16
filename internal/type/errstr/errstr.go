// Released under an MIT license. See LICENSE.

// Package errstr provides oh's string error type.
package errstr

import (
	"github.com/michaelmacinnis/oh/internal/adapted"
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/type/str"
)

const name = "errstr"

// T (errstr) is a string error.
type T str.T

// New creates a new errstr from the string v.
func New(v string) *T {
	return (*T)(&v)
}

// The errstr type is a cell.

// Equal returns true if the cell c is an errstr that wraps the same error.
func (e *T) Equal(c cell.T) bool {
	return Is(c) && *e == *To(c)
}

// Name returns the name of the errstr type.
func (e *T) Name() string {
	return name
}

// The errstr type is a boolean.

// Bool returns the boolean value errstr e.
func (e *T) Bool() bool {
	return e.String() == ""
}

// The errstr type has a literal representation.

// Literal returns the literal representation of the errstr e.
func (e *T) Literal() string {
	return "(|" + name + " " + adapted.CanonicalString(e.String()) + "|)"
}

// The errstr type is a stringer.

// String returns the text of the errstr e.
func (e *T) String() string {
	return string(*e)
}

// The two functions below could be generated for each type.

// Is returns true if c is a *T.
func Is(c cell.T) bool {
	_, ok := c.(*T)
	return ok
}

// To returns a *T if c is a *T; Otherwise is panics.
func To(c cell.T) *T {
	if e, ok := c.(*T); ok {
		return e
	}

	panic("not a " + name)
}
