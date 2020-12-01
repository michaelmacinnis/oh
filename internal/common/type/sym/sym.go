// Released under an MIT license. See LICENSE.

// Package sym provides oh's symbol cell type.
package sym

import (
	"math/big"
	"sync"

	"github.com/michaelmacinnis/oh/internal/adapted"
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/type/num"
)

const (
	name  = "symbol"
	short = 3
)

// T (sym) wraps Go's string type. Short and common strings are interned.
type T string

type sym = T

// New creates a sym cell.
func New(v string) cell.I {
	return symnew(v)
}

// Equal returns true if c is a sym and wraps the same string.
func (s *sym) Equal(c cell.I) bool {
	return Is(c) && s.String() == To(c).String()
}

// Literal returns the literal representation of the sym s.
func (s *sym) Literal() string {
	return repr(string(*s))
}

// Name returns the type name for the sym s.
func (s *sym) Name() string {
	return name
}

// Rat returns the value of the sym as a big.Rat, if possible.
func (s *sym) Rat() *big.Rat {
	return rational.Number(num.Num(s.Literal()))
}

// String returns the text of the sym s.
func (s *sym) String() string {
	return string(*s)
}

// Functions specific to sym.

// Cache caches the specified sym to reduce allocations.
func Cache(symbols ...string) {
	for _, v := range symbols {
		cache[v] = symnew(v)
	}
}

//nolint:gochecknoglobals
var (
	cache  = map[string]*sym{}
	cachel = &sync.RWMutex{}
)

func meta(s string) string {
	return "(|" + name + " " + s + "|)"
}

func repr(s string) string {
	q := adapted.CanonicalString(s)

	if len(s) == 0 {
		return meta(q)
	}

	for _, r := range s {
		if r == ' ' {
			return meta(q)
		}
	}

	if q[2:len(q)-1] != s {
		return meta(q)
	}

	return s
}

func symnew(v string) *sym {
	p, ok := symtry(v)
	if !ok {
		if len(v) <= short {
			cachel.Lock()
			defer cachel.Unlock()

			if p, ok = cache[v]; ok {
				return p
			}
		}

		s := sym(v)
		p = &s

		if len(v) <= short {
			cache[v] = p
		}
	}

	return p
}

func symtry(v string) (p *sym, ok bool) {
	cachel.RLock()
	defer cachel.RUnlock()

	p, ok = cache[v]

	return
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t sym

	// The sym type is a cell.
	_ = cell.I(&t)

	// The sym type has a literal representation.
	_ = literal.I(&t)

	// The sym type is a stringer.
	_ = common.Stringer(&t)
}
