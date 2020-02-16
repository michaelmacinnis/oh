package main

import (
	"github.com/michaelmacinnis/oh/internal/engine"
        "github.com/michaelmacinnis/oh/internal/reader/lexer"
        "github.com/michaelmacinnis/oh/internal/reader/parser"
	"testing"
)

func TestInput(t *testing.T) {
	e := engine.New()

	l := lexer.New("oh")
	p := parser.New(e.Evaluate, l.Token)

	l.Scan(`
if $success {
	debug 1
	debug 2
} else {
	debug 3
	debug 4
}

define f (method (a) = {
	debug $a
})
f (define x 7)
`)

	p.Parse()
}
