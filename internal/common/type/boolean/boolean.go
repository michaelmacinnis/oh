// Released under an MIT license. See LICENSE.

// Package boolean provides oh's boolean value type.
package boolean

import (
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/truth"
)

const name = "boolean"

// T (boolean) wraps Go's bool type.
type T bool

type boolean = T

//nolint:gochecknoglobals
var (
	False = f()
	True  = t()
)

// Bool creates new boolean from the bool b.
func Bool(b bool) cell.I {
	if b {
		return True
	}

	return False
}

func New(s string) cell.I {
	b, ok := map[string]*boolean{
		"true":  True,
		"false": False,
	}[s]

	if ok {
		return b
	}

	panic(s + " is not $'true' or $'false'")
}

// Bool returns the boolean value of the boolean b.
func (b *boolean) Bool() bool {
	return bool(*b)
}

// Equal returns true if c is a boolean with a matching value.
func (b *boolean) Equal(c cell.I) bool {
	return Is(c) && b.Bool() == To(c).Bool()
}

// Literal returns the literal representation of the boolean b.
func (b *boolean) Literal() string {
	return "(|" + name + " " + b.String() + "|)"
}

// Name returns the type name for the boolean b.
func (b *boolean) Name() string {
	return name
}

// String returns the text of the boolean b.
func (b *boolean) String() string {
	if bool(*b) {
		return "true"
	}

	return "false"
}

func f() *boolean {
	v := boolean(false)

	return &v
}

func t() *boolean {
	v := boolean(true)

	return &v
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t boolean

	// The boolean type is a cell.
	_ = cell.I(&t)

	// The boolean type has a literal representation.
	_ = literal.I(&t)

	// The boolean type is a stringer.
	_ = common.Stringer(&t)

	// The boolean type has a truth value.
	_ = truth.I(&t)
}
