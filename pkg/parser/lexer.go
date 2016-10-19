// Released under an MIT license. See LICENSE.

package parser

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/system"
	"strings"
	"unicode/utf8"
)

// Inspired by "Lexical Scanning in Go"; adapted to work with yacc and liner.

type action struct {
	f func(*lexer) *action
}

// The type lexer holds the lexer's state.
type lexer struct {
	after int             // The previous scanned item type.
	items chan *ohSymType // Channel of scanned items.
	saved *action         // The previous action.
	state *action         // The current lexer action.

	bytes string // The buffer being scanned.
	index int    // Position in the 'bytes' buffer.
	start int    // Start position of this item.
	width int    // Width of last rune read.

	deref func(string, uintptr) Cell
	input func(byte) (string, error)
	throw func(string, int, string)
	yield func(Cell, string, int, string) (Cell, bool)

	label string // The name of the thing being parsed.
	lines int    // The number of lines read.
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

var CtrlCPressed = &ohSymType{yys: CTRLC}
var Finished = &ohSymType{yys: 0}

func NewLexer(
	deref func(string, uintptr) Cell,
	input func(byte) (string, error),
	throw func(string, int, string),
	yield func(Cell, string, int, string) (Cell, bool),
	label string,
) *lexer {
	closed := make(chan *ohSymType)
	close(closed)

	return &lexer{
		items: closed,
		state: SkipWhitespace,

		deref: deref,
		input: input,
		throw: throw,
		yield: yield,

		label: label,
	}
}

func (l *lexer) Error(msg string) {
	l.throw(l.label, l.lines, msg)
}

func (l *lexer) Lex() *ohSymType {
	var retries int

	for {
		item := l.item()
		if item != nil {
			return item
		}

		if l.input == nil {
			return Finished
		}

		line, err := l.input('\n')
		if err == nil {
			retries = 0
		} else if err == ErrCtrlCPressed {
			return CtrlCPressed
		} else if system.ResetForegroundGroup(err) {
			retries++
			continue
		}

		l.lines++

		line = strings.Replace(line, "\\\n", "", -1)

		if err != nil {
			line += "\n"
			l.input = nil
		}

		l.scan(line)

		retries = 0
	}
}

func (l *lexer) Restart(lval *ohSymType) bool {
	return lval == CtrlCPressed
}

func (l *lexer) clear() {
	l.after = 0
	l.saved = nil
	l.state = SkipWhitespace

	l.bytes = ""
	l.index = 0
	l.start = 0
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

	s := l.bytes[l.start:l.index]
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

func (l *lexer) item() *ohSymType {
	return <-l.items
}

func (l *lexer) next() rune {
	r, w := l.peek()
	l.skip(w)
	return r
}

func (l *lexer) peek() (r rune, w int) {
	r, w = EOF, 0
	if l.index < len(l.bytes) {
		r, w = utf8.DecodeRuneInString(l.bytes[l.index:])
	}
	return r, w
}

func (l *lexer) scan(bytes string) {
	l.reset()
	if l.bytes != "" {
		l.bytes += bytes
	} else {
		l.bytes = bytes
	}

	l.items = make(chan *ohSymType)
	go l.run()
}

func (l *lexer) reset() {
	if l.start >= len(l.bytes) {
		l.bytes = ""
		l.index = 0
	} else {
		l.bytes = l.bytes[l.start:]
		l.index -= l.start
	}
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

/* Lexer states. */

func afterAmpersand(l *lexer) *action {
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

func afterBackslash(l *lexer) *action {
	r := l.next()

	switch r {
	case EOF:
		return nil
	}

	return l.resume()
}

func afterBang(l *lexer) *action {
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

func afterBangGreater(l *lexer) *action {
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

func afterColon(l *lexer) *action {
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

func afterGreaterThan(l *lexer) *action {
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

func afterLessThan(l *lexer) *action {
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

func afterPipe(l *lexer) *action {
	r, w := l.peek()

	t := PIPE

	switch r {
	case EOF:
		return nil
	case '+':
		l.skip(w)
	case '|':
		if l.bytes[l.start] != '!' {
			t = ORF
			l.skip(w)
		}
	}

	l.emit(t)

	return SkipWhitespace
}

func scanBangString(l *lexer) *action {
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

func scanDoubleQuoted(l *lexer) *action {
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

func scanSingleQuoted(l *lexer) *action {
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

func scanSymbol(l *lexer) *action {
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

func skipComment(l *lexer) *action {
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

func skipWhitespace(l *lexer) *action {
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
	AfterAmpersand = &action{afterAmpersand}
	AfterBang = &action{afterBang}
	AfterBackslash = &action{afterBackslash}
	AfterBangGreater = &action{afterBangGreater}
	AfterColon = &action{afterColon}
	AfterGreaterThan = &action{afterGreaterThan}
	AfterLessThan = &action{afterLessThan}
	AfterPipe = &action{afterPipe}
	ScanBangString = &action{scanBangString}
	ScanDoubleQuoted = &action{scanDoubleQuoted}
	ScanSingleQuoted = &action{scanSingleQuoted}
	ScanSymbol = &action{scanSymbol}
	SkipComment = &action{skipComment}
	SkipWhitespace = &action{skipWhitespace}
}
