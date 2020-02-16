// Released under an MIT license. See LICENSE.

// Package cell defines the interface for all oh types.
package cell

// T (cell) is the basic unit of storage in oh.
type T interface {
	Equal(c T) bool
	Name() string
}
