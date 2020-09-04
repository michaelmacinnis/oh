// Released under an MIT license. See LICENSE.

// Package errsys provides oh's system error type.
package errsys

import (
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/truth"
)

const name = "errsys"

// T (errsys) is used to pass an error where a cell is expected.
type T struct {
	error
}

type errsys = T

// New creates a new errsys to wrap the error err.
func New(err error) *errsys {
	return &errsys{err}
}

// Bool returns the boolean value of the errsys e.
func (e *errsys) Bool() bool {
	return false
}

// Equal returns true if the cell c is an errsys that wraps the same error.
func (e *errsys) Equal(c cell.I) bool {
	return Is(c) && e == To(c)
}

// Name returns the name of the errsys type.
func (e *errsys) Name() string {
	return name
}

// String returns the text of the errsys e.
func (e *errsys) String() string {
	return e.Err().Error()
}

// Methods specific to errsys.

// Err returns the wrapped error.
func (e *errsys) Err() error {
	return e.error
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t errsys

	// The errsys type is a cell.
	_ = cell.I(&t)

	// The errsys type is a stringer.
	_ = common.Stringer(&t)

	// The errsys type has a truth value.
	_ = truth.I(&t)
}
