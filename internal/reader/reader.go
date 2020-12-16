package reader

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/struct/token"
	"github.com/michaelmacinnis/oh/internal/reader/lexer"
	"github.com/michaelmacinnis/oh/internal/reader/parser"
)

// T (reader) encapsulates the oh lexer and parser.
type T struct {
	e chan error
	i chan string
	o chan cell.I
	p *parser.T
	s *lexer.T
}

type reader = T

// New creates a new reader for name.
func New(name string) *T {
	r := &T{
		e: make(chan error),
		i: make(chan string),
		o: make(chan cell.I),
		s: lexer.New(name),
	}

	var v cell.I

	r.p = parser.New(func(c cell.I) {
		v = c
	}, func() *token.T {
		t := r.s.Token()

		for t == nil {
			r.o <- v

			v = nil

			if !r.next() {
				close(r.o)
			}

			t = r.s.Token()
		}

		return t
	})

	go r.start()

	return r
}

// Close terminates the reader.
func (r *reader) Close() {
	close(r.i)
}

// Lexer returns the reader's internal lexer.T.
func (r *reader) Lexer() *lexer.T {
	return r.s
}

// Parser returns the readers's internal parser.T.
func (r *reader) Parser() *parser.T {
	return r.p
}

// Scan reads the line and returns a cell.I on a complete parse or nil otherwise.
// If scan encounters any error it returns the error.
func (r *reader) Scan(line string) (c cell.I, err error) {
	r.i <- line

	select {
	case c = <-r.o:
	case err = <-r.e:
	}

	return c, err
}

func (r *reader) next() bool {
	line, ok := <-r.i
	if ok {
		r.s.Scan(line)
	}

	return ok
}

func (r *reader) start() {
	r.next()

	r.e <- r.p.Parse()
	close(r.e)
}
