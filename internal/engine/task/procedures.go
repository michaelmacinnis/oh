// Released under an MIT license. See LICENSE.

package task

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/scope"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/obj"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/validate"
	"github.com/michaelmacinnis/oh/internal/system/options"
)

// TODO: Move this back into action.go.

// Builtins.

func cd(t *T) Op {
	dir := ""

	if t.code == pair.Null {
		_, r := t.frame.Resolve("HOME")
		dir = common.String(r.Get())
	} else {
		dir = t.tildeExpand(common.String(pair.Car(t.code)))
	}

	if dir == "-" {
		_, r := t.frame.Resolve("OLDPWD")
		dir = common.String(r.Get())
	}

	return t.Return(t.Chdir(dir))
}

// Methods.

func interpolate(t *T) Op {
	v := validate.Fixed(t.code, 1, 1)

	e := t.frame.Scope()

	b := bound(t.Result())

	s := b.self

	if scope.To(s).Expose() == e {
		s = e
	}

	cb := func(ref string) string {
		if ref == "$$" {
			return "$"
		}

		name := ref[1:]
		if name[0] == '{' {
			name = name[1 : len(name)-1]
		}

		_, r := t.frame.Resolve(name)
		if r == nil {
			panic("'" + name + "' undefined")
		}

		return common.String(r.Get())
	}

	r := regexp.MustCompile(`(?:\$\$)|(?:\${.+?})|(?:\$[0-9A-Z_a-z]+)`)

	return t.Return(str.New(r.ReplaceAllStringFunc(common.String(v[0]), cb)))
}

func _lookup_(t *T) Op { //nolint:golint
	k := literal.String(pair.Car(t.code))
	s, r := t.frame.Resolve(k)

	if r == nil {
		panic(k + " not defined")
	}

	v := r.Get()
	if c, ok := v.(command); ok {
		v = bind(c, s)
	}

	return t.Return(v)
}

func _stack_trace_(t *T) Op { //nolint:golint,stylecheck
	type point struct {
		loc string
		txt string
	}

	max := 0
	trace := []*point{}

	for s := t.stack; s != done; s = s.stack {
		if r, ok := s.op.(*registers); ok {
			if r.frame != nil {
				l := r.frame.Loc()
				n := strconv.Itoa
				p := &point{
					loc: l.Name + ":" + n(l.Line) + ":" + n(l.Char) + ":",
					txt: l.Text,
				}

				trace = append(trace, p)
			}
		}
	}

	if options.Script() {
		trace = trace[:len(trace)-3]
	}

	for _, p := range trace {
		sz := len(p.loc)
		if sz > max {
			max = sz
		}
	}

	depth := 0
	for i := len(trace) - 1; i >= 0; i-- {
		p := trace[i]
		sz := len(p.loc)

		println(p.loc + strings.Repeat(" ", 2*depth+max-sz+1) + p.txt)

		depth++
	}

	return t.PreviousOp()
}

func fatal(t *T) Op {
	t.stack = done

	return t.Return(pair.Car(t.code))
}

func resolves(t *T) Op {
	k := literal.String(pair.Car(t.code))

	_, r := t.frame.Resolve(k)

	return t.Return(boolean.Bool(r != nil))
}

// Syntax.

func builtin(t *T) Op {
	return t.Return(bind((*Builtin)(t.Closure()), obj.New(t.frame.Scope())))
}

func method(t *T) Op {
	return t.Return(bind((*Method)(t.Closure()), obj.New(t.frame.Scope())))
}

func syntax(t *T) Op {
	return t.Return(bind((*Syntax)(t.Closure()), obj.New(t.frame.Scope())))
}
