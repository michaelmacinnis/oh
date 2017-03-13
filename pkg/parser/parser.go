// Released under an MIT license. See LICENSE.

package parser

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"fmt"
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
	input ReadStringer, filename string,
	yield func(Cell, string, int, string) (Cell, bool),
) Parser {
	return &parser{
		&ohParserImpl{},
		NewLexer(
			t.deref,
			input.ReadString,
			yield,
			filename,
		),
	}
}

func (p *parser) NewStart() (rval int, pe *ParseError) {
	pe = nil

	defer func() {
		r := recover()
		if r != nil {
			pe = &ParseError{
				Filename: p.lexer.label,
				LineNumber: p.lexer.lines,
				Message: fmt.Sprintf("%v", r),
			}
		}
	}()

	return p.Parse(p.lexer), pe
}

func (p *parser) Start(thrower Thrower) bool {
	for {
		rval, pe := p.NewStart()
		if pe != nil {
			thrower.Throw(pe.Filename, pe.LineNumber, pe.Message)
		} else if rval <= 0 {
			return rval == 0
		}

		l := p.lexer

		p.lexer = NewLexer(l.deref, l.input, l.yield, l.label)
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

	return Raw(Car(lcopy.first)), lcopy.state.n, completing
}

//go:generate ohyacc -o grammar.go -p oh -v /dev/null grammar.y
//go:generate go fmt grammar.go
