// Released under an MIT license. See LICENSE.

package parser

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
)

type parser struct {
	*ohParserImpl
	*lexer
}

type template struct {
	deref func(string, uintptr) Cell
}

func Template(deref func(string, uintptr) Cell) *template {
        return &template{deref}
}

func (t *template) MakeParser(
	input ReadStringer, thrower Thrower, filename string,
	yield func(Cell, string, int, string) (Cell, bool),
) Parser {
	return &parser{
		&ohParserImpl{},
		NewLexer(
			t.deref,
			input.ReadString,
			thrower.Throw,
			yield,
			filename,
		),
	}
}

func (p *parser) Start() bool {
	rval := 1
	for rval > 0 {
		p.clear()

		rval = p.Parse(p.lexer)
	}

	return rval == 0
}

//go:generate ohyacc -o grammar.go -p oh -v /dev/null grammar.y
//go:generate go fmt grammar.go
