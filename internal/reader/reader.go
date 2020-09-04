package reader

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/struct/token"
	"github.com/michaelmacinnis/oh/internal/reader/lexer"
	"github.com/michaelmacinnis/oh/internal/reader/parser"
)

type T struct {
	i chan string
	o chan cell.I
	p *parser.T
	s *lexer.T
}

func New(name string) *T {
	r := &T{
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

// TODO: Add Parse(path string) (cell.I, error). Returns parse tree as list.
//       Actually, maybe look at the pipe type first and how to simplify it.

func (r *T) Close() {
	close(r.i)
}

func (r *T) Scan(line string) cell.I {
	r.i <- line
	return <-r.o
}

func (r *T) next() bool {
	line, ok := <-r.i
	if ok {
		r.s.Scan(line)
	}

	return ok
}

func (r *T) start() {
	r.next()

	r.p.Parse()
}
