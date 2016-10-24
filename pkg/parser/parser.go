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
	for {
		rval := p.Parse(p.lexer)
		if rval <= 0 {
			return rval == 0
		}

		l := p.lexer

		p.lexer = NewLexer(l.deref, l.input, l.throw, l.yield, l.label)
		p.lexer.lines = l.lines

		p.ohParserImpl = &ohParserImpl{}
	}
}

func (p *parser) State(line string) (string, string, string) {
	pcopy := *p.ohParserImpl
	lcopy := p.lexer.Partial(line)

	pcopy.Parse(lcopy)

	completing := ""
	if lcopy.start < len(lcopy.bytes) {
		completing = lcopy.bytes[lcopy.start:]
	}

	return lcopy.first, lcopy.state.n, completing
}

//go:generate ohyacc -o grammar.go -p oh -v /dev/null grammar.y
//go:generate go fmt grammar.go
