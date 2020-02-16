// Released under an MIT license. See LICENSE.

// Package hash provides oh's name to value mapping type.
package hash

import (
	"sync"

	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/reference"
	"github.com/michaelmacinnis/oh/internal/type/slot"
)

// T (hash) maps names to values.
type T struct {
	sync.RWMutex
	m map[string]reference.T
}

// New creates a new hash.
func New() *T {
	return &T{m: map[string]reference.T{}}
}

// Copy creates a new hash with a copy of every reference.
func (h *T) Copy() *T {
	if h == nil {
		return nil
	}

	h.RLock()
	defer h.RUnlock()

	fresh := New()
	for k, v := range h.m {
		fresh.m[k] = v.Copy()
	}

	return fresh
}

// Del frees the name k from any association in the hash h.
func (h *T) Del(k string) bool {
	if h == nil {
		return false
	}

	h.Lock()
	defer h.Unlock()

	_, ok := h.m[k]
	if !ok {
		return false
	}

	delete(h.m, k)

	return true
}

// Get retrieves the reference associated with the name k in the hash h.
func (h *T) Get(k string) reference.T {
	if h == nil {
		return nil
	}

	h.RLock()
	defer h.RUnlock()

	return h.m[k]
}

// Add associates the name k with the cell v in the hash h.
func (h *T) Set(k string, v cell.T) {
	h.Lock()
	defer h.Unlock()

	h.m[k] = slot.New(v)
}

// Size returns the number of entries in the hash h.
func (h *T) Size() int {
	h.RLock()
	defer h.RUnlock()

	return len(h.m)
}
