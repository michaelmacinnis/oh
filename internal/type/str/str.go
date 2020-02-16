// Released under an MIT license. See LICENSE.

// Package str provides oh's string type.
package str

import (
	"github.com/michaelmacinnis/oh/internal/adapted"
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

const name = "string"

// T (string) wraps Go's string type.
type T string

// New creates a new string cell.
func New(v string) cell.T {
	s := T(v)
	return &s
}

// The string type is a cell.

// Equal returns true if the cell c wraps the same string and false otherwise.
func (s *T) Equal(c cell.T) bool {
	return Is(c) && s.String() == To(c).String()
}

// Name returns the name of the string type.
func (s *T) Name() string {
	return name
}

// The string type is a boolean.

// Bool returns the boolean value of the string s.
func (s *T) Bool() bool {
	return s.String() != ""
}

// The string type has a literal representation.

// Literal returns the literal representation of the string s.
func (s *T) Literal() string {
	return adapted.CanonicalString(string(*s))
}

// The string type is a stringer.

// String returns the text of the string s.
func (s *T) String() string {
	return string(*s)
}

// The two functions below could be generated for each type.

// Is returns true if c is a *T.
func Is(c cell.T) bool {
	_, ok := c.(*T)
	return ok
}

// To returns a *T if c is a *T; Otherwise is panics.
func To(c cell.T) *T {
	if t, ok := c.(*T); ok {
		return t
	}

	panic("not a " + name)
}
