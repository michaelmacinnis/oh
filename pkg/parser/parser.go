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
	*lexer
	*parser
	input   ReadStringer
	process func(Cell, string, int, string) (Cell, bool)
	thrower Thrower

	filename string
	lineno   int

	finished bool
}

func (s *scanner) Error(msg string) {
	s.thrower.Throw(s.filename, s.lineno, msg)
}

func (s *scanner) Lex(lval *ohSymType) (token int) {
	var item *ohSymType
	var retries int

	for {
		item = s.Item()
		if item != nil {
			lval.s = item.s
			return item.yys
		}

		if s.finished {
			return 0
		}

		line, err := s.input.ReadString('\n')
		if err == nil {
			retries = 0
		} else if err == ErrCtrlCPressed {
			return CTRLC
		} else if system.ResetForegroundGroup(err) {
			retries++
			continue
		}

		s.lineno++

		line = strings.Replace(line, "\\\n", "", -1)

		if err != nil {
			line += "\n"
			s.finished = true
		}

		s.Scan(line)

		retries = 0
	}
}

func (s *scanner) Restart(r int) bool {
	return r == CTRLC
}

func New(deref func(string, uintptr) Cell) *parser {
	return &parser{deref}
}

func (p *parser) Parse(
	input ReadStringer, t Thrower, filename string,
	process func(Cell, string, int, string) (Cell, bool),
) bool {

	s := new(scanner)

	s.lexer = NewLexer()
	s.parser = p

	s.input = input
	s.process = process
	s.thrower = t

	s.filename = filename

	rval := 1
	for rval > 0 {
		s.finished = false
		s.clear()

		rval = ohParse(s)
	}

	return rval == 0
}

//go:generate ohyacc -o grammar.go -p oh -v /dev/null grammar.y
//go:generate go fmt grammar.go
