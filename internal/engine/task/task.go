// Released under an MIT license. See LICENSE.

// Package task provides the machinery used by oh tasks.
package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/michaelmacinnis/oh/internal/adapted"
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/struct/frame"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
)

const debug = false

type monitor interface {
	Await(fn func(), t *T, ts ...*T)
	Launch(t *T, path string, argv []string, attr *os.ProcAttr) error
	Spawn(p, c *T, fn func())
	Stopped(t *T)
}

// T (task) encapsulates a thread of execution.
type T struct {
	monitor
	*registers
	*state
}

// New creates a new task.
func New(m monitor, c cell.I, f *frame.T) *T {
	t := &T{
		registers: &registers{
			code:  c,
			dump:  pair.Null,
			frame: f,
			stack: done,
		},
		monitor: m,
		state:   fresh(),
	}

	return t
}

func (t *T) CellValue(s string) cell.I {
	_, ref := t.frame.Resolve(s)
	if ref == nil {
		return nil
	}

	return ref.Get()
}

func (t *T) Chdir(s string) cell.I {
	rv := boolean.True

	_, r := t.frame.Resolve("PWD")
	oldwd := r.Get()

	err := os.Chdir(s)
	if err != nil {
		rv = boolean.False
	} else if wd, err := os.Getwd(); err == nil {
		t.frame.Scope().Export("PWD", sym.New(wd))
		t.frame.Scope().Export("OLDPWD", oldwd)
	}

	return rv
}

// Closure does as the name implies and creates a closure.
func (t *T) Closure() *Closure {
	slabel := pair.Car(t.code)
	t.code = pair.Cdr(t.code)

	plabels := slabel
	if sym.Is(slabel) {
		plabels = pair.Car(t.code)
		t.code = pair.Cdr(t.code)
	} else {
		slabel = pair.Null
	}

	// TODO: Check plabels is a list of symbols. Last element can be a list.

	equals := pair.Car(t.code)
	t.code = pair.Cdr(t.code)

	elabel := pair.Null
	if literal.String(equals) != "=" {
		elabel = equals
		equals = pair.Car(t.code)
		t.code = pair.Cdr(t.code)
	}

	if literal.String(equals) != "=" {
		panic("expected '='")
	}

	return &Closure{
		Body: t.code,
		Labels: Labels{
			Env:    elabel,
			Params: plabels,
			Self:   slabel,
		},
		Op:    Action(apply),
		Scope: t.frame.Scope(),
	}
}

// Environ returns key value pairs for stringable values in the form provided by os.Environ.
func (t *T) Environ() []string {
	environ := []string{}

	for f := t.registers.frame; f != nil; f = f.Previous() {
		for s := f.Scope(); s != nil; s = s.Enclosing() {
			environ = append(environ, s.Public().Environ()...)
		}
	}

	return environ
}

func (t *T) Interrupt() {
	t.state.Stop(func() {
		t.stack = done
	})
}

func (t *T) Return(c cell.I) Op {
	t.ReplaceResult(c)

	return t.PreviousOp()
}

// Run steps through a tasks operations until they are exhausted.
func (t *T) Run() {
	t.state.Started()
	defer t.state.Stopped()

	s := t.Op()
	for t.state.Runnable() && s != nil {
		s = t.Step(s)
	}

	t.monitor.Stopped(t)
}

// Step performs a single action and determines the next action.
func (t *T) Step(s Op) (op Op) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		t.code = list.New(sym.New("throw"), str.New(fmt.Sprintf("%v", r)))

		// Push a no-op so that the saved frame isn't collapsed away.
		t.PushOp(Action(nop))
		t.PushOp(&registers{frame: t.frame})

		op = t.PushOp(Action(EvalCommand))
	}()

	if debug {
		print("Stack: ")

		for p := t.stack; p != nil && p.op != nil; p = p.stack {
			print(opString(p.op))
			print(" ")
		}

		println("")
		print("Dump: ")

		for p := t.dump; p != pair.Null; p = pair.Cdr(p) {
			c := pair.Car(p)
			if c == nil {
				print("<nil> ")
			} else {
				print(pair.Car(p).Name())
				print(" ")
			}
		}

		println("")
		print("Code: ")

		println(literal.String(t.code))

		println("")
	}

	op = s.Perform(t)

	return op
}

func (t *T) Stop() {
	t.state.Stop(nil)
}

func (t *T) expand(args cell.I) cell.I {
	l := pair.Null

	for ; args != pair.Null; args = pair.Cdr(args) {
		c := pair.Car(args)

		if !sym.Is(c) {
			l = list.Append(l, c)

			continue
		}

		s := common.String(c)

		path := t.tildeExpand(s)
		if !strings.ContainsAny(path, "*?[") {
			l = list.Append(l, sym.New(path))

			continue
		}

		pwd := ""
		if !filepath.IsAbs(path) {
			pwd = t.stringValue("PWD")
			// path = filepath.Join(pwd, path)
			path = pwd + string(os.PathSeparator) + path
			pwd = filepath.Clean(pwd)
		}

		m, err := adapted.Glob(path)
		if err != nil || len(m) == 0 {
			panic("no matches found: " + s)
		}

		for _, v := range m {
			if pwd != "" {
				rel, err := filepath.Rel(pwd, v)
				if err == nil {
					v = rel
				}
			}

			l = list.Append(l, str.New(v))
		}
	}

	return l
}

func (t *T) stringValue(s string) string {
	v := t.CellValue(s)
	if v == nil {
		return ""
	}

	return common.String(v)
}

func (t *T) tildeExpand(s string) string {
	if !strings.HasPrefix(s, "~") {
		return s
	}

	return filepath.Join(t.stringValue("HOME"), s[1:])
}
