// Released under an MIT license. See LICENSE.

// Package frame provides oh's call stack frame type.
package frame

import (
	"github.com/michaelmacinnis/oh/internal/interface/reference"
	"github.com/michaelmacinnis/oh/internal/interface/scope"
	"github.com/michaelmacinnis/oh/internal/type/loc"
)

// T (frame) is stack frame or activation record.
type T struct {
        previous *T
        scope    scope.T
        source   loc.T
}

// New creates a new frame.
func New(s scope.T) *T {
	return &T{scope: s}
}

func (f *T) New() *T {
	return &T{previous: f, source: f.source}
}

// Resolve looks for a lexical and then dynamic resolution of k.
// The scope where the reference r was found is also returned.
func (f *T) Resolve(k string) (s scope.T, r reference.T) {
	s = f.scope
	r = s.Lookup(k)

	for f = f.previous; f != nil && r == nil; f = f.previous {
		s = f.scope
		r = s.Public(k)
	}

	return
}

// Scope returns the current frame's scope.
func (f *T) Scope() scope.T {
	return f.scope
}

// SetScope sets the current frame's scope.
func (f *T) SetScope(s scope.T) {
	f.scope = s
}

// Update sets the current lexical location.
func (f *T) Update(source *loc.T) {
	f.source = *source
}
