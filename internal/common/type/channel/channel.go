// Released under an MIT license. See LICENSE.

// Package channel provides oh's channel type.
package channel

import (
	"github.com/michaelmacinnis/oh/internal/adapted"
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/conduit"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
)

const name = "channel"

// T (channel) is oh's channel conduit type.
type T chan cell.I

type channel = T

// New creates a new channel cell.
func New(cap int64) cell.I {
	c := channel(make(chan cell.I, cap))

	return &c
}

// Close closes the channel.
func (c *channel) Close() {
	c.WriterClose()
}

// Equal returns true if the cell c is the same channel and false otherwise.
func (c *channel) Equal(v cell.I) bool {
	return Is(v) && c == To(v)
}

// Name returns the name of the channel type.
func (*channel) Name() string {
	return name
}

// Read reads a cell from the channel.
func (c *channel) Read() cell.I {
	v := <-*c
	if v == nil {
		return pair.Null
	}

	return v
}

// ReadLine reads a line from the channel.
func (c *channel) ReadLine() cell.I {
	v := <-*c
	if v == nil {
		return pair.Null
	}

	i, ok := v.(common.Stringer)
	if ok {
		s, err := adapted.ActualBytes(i.String())
		if err == nil {
			return str.New(s)
		}
	}

	return str.New(literal.String(v))
}

// ReaderClose is a no-op for a channel.
func (c *channel) ReaderClose() {}

// Write writes a cell to the channel.
func (c *channel) Write(v cell.I) {
	*c <- v
}

// WriteLine writes a cell to the channel.
func (c *channel) WriteLine(v cell.I) {
	*c <- v
}

// WriterClose closes the channel.
func (c *channel) WriterClose() {
	close(*c)
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t channel

	// The channel type is a cell.
	_ = cell.I(&t)

	// The channel type is a conduit.
	_ = conduit.I(&t)
}
