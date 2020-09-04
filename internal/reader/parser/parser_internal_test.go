package parser

import (
	"testing"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/engine/boot"
	"github.com/michaelmacinnis/oh/internal/reader/lexer"
)

func TestBananaClipCons(t *testing.T) {
	check(t, "(|cons 1 2|)\n")
}

func TestBananaClipConsNils(t *testing.T) {
	check(t, "(|cons () ()|)\n")
}

func TestBananaClipNil(t *testing.T) {
	check(t, "()\n")
}

func TestBananaClipNumber(t *testing.T) {
	check(t, "(|number 42|)\n")
}

func check(t *testing.T, s string) {
	l := lexer.New("test")

	l.Scan(s)

	p := ""

	New(func(c cell.I) {
		s := literal.String(c) + "\n"
		p += s
	}, l.Token).Parse()

	m := lexer.New("test")

	m.Scan(p)

	r := ""

	New(func(c cell.I) {
		s := literal.String(c) + "\n"
		r += s
	}, m.Token).Parse()

	if p != r {
		t.Fatalf("Parsed (%s) and reparsed (%s) do not match", p, r)
	}
}

// TODO: Convert these into table-driven tests.
// TODO: Write tests that don't involve reparsing.

func TestBackground(t *testing.T) {
	check(t, "sleep 5; echo tea is ready!&\n")
}

func TestBoot(t *testing.T) {
	check(t, boot.Script())
}

func TestMultipleRedirections(t *testing.T) {
	check(t, "tr ' ' '\\n' < foo > bar\n")
}

func TestPipe(t *testing.T) {
	check(t, "ls | grep .go\n")
}

func TestPreStatementAssignment(t *testing.T) {
	check(t, "GOOS=linux GOARCH=ppc64 ./bootstrap.bash\n")
}

func TestSimpleCommand(t *testing.T) {
	check(t, "ls -la\n")
}

func TestTrailingDollar(t *testing.T) {
	check(t, "go test -run=^$\n")
}
