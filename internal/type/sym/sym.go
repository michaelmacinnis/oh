// Released under an MIT license. See LICENSE.

// Package sym provides oh's symbol cell type.
package sym

import (
	"math/big"
	"sync"

	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/type/num"
)

const name = "symbol"

// T (symbol) wraps Go's string type. Short and common strings are interned.
type T string

// New creates a symbol cell.
func New(v string) cell.T {
	p, ok := symtry(v)
	if !ok {
		if len(v) <= 3 {
			syml.Lock()
			defer syml.Unlock()
			if p, ok = sym[v]; ok {
				return p
			}
		}

		s := T(v)
		p = &s

		if len(v) <= 3 {
			sym[v] = p
		}
	}

	return p
}

// The symbol type is a cell.

// Equal returns true if c is a symbol and wraps the same string.
func (s *T) Equal(c cell.T) bool {
	return Is(c) && s.String() == To(c).String()
}

// Name returns the type name for the symbol s.
func (s *T) Name() string {
	return name
}

// The symbol type has a literal representation.

// Literal returns the literal representation of the symbol s.
func (s *T) Literal() string {
	return string(*s)
}

// The symbol can be a rational.

// Rat returns the value of the symbol as a big.Rat, if possible.
func (s *T) Rat() *big.Rat {
	return num.New(s.Literal()).Rat()
}

// The symbol type is a stringer.

// String returns the text of the *T (sym) s.
func (s *T) String() string {
	return s.Literal()
}

// Functions specific to sym.

// Cache caches the specified symbols to reduce allocations.
func Cache(symbols ...string) {
	for _, v := range symbols {
		sym[v] = New(v)
	}
}

var (
	sym  = map[string]cell.T{} //nolint:gochecknoglobals
	syml = &sync.RWMutex{} //nolint:gochecknoglobals
)

func symtry(v string) (p cell.T, ok bool) {
	syml.RLock()
	defer syml.RUnlock()
	p, ok = sym[v]
	return
}

// The two functions below could be generated for each type.

// Is returns true if c is a *T.
func Is(c cell.T) bool {
	_, ok := c.(*T)
	return ok
}

// To returns a *T if c is a *T; Otherwise is panics.
func To(c cell.T) *T {
	if t, ok := c.(*T); ok {
		return t
	}

	panic("not a " + name)
}
