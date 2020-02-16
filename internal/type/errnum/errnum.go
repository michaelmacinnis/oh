// Released under an MIT license. See LICENSE.

// Package errnum provides oh's error number type.
package errnum

import (
	"math/big"

	"github.com/michaelmacinnis/oh/internal/interface/boolean"
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/literal"
	"github.com/michaelmacinnis/oh/internal/interface/rational"
)

const name = "errnum"

// T (errnum) wraps Go's big.Rat type.
type T big.Rat

// New creates a new errnum from a string.
func New(s string) *T {
	v := &big.Rat{}

	if _, ok := v.SetString(s); !ok {
		panic("'" + s + "' is not a valid number")
	}

	return Rat(v)
}

// Rat wraps the *big.Rat r as a errnum.
func Rat(r *big.Rat) *T {
	return (*T)(r)
}

// The errnum type is a cell.

// Equal returns true if c is the same errnum as the errnum n.
func (n *T) Equal(c cell.T) bool {
	return Is(c) && n.Rat().Cmp(To(c).Rat()) == 0
}

// Name returns the type name for the errnum n.
func (n *T) Name() string {
	return name
}

// The errnum type is a boolean.

// Bool returns the boolean value of the number n.
func (n *T) Bool() bool {
	return n.Rat().Cmp(&big.Rat{}) == 0
}

// The errnum type has a literal representation.

// Literal returns the literal representation of the errnum n.
func (n *T) Literal() string {
	return "(|" + name + " " + n.String() + "|)"
}

// The errnum type is a rational.

// Rat returns the value of the errnum n as a *big.Rat.
func (n *T) Rat() *big.Rat {
	return (*big.Rat)(n)
}

// The errnum type is a stringer.

// String returns the text of the errnum n.
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

func implements() { //nolint:deadcode,unused
	// This function is never called. Its purpose is as compiler-checked
	// documentation about the interfaces this type satisfies.
	var e T

	var c cell.T = &e
	_ = c
	var b boolean.T = &e
	_ = b
	var l literal.T = &e
	_ = l
	var r rational.T = &e
	_ = r
}
