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
			yield,
		),
	}
}

func (p *parser) ParseBuffer(label string) bool {
	for {
		p.lexer.label = label

		eof, e := p.ParsePipe()
		if e == nil {
			return eof
		}

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

		l := p.lexer

		p.lexer = NewLexer(l.deref, l.input, l.yield)
		p.lexer.lines = l.lines

		p.ohParserImpl = &ohParserImpl{}
	}
}

func (p* parser) ParseCommands(label string) {
	if p.ParseBuffer(label) {
		fmt.Printf("\n")
	}
}

func (p *parser) ParsePipe() (eoi bool, r interface{}) {
	defer func() {
		r = recover()
	}()

	return p.Parse(p.lexer) == 0, nil
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
