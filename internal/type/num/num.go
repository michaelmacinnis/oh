// Released under an MIT license. See LICENSE.

// Package num provides oh's rational number type.
package num

import (
	"math/big"

	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

const name = "number"

// T (number) wraps Go's big.Rat type.
type T big.Rat

// New creates a new number from a string.
func New(s string) *T {
	v := &big.Rat{}

	if _, ok := v.SetString(s); !ok {
		panic("'" + s + "' is not a valid number")
	}

	return Rat(v)
}

// Rat creates wraps the *big.Rat r as a number.
func Rat(r *big.Rat) *T {
	return (*T)(r)
}

// The number type is a cell.

// Equal returns true if c is the same number as the number n.
func (n *T) Equal(c cell.T) bool {
	return Is(c) && n.Rat().Cmp(To(c).Rat()) == 0
}

// Name returns the type name for the number n.
func (n *T) Name() string {
	return name
}

// The number type is a boolean.

// Bool returns the boolean value of the number n.
func (n *T) Bool() bool {
	return n.Rat().Cmp(&big.Rat{}) != 0
}

// The number type has a literal representation.

// Literal returns the literal representation of the number n.
func (n *T) Literal() string {
	return "(|" + name + " " + n.String() + "|)"
}

// The number type is a rational.

// Rat returns the value of the number n as a *big.Rat.
func (n *T) Rat() *big.Rat {
	return (*big.Rat)(n)
}

// The number type is a stringer.

// String returns the text of the number n.
func (n *T) String() string {
	return n.Rat().RatString()
}

// The two functions below could be generated for each type.

// Is returns true if c is a *T.
func Is(c cell.T) bool {
	_, ok := c.(*T)
	return ok
}

// To returns a *T if c is a *T; Otherwise it panics.
func To(c cell.T) *T {
	if n, ok := c.(*T); ok {
		return n
	}

	panic("not a " + name)
}
