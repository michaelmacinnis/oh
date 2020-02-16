// Released under an MIT license. See LICENSE.

// Package env provides oh's first-class environment type.
package env

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/reference"
	"github.com/michaelmacinnis/oh/internal/interface/scope"
	"github.com/michaelmacinnis/oh/internal/type/hash"
)

const name = "environment"

// T (env) provides a public and private mapping of names to values.
type T struct {
	previous scope.T
	private  *hash.T
	*public
}

// We alias hash.T to public so that when embedded it is easy to refer to
// it by name. Embedding public also lets us access its methods directly.
type public = hash.T

// New creates a new env.
func New(previous scope.T) scope.T {
	return &T{
		previous: previous,
		private:  hash.New(),
		public:   hash.New(),
	}
}

// Clone creates a clone of the current scope.
func (e *T) Clone() scope.T {
	return &T{
		previous: e.previous,
		private:  e.private.Copy(),
		public:   e.public.Copy(),
	}
}

// Define associates the private name k with the cell v in the env e.
func (e *T) Define(k string, v cell.T) {
	e.private.Set(k, v)
}

// Equal returns true if c is the same env as e.
func (e *T) Equal(c cell.T) bool {
        return Is(c) && e == To(c)
}

// Export associates the public name k with the cell v in the env e.
func (e *T) Export(k string, v cell.T) {
	e.Set(k, v)
}

// Expose returns a scope with public and private members visible.
func (e *T) Expose() scope.T {
	return e
}

// Lookup retrieves the reference associated with the name k in the env e.
func (e *T) Lookup(k string) reference.T {
	if e == nil {
		return nil
	}

	v := e.Get(k)

	if v == nil {
		v = e.private.Get(k)
	}

	if v == nil && e.previous != nil {
		v = e.previous.Lookup(k)
	}

	return v
}

// Public retrieves the reference associated with the public name k in the env e.
func (e *T) Public(k string) reference.T {
	v := e.Get(k)
	if v != nil {
		return v
	}

	return e.previous.Public(k)
}

// Name returns the type name for the env e.
func (e *T) Name() string {
        return name
}

// Remove deletes the name k from the env e.
func (e *T) Remove(k string) bool {
	if e == nil {
		return false
	}

	return e.Del(k) || e.private.Del(k) || e.previous.Remove(k)
}

// Exported returns the number of exported variables.
func (e *T) Exported() int {
	return e.Size()
}

// Enclosing returns the enclosing scope.
func (e *T) Enclosing() scope.T {
	return e.previous
}

// Visible returns true if exported variables in o are visible in e.
func (e *T) Visible(o scope.T) bool {
	for o != nil && o.Exported() == 0 {
		o = o.Enclosing()
	}

	if o == nil {
		return true
	}

	p := o.Expose()

	o = e.Expose()
	for o != nil && o != p {
		o = o.Enclosing().Expose()
	}

	return o == p
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
