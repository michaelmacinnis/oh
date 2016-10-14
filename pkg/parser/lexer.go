// Released under an MIT license. See LICENSE.

package parser

import (
	"unicode/utf8"
)

// Inspired by "Lexical Scanning in Go"; adapted to work with goyacc.

// The type lexer holds the state of the scanner.
type lexer struct {
	after int             // The previous scanned item type.
	index int             // Current position in the input.
	input string          // The string being scanned.
	items chan *yySymType // Channel of scanned items.
	start int             // Start position of this item.
	state *action         // The action the lexer is currently performing.
	width int             // Width of last rune read.

}

type action struct {
	f func(*lexer) *action
	n string
}

const EOF = -1

// Declared but initialized in init to avoid initialization loop.
var (
	AfterAmpersand         *action
	AfterBang              *action
	AfterBangGreater       *action
	AfterColon             *action
	AfterGreaterThan       *action
	AfterLessThan          *action
	AfterPipe              *action
	ScanBangString         *action
	ScanBangStringEscape   *action
	ScanDoubleQuoted       *action
	ScanDoubleQuotedEscape *action
	ScanSingleQuoted       *action
	ScanSymbol             *action
	SkipComment            *action
	SkipWhitespace         *action
)

func NewLexer() *lexer {
	return &lexer{
		items: make(chan *yySymType),
		state: SkipWhitespace,
	}
}

func (l *lexer) Item() *yySymType {
	return <-l.items
}

func (l *lexer) Scan(input string) {
	l.reset()
	if l.input != "" {
		l.input += input
	} else {
		l.input = input
	}

	go l.run()
}

func (l *lexer) clear() {
	l.after = 0
	l.index = 0
	l.input = ""
	l.start = 0
	l.state = SkipWhitespace
	l.width = 0
}

func (l *lexer) emit(yys int) {
	operator := map[string]string{
		"!>":  "_redirect_stderr_",
		"!>>": "_append_stderr_",
		"!|":  "_pipe_stderr_",
		"!|+": "_channel_stderr_",
		"&":   "spawn",
		"&&":  "and",
		"<":   "_redirect_stdin_",
		"<(":  "_substitute_stdout_",
		">":   "_redirect_stdout_",
		">(":  "_substitute_stdin_",
		">>":  "_append_stdout_",
		"|":   "_pipe_stdout_",
		"|+":  "_channel_stdout_",
		"||":  "or",
	}

	s := l.input[l.start:l.index]
	l.start = l.index

	switch yys {
	case SYMBOL:
		left, right, token := -1, -1, SYMBOL
		for i, r := range s {
			switch r {
			case '{':
				left = i
				token = int(r)
			case '}':
				right = i
				token = int(r)
			}
		}
		yys = BRACE_EXPANSION
		if left == right || len(s) == 1 {
			yys = token
		}
	case BACKGROUND, ORF, ANDF, PIPE, REDIRECT, SUBSTITUTE:
		op, exists := operator[s]
		if exists {
			s = op
		}
	}

	l.after = yys

	l.items <- &yySymType{yys: yys, s: s}
}

func (l *lexer) next() rune {
	r, w := l.peek()
	l.skip(w)
	return r
}

func (l *lexer) peek() (r rune, w int) {
	r, w = EOF, 0
	if l.index < len(l.input) {
		r, w = utf8.DecodeRuneInString(l.input[l.index:])
	}
	return r, w
}

func (l *lexer) reset() {
	l.input = l.input[l.start:]
	l.index -= l.start
	l.start = 0
}

func (l *lexer) run() {
	for state := l.state; state != nil; {
		l.state = state
		state = state.f(l)
	}
	l.reset()
	l.items <- nil
}

func (l *lexer) skip(w int) {
	l.width = w
	l.index += l.width
}

func aAfterAmpersand(l *lexer) *action {
	r, w := l.peek()

	switch r {
	case EOF:
		return nil
	case '&':
		l.skip(w)
		l.emit(ANDF)
	default:
		l.emit(BACKGROUND)
	}

	return SkipWhitespace
}

func aAfterBang(l *lexer) *action {
	r := l.next()

	switch r {
	case EOF:
		return nil
	case '"':
		return ScanBangString
	case '>':
		return AfterBangGreater
	case '|':
		return AfterPipe
	default:
		return ScanSymbol
	}

	return SkipWhitespace
}

func aAfterBangGreater(l *lexer) *action {
	r, w := l.peek()

	switch r {
	case EOF:
		return nil
	case '>':
		l.skip(w)
	}

	l.emit(REDIRECT)

	return SkipWhitespace
}

func aAfterColon(l *lexer) *action {
	r, w := l.peek()

	switch r {
	case EOF:
		return nil
	case ':':
		l.skip(w)
		l.emit(CONS)
	default:
		l.emit(int(':'))
	}

	return SkipWhitespace
}

