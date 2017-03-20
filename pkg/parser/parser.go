// Released under an MIT license. See LICENSE.

package parser

import (
	"fmt"
	. "github.com/michaelmacinnis/oh/pkg/cell"
)

type parser struct {
	*ohParserImpl
	*lexer
}

func New(
	engine Engine, input ReadStringer,
	yield func(Cell, string, int) (Cell, bool),
) Parser {
	return &parser{
		&ohParserImpl{},
		NewLexer(
			engine.Deref,
			input.ReadString,
			0,
			yield,
		),
	}
}

func (p *parser) ParseBuffer(label string) bool {
	for {
		rval, e := p.ParsePipe(label)
		if e != nil {
			c := List(
				NewSymbol("throw"), List(
					NewSymbol("_exception"),
					NewSymbol("error/syntax"),
					NewStatus(NewSymbol("1").Status()),
					NewSymbol(fmt.Sprintf("%v", e)),
					NewInteger(int64(p.lexer.lines)),
					NewSymbol(label),
				),
			)
			p.lexer.yield(c, label, p.lexer.lines)
		} else if rval <= 0 {
			return rval == 0
		}

		l := p.lexer

		p.lexer = NewLexer(l.deref, l.input, l.lines, l.yield)
		p.ohParserImpl = &ohParserImpl{}
	}
}

func (p* parser) ParseCommands(label string) {
	if p.ParseBuffer(label) {
		fmt.Printf("\n")
	}
}

func (p *parser) ParsePipe(label string) (rval int, e interface{}) {
	defer func() {
		e = recover()
	}()

	p.lexer.label = label

	return p.Parse(p.lexer), nil
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
