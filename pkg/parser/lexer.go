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
	items chan *ohSymType // Channel of scanned items.
	saved *action         // The previous action.
	start int             // Start position of this item.
	state *action         // The action the lexer is currently performing.
	width int             // Width of last rune read.
}

type action struct {
	f func(*lexer) *action
	n string
}

const EOF = -1
const ERROR = 0

// Declared but initialized in init to avoid initialization loop.
var (
	AfterAmpersand   *action
	AfterBackslash   *action
	AfterBang        *action
	AfterBangGreater *action
	AfterColon       *action
	AfterGreaterThan *action
	AfterLessThan    *action
	AfterPipe        *action
	ScanBangString   *action
	ScanDoubleQuoted *action
	ScanSingleQuoted *action
	ScanSymbol       *action
	SkipComment      *action
	SkipWhitespace   *action
)

func NewLexer() *lexer {
	closed := make(chan *ohSymType)
	close(closed)

	return &lexer{
		items: closed,
		state: SkipWhitespace,
	}
}

func (l *lexer) Item() *ohSymType {
	return <-l.items
}

func (l *lexer) Scan(input string) {
	l.reset()
	if l.input != "" {
		l.input += input
	} else {
		l.input = input
	}

	l.items = make(chan *ohSymType)
	go l.run()
}

func (l *lexer) clear() {
	l.after = 0
	l.index = 0
	l.input = ""
	l.saved = nil
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

	l.items <- &ohSymType{yys: yys, s: s}
}

func (l *lexer) error(msg string) *action {
	l.items <- &ohSymType{yys: ERROR, s: msg}
	return nil
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

func (l *lexer) resume() *action {
	saved := l.saved
	l.saved = nil
	return saved
}

func (l *lexer) run() {
	for state := l.state; state != nil; {
		l.state = state
		state = state.f(l)
	}
	close(l.items)
	l.reset()
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

func aAfterBackslash(l *lexer) *action {
	r := l.next()

	switch r {
	case EOF:
		return nil
	}

	return l.resume()
}

func aAfterBang(l *lexer) *action {
	r, w := l.peek()

	switch r {
	case EOF:
		return nil
	case '"':
		l.skip(w)
		return ScanBangString
	case '>':
		l.skip(w)
		return AfterBangGreater
	case '|':
		l.skip(w)
		return AfterPipe
	}

	return ScanSymbol
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
			l.emit(BANG_STRING)
			return SkipWhitespace
		case '\\':
			l.saved = ScanBangString
			return AfterBackslash
		}
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
			l.saved = ScanDoubleQuoted
			return AfterBackslash
		}
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
		case '\n', '\r', '%', '&', '\'', '(', ')', ';', '<',
			'@', '`', '|', '\t', ' ', '"', '#', ':', '>':
			l.emit(SYMBOL)
			return SkipWhitespace
		case '\\':
			l.skip(w)
			l.saved = ScanSymbol
			return AfterBackslash
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
			l.emit(int('\n'))
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
		case '\n':
			switch l.after {
			case ORF, ANDF, PIPE, REDIRECT:
				continue
			}
			fallthrough // {
		case '%', '(', ')', ';', '@', '`', '}':
			l.emit(int(r))
		case '\t', '\r', ' ':
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
		case '\\':
			l.saved = ScanSymbol
			return AfterBackslash
		case '|':
			return AfterPipe
		default:
			return ScanSymbol
		}
	}

	return SkipWhitespace
}

func init() {
	AfterAmpersand = &action{aAfterAmpersand, "&"}
	AfterBang = &action{aAfterBang, "!"}
	AfterBackslash = &action{aAfterBackslash, "\\"}
	AfterBangGreater = &action{aAfterBangGreater, "BG"}
	AfterColon = &action{aAfterColon, ":"}
	AfterGreaterThan = &action{aAfterGreaterThan, ">"}
	AfterLessThan = &action{aAfterLessThan, "<"}
	AfterPipe = &action{aAfterPipe, "|"}
	ScanBangString = &action{aScanBangString, "BDQ"}
	ScanDoubleQuoted = &action{aScanDoubleQuoted, "DQ"}
	ScanSingleQuoted = &action{aScanSingleQuoted, "SQ"}
	ScanSymbol = &action{aScanSymbol, "SYM"}
	SkipComment = &action{aSkipComment, "#"}
	SkipWhitespace = &action{aSkipWhitespace, "WS"}
}
