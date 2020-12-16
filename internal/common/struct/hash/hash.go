// Released under an MIT license. See LICENSE.

// Package hash provides oh's name to value mapping type.
package hash

import (
	"fmt"
	"sync"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/reference"
	"github.com/michaelmacinnis/oh/internal/common/struct/slot"
)

// T (hash) maps names to values.
type T struct {
	sync.RWMutex
	m map[string]reference.I
}

type hash = T

// New creates a new hash.
func New() *hash {
	return &hash{m: map[string]reference.I{}}
}

// Copy creates a new hash with a copy of every reference.
func (h *hash) Copy() *hash {
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
func (h *hash) Del(k string) bool {
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

// Exported returns a map with containing all entries in h with a string value.
func (h *hash) Exported() map[string]string {
	h.Lock()
	defer h.Unlock()

	exported := map[string]string{}

	for k, v := range h.m {
		if s, ok := v.Get().(fmt.Stringer); ok {
			exported[k] = s.String()
		}
	}

	return exported
}

// Get retrieves the reference associated with the name k in the hash h.
func (h *hash) Get(k string) reference.I {
	if h == nil {
		return nil
	}

	h.RLock()
	defer h.RUnlock()

	return h.m[k]
}

// Set associates the name k with the cell v in the hash h.
func (h *hash) Set(k string, v cell.I) {
	h.Lock()
	defer h.Unlock()

	h.m[k] = slot.New(v)
}

// Size returns the number of entries in the hash h.
func (h *hash) Size() int {
	h.RLock()
	defer h.RUnlock()

	return len(h.m)
}
