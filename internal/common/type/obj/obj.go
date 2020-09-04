// Released under an MIT license. See LICENSE.

// Package obj provides oh's object type.
package obj

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/reference"
	"github.com/michaelmacinnis/oh/internal/common/interface/scope"
)

const name = "object"

// NOTE: This is not how objects were implemented in the previous version
// of oh but I think this might be closer to what we want. Shared behavior
// will need to be explicitly "pulled up" by associating a (public or
// private) name with a value in an enclosing scope. But the nice thing is
// that is will stop objects from being peepholes into the public scope
// of their creation. Or that is my current thinking. Let's see what I'm
// forgetting about that this breaks.

// T object limits access to top-level, public names in the wrapped env.
type T struct {
	wrapped
}

type obj = T

type wrapped = scope.I

// New creates a new obj.
func New(e scope.I) scope.I {
	return &obj{e}
}

// Clone creates a clone of the obj o.
func (o *obj) Clone() scope.I {
	return &obj{o.wrapped.Clone()}
}

// Define throws an error. Only public members of an obj can be added.
func (o *obj) Define(k string, v cell.I) {
	panic("private names cannot be added to object")
}

// Equal returns true if c is obj as o.
func (o *obj) Equal(c cell.I) bool {
	return Is(c) && o == To(c)
}

// Expose returns the wrapped env.
func (o *obj) Expose() scope.I {
	return o.wrapped
}

// Lookup retrieves the reference associated with the public name k in the obj o.
func (o *obj) Lookup(k string) reference.I {
	return o.wrapped.Public().Get(k)
}

// Name returns the type name for the obj o.
func (o *obj) Name() string {
	return name
}

// Remove frees the public name k from any association in the obj o.
func (o *obj) Remove(k string) bool {
	return o.wrapped.Public().Del(k)
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t obj

	// The obj type is a cell.
	_ = cell.I(&t)

	// The obj type is a scope.
	_ = scope.I(&t)
}
