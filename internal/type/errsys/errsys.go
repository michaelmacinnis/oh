// Released under an MIT license. See LICENSE.

// Package errsys provides oh's system error type.
package errsys

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

const name = "errsys"

// T (errsys) is used to pass an error where a cell is expected.
type T struct {
	error
}

// New creates a new errsys to wrap the error err.
func New(err error) *T {
	return &T{err}
}

// The errsys type is a cell.

// Equal returns true if the cell c is an errsys that wraps the same error.
func (e *T) Equal(c cell.T) bool {
	return Is(c) && e == To(c)
}

// Name returns the name of the errsys type.
func (e *T) Name() string {
	return name
}

// The errsys type is a boolean.

// Bool returns the boolean value of the errsys e.
func (e *T) Bool() bool {
	return false
}

// The errsys type is a stringer.

// String returns the text of the errsys e.
func (e *T) String() string {
	return e.Err().Error()
}

// Methods specific to errsys.

// Err returns the wrapped error.
func (e *T) Err() error {
	return e.error
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
