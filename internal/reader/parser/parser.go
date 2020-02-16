// Released under an MIT license. See LICENSE.

// Package parser provides a recursive descent parser for the oh language.
package parser

import (
	"errors"
	"strconv"
	"strings"

	"github.com/michaelmacinnis/oh/internal/adapted"
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/reader/token"
	"github.com/michaelmacinnis/oh/internal/type/errsys"
	"github.com/michaelmacinnis/oh/internal/type/list"
	"github.com/michaelmacinnis/oh/internal/type/loc"
	"github.com/michaelmacinnis/oh/internal/type/pair"
	"github.com/michaelmacinnis/oh/internal/type/str"
	"github.com/michaelmacinnis/oh/internal/type/sym"
)

// T holds the state of the parser.
type T struct {
	ahead  int             // Lookahead count.
	emit   func(cell.T)    // Function to call to emit a parsed command.
	item   func() *token.T // Function to call to get another token.
	source loc.T           // The parser's current position.
	token  *token.T        // Token lookahead.

	// Completion state.
	part  cell.T // The command being parsed, so far.
	saved cell.T // The command being parsed.
}

// Stringer is duplicated here so we don't have to import fmt.
type Stringer interface {
	String() string
}

// New creates a new parser.
// It connects a producer of tokens with a consumer of cells.
func New(emit func(cell.T), item func() *token.T) *T {
	return &T{emit: emit, item: item, part: pair.Null}
}

// Copy copies the current parser but replaces its emit and item functions.
func (p *T) Copy(emit func(cell.T), item func() *token.T) *T {
	c := *p

	c.emit = emit
	c.item = item

	return &c
}

// Current returns the command currently being parsed.
func (p *T) Current() cell.T {
	return p.saved
}

// Parse consumes tokens and emits cells until there are no more tokens.
func (p *T) Parse() {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		switch r := r.(type) {
		case *errsys.T:
			p.emit(r)
		case error:
			p.emit(errsys.New(r))
		case string:
			p.emit(errsys.New(errors.New(r)))
		case Stringer:
			p.emit(errsys.New(errors.New(r.String())))
		default:
			p.emit(errsys.New(errors.New("unexpected error")))
		}
	}()

	for t := p.peek(); t != nil; t = p.peek() {
		if t.Is('\n') {
			p.consume()
			continue
		}
		p.emit(p.possibleBackground())
	}
}

func (p *T) consume() *token.T {
	if p.ahead == 0 {
		panic("nothing to consume.")
	}

	t := p.token

	p.ahead = 0
	p.token = nil

	return t
}

func (p *T) check(c cell.T) cell.T {
	if c == nil {
		t := p.peek()

		loc := t.Source()
		l := loc.Name
		x := strconv.Itoa(loc.Char)
		y := strconv.Itoa(loc.Line)

		panic(l + ":" + y + ":" + x + ": unexpected '" + t.Value() + "'")
	}

	return c
}

func (p *T) expect(cs ...token.Class) {
	if p.peek().Is(cs...) {
		p.consume()
		return
	}

	// Make a nice error message.
	n := len(cs)
	e := make([]string, n)
	for i, c := range cs[:n-1] {
		e[i] = c.String()
	}

	l := cs[n-1].String()
	if n > 2 {
		l = ", or " + l
	} else if n > 1 {
		l = " or " + l
	}

	l = strings.Join(e, ", ") + l
	s := p.peek().Value()

	panic("expected " + l + ` got "` + s + `"`)
}

func (p *T) peek() *token.T {
	if p.ahead > 0 {
		return p.token
	}

	t := p.item()
	if t == nil {
		p.saved = p.part
	} else if p.source.Line == 0 {
		p.source = t.Source()
	}

	p.token = t
	p.ahead = 1

	return t
}

// T state functions.

// <possibleBackground> ::= <command> '&'?
func (p *T) possibleBackground() cell.T {
	c := p.command()

	t := p.peek()
	if t.Is(token.Background) {
		p.consume()

		c = list.New(sym.New(t.Value()), c)
	}

	// Reset line so that it will be set when the next token is comsumed.
	p.source.Line = 0

	return c
}

// <command> ::= <possibleAndf> (Orf <possibleAndf>)*
func (p *T) command() cell.T {
	c := p.possibleAndf()

	t := p.peek()
	if t.Is(token.Orf) {
		c = list.New(sym.New(t.Value()), c)

		for p.peek().Is(token.Orf) {
			p.consume()
			c = list.Append(c, p.possibleAndf())
		}
	}

	return c
}

// <possibleAndf> ::= <possiblePipeline> (Andf <possiblePipeline>)*
func (p *T) possibleAndf() cell.T {
	c := p.possiblePipeline()

	t := p.peek()
	if t.Is(token.Andf) {
		c = list.New(sym.New(t.Value()), c)

		for p.peek().Is(token.Andf) {
			p.consume()
			c = list.Append(c, p.possiblePipeline())
		}
	}

	return c
}

// <possiblePipeline> ::= <possibleSequence> (Pipe <possiblePipeline>)?
func (p *T) possiblePipeline() cell.T {
	c := p.possibleSequence()

	if p.peek().Is(token.Pipe) {
		s := sym.New(p.consume().Value())

		c = pair.Cons(p.possiblePipeline(), c)
		c = pair.Cons(s, c)
	}

	return c
}

