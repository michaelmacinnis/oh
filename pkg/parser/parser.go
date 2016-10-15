// Released under an MIT license. See LICENSE.

package parser

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/system"
	"strings"
)

type parser struct {
	deref func(string, uintptr) Cell
}

type scanner struct {
	*parser
	t        Thrower
	filename string
	input    ReadStringer
	l        *lexer
	process  func(Cell, string, int, string) (Cell, bool)

	lineno   int
	finished bool
}

func (s *scanner) Lex(lval *yySymType) (token int) {
	var item *yySymType
	var retries int

	if s.l == nil {
		s.l = NewLexer()
		goto read
	}

scan:
	item = s.l.Item()
	if item != nil {
		lval.s = item.s
		return item.yys
	}

	if s.finished {
		return 0
	}

read:
	line, err := s.input.ReadString('\n')
	if err == nil {
		retries = 0
	} else if err == ErrCtrlCPressed {
		return CTRLC
	} else if system.ResetForegroundGroup(err) {
		retries++
		goto scan
	}

	s.lineno++

	line = strings.Replace(line, "\\\n", "", -1)

	if err != nil {
		line += "\n"
		s.finished = true
	}

	s.l.Scan(line)

	retries = 0
	goto scan
}

func (s *scanner) Error(msg string) {
	s.t.Throw(s.filename, s.lineno, msg)
}

func New(deref func(string, uintptr) Cell) *parser {
	return &parser{deref}
}

func (p *parser) Parse(
	input ReadStringer, t Thrower, filename string,
	process func(Cell, string, int, string) (Cell, bool),
) bool {

	s := new(scanner)

	s.filename = filename
	s.input = input
	s.parser = p
	s.process = process
	s.t = t

	rval := 1
	for rval > 0 {
		s.finished = false
		if s.l != nil {
			s.l.clear()
		}

		rval = yyParse(s)
	}

	return rval == 0
}

//go:generate goyacc -o grammar.go grammar.y
//go:generate sed -i.save -f grammar.sed grammar.go
//go:generate go fmt grammar.go
//go:generate rm -f y.output grammar.go.save
