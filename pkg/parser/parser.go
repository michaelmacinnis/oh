// Released under an MIT license. See LICENSE.

package parser

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
)

type parser struct {
	deref func(string, uintptr) Cell
}

func New(deref func(string, uintptr) Cell) *parser {
        return &parser{deref}
}

func (p *parser) Parse(
	input ReadStringer, t Thrower, filename string,
	yield func(Cell, string, int, string) (Cell, bool),
) bool {
	lexer := NewLexer(p.deref, input.ReadString, t.Throw, yield, filename)

	rval := 1
	for rval > 0 {
		lexer.clear()

		rval = ohParse(lexer)
	}

	return rval == 0
}

//go:generate ohyacc -o grammar.go -p oh -v /dev/null grammar.y
//go:generate go fmt grammar.go