// <possibleSequence> ::= <possibleRedirection> (';' <possibleRedirection>)*
func (p *T) possibleSequence() cell.T {
	c := p.possibleRedirection()

	if p.peek().Is(';') {
		c = list.New(sym.New("block"), c)

		for p.peek().Is(';') {
			p.consume()

			c = list.Append(c, p.possibleRedirection())
		}
	}

	return c
}

// <possibleRedirection> ::= <possibleSustitution> (Redirect <expression>)*
func (p *T) possibleRedirection() cell.T {
	c := p.possibleSubstitution()

	for p.peek().Is(token.Redirect) {
		s := sym.New(p.consume().Value())
		c = list.New(s, p.check(p.possibleImplicitJoin()), c)

		for p.peek().Is(token.Space) {
			p.consume()
		}
	}

	return c
}

// <possibleSubstitution> ::= <statement> (Substitute <command> ')' <statement>?)*
func (p *T) possibleSubstitution() cell.T {
	c := p.statement()
	if c == nil {
		return c
	}

	if p.peek().Is(token.Substitute) {
		c = pair.Cons(sym.New("_process_substitution_"), c)

		for p.peek().Is(token.Substitute) {
			s := sym.New(p.consume().Value())
			l := list.New(s, p.element())
			c = list.Append(c, l)

			if !p.peek().Is(token.Substitute) {
				s := p.statement()
				if s != nil {
					c = list.Join(c, s)
				}
			}
		}
	}

	return c
}

func (p *T) statement() (c cell.T) {
	for p.peek().Is(token.Space) {
		p.consume()
	}

	// TODO: Pull this if-else block into its own function.
	//       It is repeated below.
	if p.peek().Is('{') {
		p.consume()

		if p.peek().Is('\n') {
			p.consume()

			c = p.subStatement()
		} else {
			c = p.possibleImplicitJoin()
			c = pair.Cons(c, pair.Null, p.source)
			p.expect('}')
		}
	} else {
		c = p.possibleImplicitJoin()
		if c == nil {
			return nil
		}

		c = pair.Cons(c, pair.Null, p.source)
	}

	// Push new part onto current stack.
	p.part = pair.Cons(c, p.part)

	for {
		var t cell.T

		if p.peek().Is(token.Space) {
			p.consume()
			continue
		}

		if p.peek().Is('{') {
			p.consume()

			if p.peek().Is('\n') {
				p.consume()

				t = p.subStatement()
			} else {
				t = p.possibleImplicitJoin()
				t = pair.Cons(t, pair.Null)
				p.expect('}')
			}
		} else {
			t = p.possibleImplicitJoin()
			if t == nil {
				break
			}

			t = pair.Cons(t, pair.Null)
		}

		c = list.Join(c, t)

		// Update current part.
		pair.SetCar(p.part, c)
	}

	// Pop previous part off stack.
	p.part = pair.Cdr(p.part)

	return c
}

func (p *T) subStatement() cell.T {
	c := p.block()

	p.expect('}')

	for p.peek().Is(token.Space) {
		p.consume()
	}

	s := p.statement()
	if s != nil {
		c = list.Join(c, s)
	}

	return c
}

func (p *T) block() cell.T {
	c := pair.Null

	for !p.peek().Is('}') {
		if p.peek().Is('\n') {
			p.consume()
			continue
		}

		c = list.Append(c, p.check(p.possibleBackground()))
	}

	return c
}

func (p *T) possibleImplicitJoin() cell.T {
	c := p.element()
	if c == nil {
		return nil
	}

	c = list.New(c)

	for t := p.element(); t != nil; t = p.element() {
		c = list.Append(c, t)
	}

	if list.Length(c) == 1 {
		return pair.Car(c)
	}

	return pair.Cons(sym.New("_join_"), c)
}

func (p *T) element() cell.T {
	if p.peek().Is('`') {
		p.consume()

		c := p.check(p.value())

		c = list.New(sym.New("_capture_"), c)
		c = list.New(sym.New("_splice_"), c)

		return c
	}

	return p.expression()
}

func (p *T) expression() cell.T {
	if p.peek().Is('$') {
		p.consume()

		return list.New(sym.New("_lookup_"), p.check(p.expression()))
	}

	return p.value()
}

func (p *T) value() cell.T {
	if !p.peek().Is('(') {
		return p.word()
	}

	p.consume()

	c := p.command()
	if c == nil {
		t := p.peek()
		if t.Is(')') {
			p.consume()

			return pair.Null
		}
		panic("unexpected '" + t.Value() + "'")
	}

	p.expect(')')

	return c
}

func (p *T) word() cell.T {
	t := p.peek()
	if t.Is(token.DollarSingleQuoted) {
		p.consume()

		text := t.Value()
		s, err := adapted.ActualBytes(text[2 : len(text)-1])
		if err != nil {
			panic(err)
		}
		return str.New(s)
	}

	if t.Is(token.DoubleQuoted) {
		p.consume()

		text := t.Value()
		s, err := adapted.ActualBytes(text[1 : len(text)-1])
		if err != nil {
			panic(err)
		}
		return list.New(sym.New("interpolate"), str.New(s))
	}

	if t.Is(token.SingleQuoted) {
		p.consume()

		s := t.Value()
		return str.New(s[1 : len(s)-1])
	}

	if t.Is(token.Symbol) {
		p.consume()

		return sym.New(t.Value())
	}

	return nil
}
