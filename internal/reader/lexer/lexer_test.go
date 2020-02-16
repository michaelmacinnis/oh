package lexer

import (
	"testing"

	"github.com/michaelmacinnis/oh/internal/reader/token"
	"github.com/michaelmacinnis/oh/internal/type/loc"
)

func TestBackground(t *testing.T) {
	h := setup(t, "Background")

	h.scan("1 &\n",
		h.symbol("1"),
		h.literal(" "),
		h.literal("&"),
		h.literal("\n"),
		nil,
	)
}

func TestDollarDollar(t *testing.T) {
	h := setup(t, "DollarDollar")

	h.scan("$$\n",
		h.literal("$"),
		h.symbol("$"),
		h.literal("\n"),
		nil,
	)
}

func TestImplicitConcatenation(t *testing.T) {
	h := setup(t, "ImplicitConcatenation")

	h.scan("1'foo'\"bar\"2\n",
		h.symbol("1"),
		h.other(token.SingleQuoted, "'foo'"),
		h.other(token.DoubleQuoted, "\"bar\""),
		h.symbol("2"),
		h.literal("\n"),
		nil,
	)
}

func TestImplicitContinuation(t *testing.T) {
	h := setup(t, "ImplicitContinuation")

	for _, op := range []string{
		"&&", "|", "|&", "||",
	} {
		v := "1 " + op + "\n 2\n"
		h.scan(v,
			h.symbol("1"),
			h.literal(" "),
			h.literal(op),
			h.escapedNewline(),
			h.silentSpace(1),
			h.symbol("2"),
			h.literal("\n"),
			nil,
		)
	}
}

func TestInfixOperators(t *testing.T) {
	h := setup(t, "InfixOperators")

	for _, op := range []string{
		"<", ">", ">&", ">&|", ">>",
		">>&", ">|", "|<", "|>",
	} {
		v := "1 " + op + " 2\n"
		h.scan(v,
			h.symbol("1"),
			h.literal(" "),
			h.literal(op),
			h.silentSpace(1),
			h.symbol("2"),
			h.literal("\n"),
			nil,
		)
	}

	for _, op := range []string{
		"<", ">", ">&", ">&|", ">>",
		">>&", ">|", "|<", "|>",
	} {
		// The newline before '2' will cause
		// the parser to throw an error, but
		// we want to test that the lexer
		// doesn't treat these operators as
		// implicit line continuations.
		v := "1 " + op + "\n2\n"
		h.scan(v,
			h.symbol("1"),
			h.literal(" "),
			h.literal(op),
			h.literal("\n"),
			h.symbol("2"),
			h.literal("\n"),
			nil,
		)
	}
}

func TestMeta(t *testing.T) {
	h := setup(t, "Meta")

	h.scan("(|boolean false|)\n",
		h.literal("(|"),
		h.symbol("boolean"),
		h.literal(" "),
		h.symbol("false"),
		h.literal("|)"),
		h.literal("\n"),
		nil,
	)
}

func TestTrailingDollar(t *testing.T) {
	h := setup(t, "TrailingDollar")

	h.scan("1$ 2\n",
		h.symbol("1"),
		h.symbol("$"),
		h.literal(" "),
		h.symbol("2"),
		h.literal("\n"),
		nil,
	)
}

type harness struct {
	index  int
	lexer  *T
	source loc.T
	t      *testing.T
}

var skip = token.New(token.Error, "", loc.T{ //nolint:gochecknoglobals
	Char: 0,
	Line: 0,
	Name: "",
})

func setup(t *testing.T, label string) *harness {
	return &harness{
		index: 1,
		lexer: New(label),
		source: loc.T{
			Char: 1,
			Line: 1,
			Name: label,
		},
		t: t,
	}
}

func (h *harness) expect(tokens ...*token.T) {
	for _, e := range tokens {
		if e == skip {
			continue
		}

		a := h.lexer.Token()

		// TODO: Print actual and expected on command-line option.

		switch {
		case a == e:
			if a != nil {
				println(a.String(), e.String())
			}
			continue
		case a == nil:
			h.t.Fatalf("Expected %v but there are no tokens", e)
		case e == nil:
			h.t.Fatalf("Expected no tokens; got %v", a)
		case *a != *e:
			h.t.Fatalf("Expected %v; got %v", e, a)
		}
	}
}

func (h *harness) literal(s string) *token.T {
	h.source.Char = h.index
	h.index += len(s)

	id, found := map[string]token.Class{
		" ":   token.Space,
		"&":   token.Background,
		"&&":  token.Andf,
		"(|":  token.MetaOpen,
		"<":   token.Redirect,
		">":   token.Redirect,
		">&":  token.Redirect,
		">&|": token.Redirect,
		">>":  token.Redirect,
		">>&": token.Redirect,
		">|":  token.Redirect,
		"|":   token.Pipe,
		"|&":  token.Pipe,
		"|)":  token.MetaClose,
		"|<":  token.Substitute,
		"|>":  token.Substitute,
		"||":  token.Orf,
	}[s]
	if !found {
		id = token.Class(s[0])
	}

	if op := operator(s); op != "" {
		s = op
	}

	if s == "\n" {
		h.index = 1
	}

	t := token.New(id, s, h.source)

	if s == "\n" {
		h.source.Line++
	}

	return t
}

func (h *harness) escapedNewline() *token.T {
	h.index = 1
	h.source.Line++
	return skip
}

func (h *harness) silentSpace(n int) *token.T {
	h.index += n
	return skip
}

func (h *harness) other(id token.Class, s string) *token.T {
	h.source.Char = h.index
	h.index += len(s)
	return token.New(id, s, h.source)
}

func (h *harness) scan(s string, tokens ...*token.T) {
	h.lexer.Scan(s)
	h.expect(tokens...)
}

func (h *harness) symbol(s string) *token.T {
	h.source.Char = h.index
	h.index += len(s)
	return token.New(token.Symbol, s, h.source)
}
