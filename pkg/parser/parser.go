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
	input ReadStringer,
	yield func(Cell, string, int) (Cell, bool),
) Parser {
	return &parser{
		&ohParserImpl{},
		NewLexer(
			t.deref,
			input.ReadString,
			yield,
		),
	}
}

func (p *parser) ReadYieldLoop() (normal bool, r interface{}) {
	defer func() {
		r = recover()
	}()

	return p.Parse(p.lexer) == 0, nil
}

func (p *parser) ReadEvalLoop(filename string) {
	for {
		p.lexer.label = filename

		normal, e := p.ReadYieldLoop()
		if e != nil {
        		c := List(
                		NewSymbol("throw"), List(
                        		NewSymbol("_exception"),
                        		NewSymbol("error/syntax"),
                        		NewStatus(NewSymbol("1").Status()),
                        		NewSymbol(fmt.Sprintf("%v", e)),
                        		NewInteger(int64(p.lexer.lines)),
                        		NewSymbol(filename),
                		),
        		)
			p.lexer.yield(c, filename, p.lexer.lines)
		} else if normal {
			return
		}

		l := p.lexer

		p.lexer = NewLexer(l.deref, l.input, l.yield)
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