func aAfterGreaterThan(l *lexer) *action {
	r, w := l.peek()

	t := REDIRECT

	switch r {
	case EOF:
		return nil
	case '(':
		l.skip(w)
		t = SUBSTITUTE
	case '>':
		l.skip(w)
	}

	l.emit(t)

	return SkipWhitespace
}

func aAfterLessThan(l *lexer) *action {
	r, w := l.peek()

	t := REDIRECT

	switch r {
	case EOF:
		return nil
	case '(':
		l.skip(w)
		t = SUBSTITUTE
	}

	l.emit(t)

	return SkipWhitespace
}

func aAfterPipe(l *lexer) *action {
	r, w := l.peek()

	t := PIPE

	switch r {
	case EOF:
		return nil
	case '+':
		l.skip(w)
	case '|':
		if l.input[l.start] != '!' {
			t = ORF
			l.skip(w)
		}
	}

	l.emit(t)

	return SkipWhitespace
}

func aScanBangString(l *lexer) *action {
	for {
		c := l.next()

		switch c {
		case EOF:
			return nil
		case '"':
			l.emit(BANG_DOUBLE)
			return SkipWhitespace
		case '\\':
			return ScanBangStringEscape
		}
	}
}

func aScanBangStringEscape(l *lexer) *action {
	for {
		c := l.next()
		switch c {
		case EOF:
			return nil
		}
		return ScanBangString
	}
}

func aScanDoubleQuoted(l *lexer) *action {
	for {
		c := l.next()

		switch c {
		case EOF:
			return nil
		case '"':
			l.emit(DOUBLE_QUOTED)
			return SkipWhitespace
		case '\\':
			return ScanDoubleQuotedEscape
		}
	}
}

func aScanDoubleQuotedEscape(l *lexer) *action {
	for {
		c := l.next()
		switch c {
		case EOF:
			return nil
		}
		return ScanDoubleQuoted
	}
}

func aScanSingleQuoted(l *lexer) *action {
	for {
		r := l.next()

		switch r {
		case EOF:
			return nil
		case '\'':
			l.emit(SINGLE_QUOTED)
			return SkipWhitespace
		}
	}
}

func aScanSymbol(l *lexer) *action {
	for {
		r, w := l.peek()

		switch r {
		case EOF:
			return nil
		case '\n', '%', '&', '\'', '(', ')', ';', '<', '@',
			'`', '|', '\t', ' ', '"', '#', ':', '>':
			l.emit(SYMBOL)
			return SkipWhitespace
		default:
			l.skip(w)
		}
	}
}

func aSkipComment(l *lexer) *action {
	for {
		r := l.next()

		switch r {
		case EOF:
			return nil
		case '\n':
			return SkipWhitespace
		}
	}
}

func aSkipWhitespace(l *lexer) *action {
	for {
		l.start = l.index
		r := l.next()

		switch r {
		case EOF:
			return nil
		default:
			return ScanSymbol // {
		case '%', '(', ')', ';', '@', '`', '}':
			l.emit(int(r))
		case '\n':
			switch l.after {
			case ORF, ANDF, PIPE, REDIRECT:
				continue
			}
			l.emit(int(r))
		case '\t', ' ':
			continue
		case '!':
			return AfterBang
		case '"':
			return ScanDoubleQuoted
		case '#':
			return SkipComment
		case '&':
			return AfterAmpersand
		case '\'':
			return ScanSingleQuoted
		case ':':
			return AfterColon
		case '<':
			return AfterLessThan
		case '>':
			return AfterGreaterThan
		case '|':
			return AfterPipe
		}
	}

	return SkipWhitespace
}

func init() {
	AfterAmpersand = &action{aAfterAmpersand, "&"}
	AfterBang = &action{aAfterBang, "!"}
	AfterBangGreater = &action{aAfterBangGreater, "BG"}
	AfterColon = &action{aAfterColon, ":"}
	AfterGreaterThan = &action{aAfterGreaterThan, ">"}
	AfterLessThan = &action{aAfterLessThan, "<"}
	AfterPipe = &action{aAfterPipe, "|"}
	ScanBangString = &action{aScanBangString, "BDQ"}
	ScanBangStringEscape = &action{aScanBangStringEscape, "BDQE"}
	ScanDoubleQuoted = &action{aScanDoubleQuoted, "DQ"}
	ScanDoubleQuotedEscape = &action{aScanDoubleQuotedEscape, "DQE"}
	ScanSingleQuoted = &action{aScanSingleQuoted, "SQ"}
	ScanSymbol = &action{aScanSymbol, "SYM"}
	SkipComment = &action{aSkipComment, "#"}
	SkipWhitespace = &action{aSkipWhitespace, "WS"}
}
