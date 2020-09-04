// Released under an MIT license. See LICENSE.

// Package env provides oh's first-class environment type.
package env

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/reference"
	"github.com/michaelmacinnis/oh/internal/common/interface/scope"
	"github.com/michaelmacinnis/oh/internal/common/struct/hash"
)

const name = "environment"

// T (env) provides a public and private mapping of names to values.
type T struct {
	previous scope.I
	private  *hash.T
	*public
}

type env = T

// We alias hash.T to public so that when embedded it is easy to refer to
// it by name. Embedding public also lets us access its methods directly.
type public = hash.T

// New creates a new env.
func New(previous scope.I) scope.I {
	return &env{
		previous: previous,
		private:  hash.New(),
		public:   hash.New(),
	}
}

// Clone creates a clone of the current scope.
func (e *env) Clone() scope.I {
	return &env{
		previous: e.previous,
		private:  e.private.Copy(),
		public:   e.public.Copy(),
	}
}

// Define associates the private name k with the cell v in the env e.
func (e *env) Define(k string, v cell.I) {
	e.private.Set(k, v)
}

// Enclosing returns the enclosing scope.
func (e *env) Enclosing() scope.I {
	return e.previous
}

// Equal returns true if c is the same env as e.
func (e *env) Equal(c cell.I) bool {
	return Is(c) && e == To(c)
}

// Export associates the public name k with the cell v in the env e.
func (e *env) Export(k string, v cell.I) {
	e.Set(k, v)
}

// Exported returns the number of exported variables.
func (e *env) Exported() int {
	return e.Size()
}

// Expose returns a scope with public and private members visible.
func (e *env) Expose() scope.I {
	return e
}

// Lookup retrieves the reference associated with the name k in the env e.
func (e *env) Lookup(k string) reference.I {
	if e == nil {
		return nil
	}

	v := e.private.Get(k)

	if v == nil {
		v = e.Get(k)
	}

	if v == nil && e.previous != nil {
		v = e.previous.Lookup(k)
	}

	return v
}

// Name returns the type name for the env e.
func (e *env) Name() string {
	return name
}

// Public returns the public hash for the env e.
func (e *env) Public() *hash.T {
	return e.public
}

// Remove deletes the name k from the env e.
func (e *env) Remove(k string) bool {
	if e == nil {
		return false
	}

	return e.private.Del(k) || e.Del(k) || e.Enclosing().Remove(k)
}

// Visible returns true if exported variables in o are visible in e.
func (e *env) Visible(o scope.I) bool {
	for o != nil && o.Exported() == 0 {
		o = o.Enclosing()
	}

	if o == nil {
		return true
	}

	p := o.Expose()

	o = e
	for o != nil && o.Exported() == 0 {
		o = o.Enclosing()
	}

	if o == nil {
		return false
	}

	return p == o.Expose()
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t env

	// The env type is a cell.
	_ = cell.I(&t)

	// The env type is a scope.
	_ = scope.I(&t)
}
