// Released under an MIT license. See LICENSE.

// Package object provides oh's object type.
package object

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/reference"
	"github.com/michaelmacinnis/oh/internal/interface/scope"
	"github.com/michaelmacinnis/oh/internal/type/env"
)

const name = "object"

// NOTE: This is not how objects were implemented in the previous version
// of oh but I think this might be closer to what we want. Shared behavior
// will need to be explicitly "pulled up" by associating a (public or
// private) name with a value in an enclosing scope. But this nice thing is
// that is will stop objects from being peepholes into the public scope
// of their creation. Or that is my current thinking. Let's see what I'm
// forgetting about that this breaks.

// T object limits access to top-level, public names in the wrapped env.
type T struct {
	wrapped
}

type wrapped *env.T

// New creates a new obj.
func New(e *env.T) scope.T {
	return &T{e}
}

// Clone creates a clone of the obj o.
func (o *T) Clone() scope.T {
	return &T{o.wrapped.Copy()}
}

// Define throws an error. Only public members of an obj can be added.
func (o *T) Define(k string, v cell.T) {
	panic("private names cannot be added to object")
}

// Equal returns true if c is obj as o.
func (o *T) Equal(c cell.T) bool {
        return Is(c) && o == To(c)
}

// Expose returns the wrapped env.
func (o *T) Expose() scope.T {
	return o.wrapped
}

// Lookup retrieves the reference associated with the public name k in the obj o.
func (o *T) Lookup(k string) reference.T {
	return o.wrapped.Get(k)
}

// Name returns the type name for the obj o.
func (o *T) Name() string {
        return name
}

// Remove frees the public name k from any association in the obj o.
func (o *T) Remove(k string) bool {
	return o.wrapped.Unbind(k)
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
