// Released under an MIT license. See LICENSE.

// Package pair provides oh's cons cell type.
package pair

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/literal"
	"github.com/michaelmacinnis/oh/internal/type/loc"
)

const name = "cons"

var (
	// Null is the empty list. It is also used to mark the end of a list.
	Null cell.T //nolint:gochecknoglobals
)

// T (pair) is a cons cell.
type T struct {
	car cell.T
	cdr cell.T
}

// Plus is a pair but with contextual information.
type Plus struct {
	*T
	source loc.T
}

// The pair type is a cell.

// Equal returns true if c is a pair with elements that are equal to p's.
func (p *T) Equal(c cell.T) bool {
	if p == Null && c == Null {
		return true
	}
	return p.car.Equal(Car(c)) && p.cdr.Equal(Cdr(c))
}

// Name returns the name for a pair type.
func (p *T) Name() string {
	return name
}

// The pair type is a boolean.

// Bool returns the boolean value of the pair p.
func (p *T) Bool() bool {
	return p != Null
}

// The pair type has a literal representation.

// Literal returns the literal representation of the pair p.
func (p *T) Literal() string {
	s := ""

	improper := false

	tail := Cdr(p)
	if !Is(tail) {
		improper = true
		s += "(|" + name + " "
	}

	sublist := false

	head := Car(p)
	if Is(head) && Is(Cdr(head)) {
		sublist = true
		s += "("
	}

	if head == nil {
		s += "(|nil|)"
	} else if head != Null {
		s += literal.String(head)
	}

	if sublist {
		s += ")"
	}

	if !improper && tail == Null {
		return s
	}

	s += " "
	if tail == nil {
		s += "(|nil|)"
	} else {
		s += literal.String(tail)
	}

	if improper {
		s += "|)"
	}

	return s
}

// The pair type is a stringer.

// String returns the text representation of the pair p.
func (p *T) String() string {
	return p.Literal()
}

// Source returns the lexical location for a pair plus.
func (p *Plus) Source() *loc.T {
	return &p.source
}

// Functions specific to pair.

// Car returns the car/head/first member of the pair c.
// If c is not a pair, this function will panic.
func Car(c cell.T) cell.T {
	return To(c).car
}

// Cdr returns the cdr/tail/rest member of the pair c.
// If c is not a pair, this function will panic.
func Cdr(c cell.T) cell.T {
	return To(c).cdr
}

// Caar returns the car of the car of the pair c.
// A non-pair value where a pair is expected will cause a panic.
func Caar(c cell.T) cell.T {
	return To(To(c).car).car
}

// Cadr returns the car of the cdr of the pair c.
// A non-pair value where a pair is expected will cause a panic.
func Cadr(c cell.T) cell.T {
	return To(To(c).cdr).car
}

// Cdar returns the cdr of the car of the pair c.
// A non-pair value where a pair is expected will cause a panic.
func Cdar(c cell.T) cell.T {
	return To(To(c).car).cdr
}

// Cddr returns the cdr of the cdr of the pair c.
// A non-pair value where a pair is expected will cause a panic.
func Cddr(c cell.T) cell.T {
	return To(To(c).cdr).cdr
}

// Caddr returns the car of the cdr of the cdr of the pair c.
// A non-pair value where a pair is expected will cause a panic.
func Caddr(c cell.T) cell.T {
	return To(To(To(c).cdr).cdr).car
}

// Cons conses h and t together to form a new pair.
// If a source location is provided that contextual information is added.
func Cons(h, t cell.T, source ...loc.T) cell.T {
	p := &T{car: h, cdr: t}

	length := len(source)
	if length == 0 {
		return p
	}

	if length > 1 {
		panic("cons can't have more than one source")
	}

	return &Plus{T: p, source: source[0]}
}

// Is returns true if c is a pair or pair plus.
func Is(c cell.T) bool {
	switch c.(type) {
	case *T, *Plus:
		return true
	}
	return false
}

// IsNull returns true if c is the Null cell.
func IsNull(c cell.T) bool {
	return c == Null
}

// SetCar sets the car/head/first of the pair c to value.
// If c is not a pair, this function will panic.
func SetCar(c, value cell.T) {
	To(c).car = value
}

// SetCdr sets the cdr/tail/rest of the pair c to value.
// If c is not a pair, this function will panic.
func SetCdr(c, value cell.T) {
	To(c).cdr = value
}

// To returns a pair if c is a pair or pair plus; Otherwise it panics.
func To(c cell.T) *T {
	switch t := c.(type) {
	case *T:
		return t
	case *Plus:
		return t.T
	}

	panic("not a " + name + " cell")
}

//nolint:gochecknoinits
func init() {
	pair := &T{}
	pair.car = pair
	pair.cdr = pair

	Null = cell.T(pair)
}
