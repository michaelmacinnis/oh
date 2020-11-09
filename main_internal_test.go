package main

import (
	"testing"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/engine"
	"github.com/michaelmacinnis/oh/internal/reader"
	"github.com/michaelmacinnis/oh/internal/system/job"
	"github.com/michaelmacinnis/oh/internal/system/process"
)

func TestCompletion(*testing.T) {
	r := reader.New("test")
	s := "ls "
	n := 3

	h := s[:n]
	//t := s[n:]

	lc := r.Lexer().Copy()

	lc.Scan(h)

	lp := r.Parser().Copy(func(_ cell.I) {}, lc.Token)

	lp.Parse()

	cs := lc.Expected()
	if len(cs) != 0 {
		return
	}
}

func TestPrintingSymbol(*testing.T) {
	engine.Boot(nil)

	j := job.New(process.Group())
	r := reader.New("oh")

	engine.Evaluate(j, r.Scan(`echo (symbol '')
	`))
}

func TestFunctionScope(*testing.T) {
	engine.Boot(nil)

	j := job.New(process.Group())
	r := reader.New("oh")

	engine.Evaluate(j, r.Scan(`define f: method (x) = {
	debug $x
}
	`))
	engine.Evaluate(j, r.Scan("f 3\n"))
	engine.Evaluate(j, r.Scan("debug $x\n"))
	engine.Evaluate(j, r.Scan("f 4\n"))
	engine.Evaluate(j, r.Scan("debug $x\n"))
}

func TestContinuations(*testing.T) {
	engine.Boot(nil)

	r := reader.New("oh")

	engine.Evaluate(job.New(process.Group()), r.Scan(`
source convoluted.oh
`))
}

func TestDollarQuestionMark(*testing.T) {
	engine.Boot(nil)

	j := job.New(process.Group())
	r := reader.New("oh")

	// The output of this should be 3.
	engine.Evaluate(j, r.Scan(`define f: method () = {
	fatal 3
}
	`))
	engine.Evaluate(j, r.Scan("f\n"))
	engine.Evaluate(j, r.Scan("debug $?\n"))
}

func TestExported(*testing.T) {
	engine.Boot(nil)

	j := job.New(process.Group())
	r := reader.New("oh")

	engine.Evaluate(j, r.Scan(`
define f: method () = {
    debug $x
}
`))
	engine.Evaluate(j, r.Scan(`
define g: method () = {
    export x 6
    f
}
`))
	engine.Evaluate(j, r.Scan("g\n"))
}

func TestInnerScope(*testing.T) {
	engine.Boot(nil)

	j := job.New(process.Group())
	r := reader.New("oh")

	// The output of this should be 3.
	engine.Evaluate(j, r.Scan(`define f: method () = {
	define x 3
	fatal $x
}
	`))
	engine.Evaluate(j, r.Scan("f\n"))
	engine.Evaluate(j, r.Scan("debug $?\n"))
}

func TestEvaluate(*testing.T) {
	engine.Boot(nil)

	c := list.New(sym.New("ls"))

	engine.Evaluate(job.New(process.Group()), c)
}

func TestSource(*testing.T) {
	engine.Boot(nil)

	r := reader.New("oh")

	engine.Evaluate(job.New(process.Group()), r.Scan("source blah.oh\n"))
}

func TestInput(*testing.T) {
	engine.Boot(nil)

	r := reader.New("oh")

	engine.Evaluate(job.New(process.Group()), r.Scan(`
if $True {
	debug 1
	debug 2
	debug $SHELL
} else {
	debug 3
	debug 4
}
`))

	engine.Evaluate(job.New(process.Group()), r.Scan(`
define g (method () = {
	return 42
	debug "we shouldn't see this"
})
`))

	engine.Evaluate(job.New(process.Group()), r.Scan(`
debug (g)
	`))

	engine.Evaluate(job.New(process.Group()), r.Scan(`
define f (method (a) = {
	debug $a
})
`))

	engine.Evaluate(job.New(process.Group()), r.Scan(`
f (define x 7)
`))
}

func TestObject(*testing.T) {
	engine.Boot(nil)

	r := reader.New("oh")

	engine.Evaluate(job.New(process.Group()), r.Scan(`
define l (cons 1 2)
`))
	engine.Evaluate(job.New(process.Group()), r.Scan(`
debug (l head)
`))
}

func TestStackTrace(*testing.T) {
	engine.Boot(nil)

	j := job.New(process.Group())
	r := reader.New("oh")

	r.Scan("define f: method () = {\n")
	r.Scan("    stack-trace\n")
	r.Scan("    debug leaving f\n")
	engine.Evaluate(j, r.Scan("}\n"))
	r.Scan("define g: method () = {\n")
	r.Scan("    f\n")
	r.Scan("    debug leaving g\n")
	engine.Evaluate(j, r.Scan("}\n"))
	engine.Evaluate(j, r.Scan("g\n"))
}

func TestThrow(*testing.T) {
	engine.Boot(nil)

	j := job.New(process.Group())
	r := reader.New("oh")

	engine.Evaluate(j, r.Scan(`
define f: method () = {
    throw blah
}
`))
	engine.Evaluate(j, r.Scan(`
define g: method () = {
    export throw: method (msg) = {
		debug (list new throw $msg)
	}
	f
}
`))
	engine.Evaluate(j, r.Scan("g\n"))
}
