// Released under an MIT license. See LICENSE.

package parser

import (
	"fmt"
	. "github.com/michaelmacinnis/oh/pkg/cell"
)

type parser struct {
	deref DerefFunc
	input InputFunc
	*ohParserImpl
	*lexer
}

func New(
	deref DerefFunc,
	input InputFunc,
) Parser {
	return &parser{
		deref: deref,
		input: input,
	}
}

func (p *parser) ParseBuffer(label string, yield YieldFunc) bool {
	lines := 0
	for {
		rval, lines, e := p.parsePipe(label, lines, yield)
		if e != nil {
			c := List(
				NewSymbol("throw"), List(
					NewSymbol("_exception"),
					List(NewSymbol("quote"), NewSymbol("error/syntax")),
					NewStatus(NewSymbol("1").Status()),
					List(NewSymbol("quote"), NewSymbol(fmt.Sprintf("%v", e))),
					NewInteger(int64(lines)),
					List(NewSymbol("quote"), NewSymbol(label)),
				),
			)
			yield(c)
		} else if rval <= 0 {
			return rval == 0
		}
	}
}

func (p *parser) ParseCommands(label string, yield YieldFunc) {
	if p.ParseBuffer(label, yield) {
		fmt.Printf("\n")
	}
}

func (p *parser) ParsePipe(label string, yield YieldFunc) (int, interface{}) {
	_, l, e := p.parsePipe(label, 0, yield)

	return l, e
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

func (p *parser) parsePipe(label string, start int, yield YieldFunc) (rval int, lines int, e interface{}) {
	p.lexer = NewLexer(p.deref, p.input, label, start, yield)
	defer func() {
		e = recover()
		lines = p.lexer.lines
	}()

	p.ohParserImpl = &ohParserImpl{}

	return p.Parse(p.lexer), p.lexer.lines, nil
}

//go:generate ohyacc -o grammar.go -p oh -v /dev/null grammar.y
//go:generate go fmt grammar.go
