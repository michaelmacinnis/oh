// Released under an MIT license. See LICENSE.

// Package conduit defines the interface for oh channels and pipes.
package conduit

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// I (conduit) is the interface oh channels and pipes satisfy.
type I interface {
	cell.I

	Close()
	Read() cell.I
	ReadLine() cell.I
	ReaderClose()
	Write(v cell.I)
	WriterClose()
}

type conduit = I

// Is returns true if c is an I.
func Is(c cell.I) bool {
	_, ok := c.(conduit)

	return ok
}

// To returns a I if c is a I; Otherwise it panics.
func To(c cell.I) I {
	if t, ok := c.(conduit); ok {
		return t
	}

	panic("not a conduit")
}
