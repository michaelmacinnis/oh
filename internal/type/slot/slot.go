// Released under an MIT license. See LICENSE.

// Package slot provides oh's variable type.
package slot

import (
	"sync"

	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/reference"
)

// T (slot) holds a cell value.
type T struct {
	sync.RWMutex
	c cell.T
}

// New creates a new slot with the cell c.
func New(c cell.T) *T {
	return &T{c: c}
}

// Copy creates a new slot with the same cell as slot s.
func (s *T) Copy() reference.T {
	return New(s.Get())
}

// Get returns the cell in slot s.
func (s *T) Get() cell.T {
	s.RLock()
	defer s.RUnlock()

	return s.c
}

// Set replaces the cell in slot s with the cell c.
func (s *T) Set(c cell.T) {
	s.Lock()
	defer s.Unlock()

	s.c = c
}
