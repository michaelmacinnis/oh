// Released under an MIT license. See LICENSE.

// Package num provides oh's rational number type.
package num

import (
	"math/big"

	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/interface/truth"
)

const name = "number"

// T (num) wraps Go's big.Rat type.
type T big.Rat

type num = T

// New creates a new num cell from a string.
func New(s string) cell.I {
	return Num(s)
}

// Num creates a new num from a string.
func Num(s string) cell.I {
	v := &big.Rat{}

	if _, ok := v.SetString(s); !ok {
		panic("'" + s + "' is not a valid number")
	}

	return Rat(v)
}

// Int creates a num from the integer i.
func Int(i int) cell.I {
	return Rat(big.NewRat(int64(i), 1))
}

// Rat creates wraps the *big.Rat r as a num.
func Rat(r *big.Rat) cell.I {
	return (*num)(r)
}

// Bool returns the boolean value of the num n.
func (n *num) Bool() bool {
	return n.Rat().Cmp(&big.Rat{}) == 0
}

// Equal returns true if c is the same number as the num n.
func (n *num) Equal(c cell.I) bool {
	return Is(c) && n.Rat().Cmp(To(c).Rat()) == 0
}

// Literal returns the literal representation of the num n.
func (n *num) Literal() string {
	return "(|" + name + " " + n.String() + "|)"
}

// Name returns the type name for the num n.
func (n *num) Name() string {
	return name
}

// Rat returns the value of the num n as a *big.Rat.
func (n *num) Rat() *big.Rat {
	return (*big.Rat)(n)
}

// String returns the text of the num n.
func (n *num) String() string {
	return n.Rat().RatString()
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t num

	// The num type is a cell.
	_ = cell.I(&t)

	// The num type has a literal representation.
	_ = literal.I(&t)

	// The num type is a rational.
	_ = rational.I(&t)

	// The num type is a stringer.
	_ = common.Stringer(&t)

	// The num type has a truth value.
	_ = truth.I(&t)
}
