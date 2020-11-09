// Released under an MIT license. See LICENSE.

// Package token is shared by the oh lexer and parser.
package token

import (
	"strconv"
	"unicode"

	"github.com/michaelmacinnis/oh/internal/common/struct/loc"
)

// Class is a token's type.
type Class rune

// T (token) is a lexical item returned by the scanner.
type T struct {
	class  Class
	source *loc.T
	value  string
}

type token = T

// Token classes.
const (
	Error Class = iota

	Andf Class = unicode.MaxRune + iota
	Background
	DollarSingleQuoted
	DoubleQuoted
	MetaClose
	MetaOpen
	Orf
	Pipe
	Redirect
	SingleQuoted
	Space
	Substitute
	Symbol
)

// New creates a new token.
func New(class Class, value string, source *loc.T) *token {
	return &token{
		class:  class,
		source: source,
		value:  value,
	}
}

// String returns a string representation of Class. Useful for debugging.
func (c *Class) String() string {
	switch *c {
	case Error:
		return "Error"
	case Andf:
		return "Andf"
	case Background:
		return "Background"
	case DollarSingleQuoted:
		return "DollarSingleQuoted"
	case DoubleQuoted:
		return "DoubleQuoted"
	case MetaClose:
		return "MetaClose"
	case MetaOpen:
		return "MetaOpen"
	case Orf:
		return "Orf"
	case Pipe:
		return "Pipe"
	case Redirect:
		return "Redirect"
	case SingleQuoted:
		return "SingleQuoted"
	case Space:
		return "Space"
	case Substitute:
		return "Substitute"
	case Symbol:
		return "Symbol"
	}

	return strconv.QuoteRune(rune(*c))
}

// Is returns true if the token t is any of the classes in cs.
func (t *token) Is(cs ...Class) bool {
	if t == nil {
		return false
	}

	for _, c := range cs {
		if t.class == c {
			return true
		}
	}

	return false
}

// Source returns the source location for this token.
func (t *token) Source() *loc.T {
	return t.source
}

// String returns the token's string representation. Useful for debugging.
func (t *token) String() string {
	return strconv.Quote(t.value) + "(" +
		t.class.String() + "," +
		t.source.String() + ")"
}

// Value returns the token's string value.
func (t *token) Value() string {
	return t.value
}
