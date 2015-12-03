// Released under an MIT-style license. See LICENSE.

package parser

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/common"
	"github.com/michaelmacinnis/oh/pkg/task"
	"github.com/michaelmacinnis/oh/pkg/ui"
)

type scanner struct {
	deref   func(string, uintptr) Cell
	input   common.ReadStringer
	line    []rune
	process func(Cell)
	task    *task.Task

	cursor   int
	lineno   int
	filename string
	previous rune
	start    int
	state    int
	token    rune

	finished bool
}

const (
	ssStart = iota
	ssAmpersand
	ssBang
	ssBangGreater
	ssColon
	ssComment
	ssDoubleQuoted
	ssDoubleQuotedEscape
	ssGreater
	ssLess
	ssPipe
	ssSingleQuoted
	ssSymbol
)

func (s *scanner) Lex(lval *yySymType) (token int) {
	var operator = map[string]string{
		"!>":  "redirect-stderr",
		"!>>": "append-stderr",
		"!|":  "pipe-stderr",
		"!|+": "channel-stderr",
		"&":   "spawn",
		"&&":  "and",
		"<":   "redirect-stdin",
		"<(":  "substitute-stdout",
		">":   "redirect-stdout",
		">(":  "substitute-stdin",
		">>":  "append-stdout",
		"|":   "pipe-stdout",
		"|+":  "channel-stdout",
		"||":  "or",
	}

	defer func() {
		exists := false

		switch s.token {
		case BACKGROUND, ORF, ANDF, PIPE, REDIRECT, SUBSTITUTE:
			lval.s, exists = operator[string(s.line[s.start:s.cursor])]
			if exists {
				break
			}
			fallthrough
		default:
			lval.s = string(s.line[s.start:s.cursor])
		}

		s.state = ssStart
		s.previous = s.token
		s.token = 0
	}()

main:
	for s.token == 0 {
		if s.cursor >= len(s.line) {
			if s.finished {
				return 0
			}

			line, error := s.input.ReadString('\n')
			if error == ui.CtrlCPressed {
				s.start = 0
				s.token = CTRLC
				break
			}

			s.lineno++
			s.task.File = s.filename
			s.task.Line = s.lineno

			runes := []rune(line)
			last := len(runes) - 2
			if last >= 0 && runes[last] == '\r' {
				runes = append(runes[0:last], '\n')
				last--
			}

			if last >= 0 && runes[last] == '\\' {
				runes = runes[0:last]
			}

			if error != nil {
				runes = append(runes, '\n')
				s.finished = true
			}

			if s.start < s.cursor-1 {
				s.line = append(s.line[s.start:s.cursor], runes...)
				s.cursor -= s.start
			} else {
				s.cursor = 0
				s.line = runes
			}
			s.start = 0
			s.token = 0
		}

		switch s.state {
		case ssStart:
			s.start = s.cursor

			switch s.line[s.cursor] {
			default:
				s.state = ssSymbol
				continue main
			case '\n', '%', '(', ')', ';', '@', '^', '`', '{', '}':
				s.token = s.line[s.cursor]
			case '&':
				s.state = ssAmpersand
			case '<':
				s.state = ssLess
			case '|':
				s.state = ssPipe
			case '\t', ' ':
				s.state = ssStart
			case '!':
				s.state = ssBang
			case '"':
				s.state = ssDoubleQuoted
			case '#':
				s.state = ssComment
			case '\'':
				s.state = ssSingleQuoted
			case ':':
				s.state = ssColon
			case '>':
				s.state = ssGreater
			}

		case ssAmpersand:
			switch s.line[s.cursor] {
			case '&':
				s.token = ANDF
			default:
				s.token = BACKGROUND
				continue main
			}

		case ssBang:
			switch s.line[s.cursor] {
			case '>':
				s.state = ssBangGreater
			case '|':
				s.state = ssPipe
			default:
				s.state = ssSymbol
				continue main
			}

		case ssBangGreater:
			s.token = REDIRECT
			if s.line[s.cursor] != '>' {
				continue main
			}

		case ssColon:
			switch s.line[s.cursor] {
			case ':':
				s.token = CONS
			default:
				s.token = ':'
				continue main
			}

		case ssComment:
			for s.line[s.cursor] != '\n' ||
				s.line[s.cursor-1] == '\\' {
				s.cursor++

				if s.cursor >= len(s.line) {
					continue main
				}
			}
			s.cursor--
			s.state = ssStart

		case ssDoubleQuoted, ssDoubleQuotedEscape:
			for s.cursor < len(s.line) {
				if s.state == ssDoubleQuotedEscape {
					s.state = ssDoubleQuoted
				} else if s.line[s.cursor] == '"' {
					break
				} else if s.line[s.cursor] == '\\' {
					s.state = ssDoubleQuotedEscape
				}
				s.cursor++
			}
			if s.cursor >= len(s.line) {
				if s.line[s.cursor-1] == '\n' {
					s.line = append(s.line[0:s.cursor-1], []rune("\\n")...)
				}
				continue main
			}
			s.token = DOUBLE_QUOTED

		case ssGreater:
			s.token = REDIRECT
			if s.line[s.cursor] == '(' {
				s.token = SUBSTITUTE
			} else if s.line[s.cursor] != '>' {
				continue main
			}

		case ssLess:
			s.token = REDIRECT
			if s.line[s.cursor] == '(' {
				s.token = SUBSTITUTE
			} else {
				continue main
			}

		case ssPipe:
			switch s.line[s.cursor] {
			case '+':
				s.token = PIPE
			case '|':
				s.token = ORF
			default:
				s.token = PIPE
				continue main
			}

		case ssSingleQuoted:
			for s.cursor < len(s.line) && s.line[s.cursor] != '\'' {
				s.cursor++
			}
			if s.cursor >= len(s.line) {
				if s.line[s.cursor-1] == '\n' {
					s.line = append(s.line[0:s.cursor-1], []rune("\\n")...)
				}
				continue main
			}
			s.token = SINGLE_QUOTED

		case ssSymbol:
			switch s.line[s.cursor] {
			case '\n', '%', '&', '\'', '(', ')', ';',
				'<', '@', '^', '`', '{', '|', '}',
				'\t', ' ', '"', '#', ':', '>':
				s.token = SYMBOL
				continue main
			}

		}
		s.cursor++

		if s.token == '\n' {
			switch s.previous {
			case ORF, ANDF, PIPE, REDIRECT:
				s.token = 0
			}
		}
	}

	return int(s.token)
}

func (s *scanner) Error(msg string) {
	task.PrintError(s.filename, s.lineno, msg)
}

func Parse(t *task.Task,
	r common.ReadStringer,
	f string,
	d func(string, uintptr) Cell,
	c func(Cell)) bool {

	s := new(scanner)

	s.deref = d
	s.filename = f
	s.input = r
	s.process = c
	s.task = t

restart:
	s.line = []rune("")
	s.cursor = 0
	s.previous = 0
	s.start = 0
	s.token = 0

	s.finished = false

	s.state = ssStart

	rval := yyParse(s)
	if rval > 0 {
		goto restart
	}

	return rval == 0
}

//go:generate go tool yacc -o grammar.go grammar.y
//go:generate sed -i.save -f grammar.sed grammar.go
//go:generate rm -f y.output grammar.go.save
