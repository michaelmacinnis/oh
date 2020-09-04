// Released under an MIT license. See LICENSE.

// Package slot provides oh's variable type.
package slot

import (
	"sync"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/reference"
)

// T (slot) holds a cell value.
type T struct {
	sync.RWMutex
	c cell.I
}

type slot = T

// New creates a new slot with the cell c.
func New(c cell.I) *slot {
	return &slot{c: c}
}

// Copy creates a new slot with the same cell as slot s.
func (s *slot) Copy() reference.I {
	return New(s.Get())
}

// Get returns the cell in slot s.
func (s *slot) Get() cell.I {
	s.RLock()
	defer s.RUnlock()

	return s.c
}

// Set replaces the cell in slot s with the cell c.
func (s *slot) Set(c cell.I) {
	s.Lock()
	defer s.Unlock()

	s.c = c
}
