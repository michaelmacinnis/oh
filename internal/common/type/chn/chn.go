// Released under an MIT license. See LICENSE.

// Package chn provides oh's channel type.
package chn

import (
	"fmt"

	"github.com/michaelmacinnis/adapted"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/conduit"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
)

const name = "chan"

// T (chn) is oh's channel conduit type.
type T chan cell.I

type chn = T

// New creates a new chn cell.
func New(cap int64) cell.I {
	c := chn(make(chan cell.I, cap))

	return &c
}

// Close closes the chn.
func (c *chn) Close() {
	c.WriterClose()
}

// Equal returns true if the cell c is the same chn and false otherwise.
func (c *chn) Equal(v cell.I) bool {
	return Is(v) && c == To(v)
}

// Name returns the name of the chn type.
func (*chn) Name() string {
	return name
}

// Read reads a cell from the chn.
func (c *chn) Read() cell.I {
	v := <-*c
	if v == nil {
		return pair.Null
	}

	return v
}

// ReadLine reads a line from the chn.
func (c *chn) ReadLine() cell.I {
	v := <-*c
	if v == nil {
		return pair.Null
	}

	i, ok := v.(fmt.Stringer)
	if ok {
		s, err := adapted.ActualBytes(i.String())
		if err == nil {
			return str.New(s)
		}
	}

	return str.New(literal.String(v))
}

// ReaderClose is a no-op for a chn.
func (c *chn) ReaderClose() {}

// Write writes a cell to the chn.
func (c *chn) Write(v cell.I) {
	*c <- v
}

// WriteLine writes a cell to the chn.
func (c *chn) WriteLine(v cell.I) {
	*c <- v
}

// WriterClose closes the chn.
func (c *chn) WriterClose() {
	close(*c)
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t chn

	// The chn type is a cell.
	_ = cell.I(&t)

	// The chn type is a conduit.
	_ = conduit.I(&t)
}
