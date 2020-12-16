// Released under an MIT license. See LICENSE.

// Package status provides oh's numeric exit status type.
package status

import (
	"fmt"
	"math/big"

	"github.com/michaelmacinnis/oh/internal/common/interface/boolean"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
)

const name = "status"

// T (status) is oh's numeric status type.
type T big.Rat

type status = T

// Int creates a status from the integer i.
func Int(i int) cell.I {
	return Rat(big.NewRat(int64(i), 1))
}

// New creates a new status cell from a string.
func New(s string) cell.I {
	v := &big.Rat{}

	if _, ok := v.SetString(s); !ok {
		panic("'" + s + "' is not a valid number")
	}

	return Rat(v)
}

// Rat creates wraps the *big.Rat r as a num.
func Rat(r *big.Rat) cell.I {
	return (*status)(r)
}

// Bool returns the boolean value of the status s.
func (s *status) Bool() bool {
	return s.Rat().Cmp(&big.Rat{}) == 0
}

// Equal returns true if c is the same number as the status s.
func (s *status) Equal(c cell.I) bool {
	return Is(c) && s.Rat().Cmp(To(c).Rat()) == 0
}

// Literal returns the literal representation of the status s.
func (s *status) Literal() string {
	return "(|" + name + " " + s.String() + "|)"
}

// Rat returns the value of the status s as a *big.Rat.
func (s *status) Rat() *big.Rat {
	return (*big.Rat)(s)
}

// Name returns the type name for the status s.
func (s *status) Name() string {
	return name
}

// String returns the text of the status s.
func (s *status) String() string {
	return s.Rat().RatString()
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t status

	// The status type has a boolean value.
	_ = boolean.I(&t)

	// The status type is a cell.
	_ = cell.I(&t)

	// The status type has a literal representation.
	_ = literal.I(&t)

	// The status type is a rational.
	_ = rational.I(&t)

	// The status type is a stringer.
	_ = fmt.Stringer(&t)
}
