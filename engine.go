/* released under an MIT-style license. See LICENSE. */

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const (
	psNone = 0

	psChangeScope = SaveMax + iota
	psCreateModule

	psDoEvalArguments
	psDoEvalArgumentsBC
	psDoEvalCommand

	psEvalAccess
	psEvalArguments
	psEvalArgumentsBC
	psEvalBlock
	psEvalCommand
	psEvalElement
	psEvalElementBC
	psEvalTopBlock
	psEvalWhileBody
	psEvalWhileTest

	psExecApplicative
	psExecBuiltin
	psExecDefine
	psExecDynamic
	psExecExternal
	psExecIf
	psExecImport
	psExecSource
	psExecObject
	psExecOperative
	psExecPublic
	psExecSet
	psExecSetenv
	psExecSplice

	/* Commands. */
	psBlock
	psBuiltin
	psDefine
	psDynamic
	psIf
	psImport
	psMethod
	psObject
	psPublic
	psQuote
	psReturn
	psSet
	psSetenv
	psSource
	psSyntax
	psSpawn
	psSplice
	psWhile

	/* Operators. */
	psBackground
	psBacktick

	psAppendStdout
	psAppendStderr
	psPipeChild
	psPipeParent
	psPipeStderr
	psPipeStdout
	psRedirectCleanup
	psRedirectSetup
	psRedirectStderr
	psRedirectStdin
	psRedirectStdout

	psMax
)

var proc0 *Process
var block0 Cell

func channel(p *Process, r, w *os.File) Interface {
	c, ch := NewLexicalScope(p.Lexical), NewChannel(r, w)

	var read Function = func(p *Process, args Cell) bool {
		SetCar(p.Scratch, ch.Read())
		return false
	}

	var readline Function = func(p *Process, args Cell) bool {
		SetCar(p.Scratch, ch.ReadLine())
		return false
	}

	var write Function = func(p *Process, args Cell) bool {
		ch.Write(args)
		SetCar(p.Scratch, True)
		return false
	}

	c.Public(NewSymbol("guts"), ch)
	c.Public(NewSymbol("read"), method(read, Null, c))
	c.Public(NewSymbol("readline"), method(readline, Null, c))
	c.Public(NewSymbol("write"), method(write, Null, c))

	return NewObject(c)
}

func debug(p *Process, s string) {
	fmt.Printf("%s: p.Code = %v, p.Scratch = %v\n", s, p.Code, p.Scratch)
}

func expand(args Cell) Cell {
	list := Null

	for args != Null {
		c := Car(args)

		s := Raw(c)
		if _, ok := c.(*Symbol); ok {
			if s[:1] == "~" {
				s = filepath.Join(os.Getenv("HOME"), s[1:])
			}

			if strings.IndexAny(s, "*?[") != -1 {
				m, err := filepath.Glob(s)
				if err != nil || len(m) == 0 {
					panic("no matches found: " + s)
				}

				for _, e := range m {
					if e[0] != '.' || s[0] == '.' {
						list = AppendTo(list, NewSymbol(e))
					}
				}
			} else {
				list = AppendTo(list, NewSymbol(s))
			}
		} else {
			list = AppendTo(list, NewSymbol(s))
		}
		args = Cdr(args)
	}

	return list
}

func external(p *Process, args Cell) bool {
	name, err := exec.LookPath(Raw(Car(p.Scratch)))

	SetCar(p.Scratch, False)

	if err != nil {
		panic(err)
	}

	argv := []string{name}

	for args = expand(args); args != Null; args = Cdr(args) {
		argv = append(argv, Car(args).String())
	}

	c := Resolve(p.Lexical, p.Dynamic, NewSymbol("$cwd"))
	dir := c.GetValue().String()

	var fd []*os.File //{os.Stdin, os.Stdout, os.Stderr}

	fd = append(fd,
		rpipe(Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdin")).GetValue()),
		wpipe(Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdout")).GetValue()),
		wpipe(Resolve(p.Lexical, p.Dynamic, NewSymbol("$stderr")).GetValue()))

	proc, err := os.StartProcess(name, argv, &os.ProcAttr{dir, nil, fd, nil})
	if err != nil {
		panic(err)
	}

	var status int64 = 0

	msg, err := proc.Wait(0)
	status = int64(msg.WaitStatus.ExitStatus())

	SetCar(p.Scratch, NewStatus(status))

	return false
}

func function(body, param Cell, scope *Scope) *Applicative {
	return NewApplicative(NewClosure(body, param, scope), nil)
}

func method(body, param Cell, scope *Scope) *Applicative {
	return NewApplicative(NewClosure(body, param, scope), scope)
}

func module(f string) (string, error) {
	i, err := os.Stat(f)
	if err != nil {
		return "", err
	}

	m := "$" + f + "-" + i.Name() + "-" +
		strconv.FormatInt(i.Size(), 10) + "-" +
		strconv.Itoa(i.ModTime().Second()) + "-" +
		strconv.Itoa(i.ModTime().Nanosecond())

	return m, nil
}

func next(p *Process) bool {
	switch Car(p.Scratch).(type) {
	case *Applicative:

		body := Car(p.Scratch).(*Applicative).Func.Body

		switch t := body.(type) {
		case Function:
			p.ReplaceState(psExecBuiltin)

		case *Integer:
			p.ReplaceState(t.Int())
			return true

		default:
			p.ReplaceState(psExecApplicative)
		}

		return false

	case *Operative:
		p.ReplaceState(psExecOperative)
	}

	return true
}

func rpipe(c Cell) *os.File {
	r := Resolve(c.(Interface).Expose(), nil, NewSymbol("guts"))
	return r.GetValue().(*Channel).ReadEnd()
}

func run(p *Process) {
	defer func(saved Process) {
		r := recover()
		if r == nil {
			return
		}

		fmt.Printf("oh: %v\n", r)

		*p = saved

		p.Code = Null
		p.Scratch = Cons(False, p.Scratch)
		p.Stack = Cdr(p.Stack)
	}(*p)

	for p.Stack != Null {
		switch state := p.GetState(); state {
		case psNone:
			return

		case psDoEvalCommand:
			switch Car(p.Scratch).(type) {
			case *String, *Symbol:
				p.ReplaceState(psExecExternal)

			default:
				if next(p) {
					continue
				}
			}

			if p.GetState() == psExecExternal ||
				p.GetState() == psExecApplicative &&
					Car(p.Scratch).(*Applicative).Self == nil {
				p.NewState(psEvalArgumentsBC)
			} else {
				p.NewState(psEvalArguments)
			}

			fallthrough
		case psEvalArguments, psEvalArgumentsBC:
			p.Scratch = Cons(nil, p.Scratch)

			if p.GetState() == psEvalArgumentsBC {
				p.ReplaceState(psDoEvalArgumentsBC)
			} else {
				p.ReplaceState(psDoEvalArguments)
			}

			fallthrough
		case psDoEvalArguments, psDoEvalArgumentsBC:
			if p.Code == Null {
				break
			}

			state = p.GetState()

			p.SaveState(SaveCode, Cdr(p.Code))
			p.Code = Car(p.Code)

			if state == psDoEvalArgumentsBC {
				p.NewState(psEvalElementBC)
			} else {
				p.NewState(psEvalElement)
			}

			fallthrough
		case psEvalElement, psEvalElementBC:
			if p.Code == Null {
				p.Scratch = Cons(p.Code, p.Scratch)
				break
			} else if IsCons(p.Code) {
				if IsAtom(Cdr(p.Code)) {
					p.ReplaceState(psEvalAccess)
				} else {
					p.ReplaceState(psEvalCommand)
					continue
				}
			} else if sym, ok := p.Code.(*Symbol); ok {
				c := Resolve(p.Lexical, p.Dynamic, sym)
				if c == nil ||
					p.GetState() == psEvalElementBC &&
						!IsSimple(c.GetValue()) {
					p.Scratch = Cons(sym, p.Scratch)
				} else {
					p.Scratch = Cons(c.GetValue(), p.Scratch)
				}
				break
			} else {
				p.Scratch = Cons(p.Code, p.Scratch)
				break
			}

			fallthrough
		case psEvalAccess:
			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.NewState(psEvalElement)
			p.NewState(psChangeScope)
			p.SaveState(SaveCode, Cdr(p.Code))

			p.Code = Car(p.Code)

			p.NewState(psEvalElement)
			continue

		case psEvalTopBlock:
			if p.Code == block0 {
				return
			}

			p.SaveState(SaveCode, Cdr(p.Code))
			p.NewState(psEvalCommand)

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)
			continue

		case psBlock:
			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.NewScope(p.Dynamic, p.Lexical)

			p.NewState(psEvalBlock)

			fallthrough
		case psEvalBlock:
			if !IsCons(p.Code) || !IsCons(Car(p.Code)) {
				break
			}

			if Cdr(p.Code) == Null || !IsCons(Cadr(p.Code)) {
				p.ReplaceState(psEvalCommand)
			} else {
				p.SaveState(SaveCode, Cdr(p.Code))
				p.NewState(psEvalCommand)
			}

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			fallthrough
		case psEvalCommand:
			p.ReplaceState(psDoEvalCommand)
			p.SaveState(SaveCode, Cdr(p.Code))

			p.Code = Car(p.Code)

			p.NewState(psEvalElement)
			continue

		case psExecApplicative:
			args := p.Arguments()

			m := Car(p.Scratch).(*Applicative)
			if m.Self == nil {
				args = expand(args)
			}

			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.Code = m.Func.Body
			p.NewScope(p.Dynamic, m.Func.Lexical)

			param := m.Func.Param
			for args != Null && param != Null {
				p.Lexical.Public(Car(param), Car(args))
				args, param = Cdr(args), Cdr(param)
			}
			p.Lexical.Public(NewSymbol("$args"), args)
			p.Lexical.Public(NewSymbol("$self"), m.Self)
			p.Lexical.Public(NewSymbol("return"),
				p.Continuation(psReturn))

			p.NewState(psEvalBlock)
			continue

		case psExecOperative:
			args := p.Code
			env := p.Lexical

			m := Car(p.Scratch).(*Operative)
			if m.Self == nil {
				args = expand(args)
			}

			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.Code = m.Func.Body
			p.NewScope(p.Dynamic, m.Func.Lexical)

			param := m.Func.Param
			if param != Null {
				p.Lexical.Public(Car(param), env)
			}
			p.Lexical.Public(NewSymbol("$args"), args)
			p.Lexical.Public(NewSymbol("$self"), m.Self)
			p.Lexical.Public(NewSymbol("return"),
				p.Continuation(psReturn))

			p.NewState(psEvalBlock)
			continue

		case psSet:
			p.RemoveState()
			p.Scratch = Cdr(p.Scratch)

			s := Car(p.Code)
			if !IsCons(s) {
				p.NewState(psExecSet)
				p.SaveState(SaveCode, s)
			} else {
				p.SaveState(SaveDynamic | SaveLexical)
				p.NewState(psExecSet)
				p.SaveState(SaveCode, Cdr(s))
				p.NewState(psChangeScope)
				p.NewState(psEvalElement)
				p.SaveState(SaveCode, Car(s))
			}

			p.NewState(psEvalElement)

			p.Code = Cadr(p.Code)
			continue

		case psExecSet:
			k := p.Code.(*Symbol)
			r := Resolve(p.Lexical, p.Dynamic, k)
			if r == nil {
				panic("'" + k.String() + "' is not defined")
			}

			r.SetValue(Car(p.Scratch))

		case psDefine, psPublic:
			p.RemoveState()

			l := Car(p.Scratch).(*Applicative).Self
			if p.Lexical != l {
				p.SaveState(SaveLexical)
				p.Lexical = l
			}

			if state == psDefine {
				p.NewState(psExecDefine)
			} else {
				p.NewState(psExecPublic)
			}

			k := Car(p.Code)

			p.Code = Cadr(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.SaveState(SaveCode|SaveLexical, k)
			p.NewState(psEvalElement)
			continue

		case psExecDefine, psExecPublic:
			if state == psDefine {
				p.Lexical.Define(p.Code, Car(p.Scratch))
			} else {
				p.Lexical.Public(p.Code, Car(p.Scratch))
			}

		case psDynamic, psSetenv:
			k := Car(p.Code)

			if state == psSetenv {
				if !strings.HasPrefix(k.String(), "$") {
					break
				}
				p.ReplaceState(psExecSetenv)
			} else {
				p.ReplaceState(psExecDynamic)
			}

			p.Code = Cadr(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.SaveState(SaveCode|SaveDynamic, k)
			p.NewState(psEvalElement)
			continue

		case psExecDynamic, psExecSetenv:
			k := p.Code
			v := Car(p.Scratch)

			if state == psExecSetenv {
				s := Raw(v)
				os.Setenv(strings.TrimLeft(k.String(), "$"), s)
			}

			p.Dynamic.Define(k, v)

		case psWhile:
			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.NewState(psEvalWhileTest)

			fallthrough
		case psEvalWhileTest:
			p.ReplaceState(psEvalWhileBody)
			p.SaveState(SaveCode, p.Code)

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(psEvalElement)
			continue

		case psEvalWhileBody:
			if !Car(p.Scratch).Bool() {
				break
			}

			p.ReplaceState(psEvalWhileTest)
			p.SaveState(SaveCode, p.Code)

			p.Code = Cdr(p.Code)

			p.NewState(psEvalBlock)
			continue

		case psChangeScope:
			p.Dynamic = nil
			p.Lexical = Car(p.Scratch).(Interface)
			p.Scratch = Cdr(p.Scratch)

		case psExecBuiltin:
			args := p.Arguments()

			m := Car(p.Scratch).(*Applicative)
			if m.Self == nil {
				args = expand(args)
			}

			if m.Func.Body.(Function)(p, args) {
				continue
			}

		case psExecExternal:
			args := p.Arguments()

			if external(p, args) {
				continue
			}

		case psExecIf:
			if !Car(p.Scratch).Bool() {
				p.Code = Cdr(p.Code)

				for Car(p.Code) != Null &&
					!IsAtom(Car(p.Code)) {
					p.Code = Cdr(p.Code)
				}

				p.Code = Cdr(p.Code)
			}

			if p.Code == Null {
				break
			}

			p.ReplaceState(psEvalBlock)
			continue

		case psExecImport:
			n := Raw(Car(p.Scratch))

			k, err := module(n)
			if err != nil {
				SetCar(p.Scratch, False)
				break
			}

			v := Resolve(p.Lexical, p.Dynamic, NewSymbol(k))
			if v != nil {
				SetCar(p.Scratch, v.GetValue())
				break
			}

			p.ReplaceState(psCreateModule)
			p.SaveState(SaveCode, NewSymbol(n))
			p.NewState(psExecSource)

			fallthrough
		case psExecSource:
			f, err := os.OpenFile(
				Raw(Car(p.Scratch)),
				os.O_RDONLY, 0666)
			if err != nil {
				panic(err)
			}

			p.Code = Null
			ParseFile(f, func(c Cell) {
				p.Code = AppendTo(p.Code, c)
			})

			if state == psExecImport {
				p.RemoveState()
				p.SaveState(SaveDynamic | SaveLexical)

				p.NewScope(p.Dynamic, p.Lexical)

				p.NewState(psExecObject)
				p.NewState(psEvalBlock)
			} else {
				if p.Code == Null {
					break
				}

				p.ReplaceState(psEvalBlock)
			}
			continue

		case psCreateModule:
			k, _ := module(p.Code.String())

			s := p.Lexical
			for s.Prev() != nil {
				s = s.Prev()
			}
			p.Lexical.Define(NewSymbol(k), Car(p.Scratch))

		case psExecObject:
			SetCar(p.Scratch, NewObject(p.Lexical))

		case psExecSplice:
			l := Car(p.Scratch)
			p.Scratch = Cdr(p.Scratch)

			if !IsCons(l) {
				break
			}

			for l != Null {
				p.Scratch = Cons(Car(l), p.Scratch)
				l = Cdr(l)
			}

			/* Command states */
		case psBackground:
			child := NewProcess(psNone, p.Dynamic, p.Lexical)

			child.NewState(psEvalCommand)

			child.Code = Car(p.Code)
			SetCar(p.Scratch, True)

			go run(child)

		case psBacktick:
			c := channel(p, nil, nil)

			child := NewProcess(psNone, p.Dynamic, p.Lexical)

			child.NewState(psPipeChild)

			s := NewSymbol("$stdout")
			child.SaveState(SaveCode, s)

			child.Code = Car(p.Code)
			child.Dynamic.Define(s, c)

			child.NewState(psEvalCommand)

			go run(child)

			b := bufio.NewReader(rpipe(c))

			l := Null

			done := false
			line, err := b.ReadString('\n')
			for !done {
				if err != nil {
					done = true
				}

				line = strings.Trim(line, " \t\n")

				if len(line) > 0 {
					l = AppendTo(l, NewString(line))
				}

				line, err = b.ReadString('\n')
			}

			SetCar(p.Scratch, l)

		case psBuiltin, psMethod, psSyntax:
			param := Null
			for !IsCons(Car(p.Code)) {
				param = Cons(Car(p.Code), param)
				p.Code = Cdr(p.Code)
			}

			if state == psBuiltin {
				SetCar(
					p.Scratch,
					function(p.Code, Reverse(param), p.Lexical.Expose()))
			} else if state == psMethod {
				SetCar(
					p.Scratch,
					method(p.Code, Reverse(param), p.Lexical.Expose()))
			} else {
				SetCar(
					p.Scratch,
					syntax(p.Code, Reverse(param), p.Lexical.Expose()))
			}

		case psIf:
			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.NewScope(p.Dynamic, p.Lexical)

			p.NewState(psExecIf)
			p.SaveState(SaveCode, Cdr(p.Code))
			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(psEvalElement)
			continue

		case psImport, psSource:
			if state == psImport {
				p.ReplaceState(psExecImport)
			} else {
				p.ReplaceState(psExecSource)
			}

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(psEvalElement)
			continue

		case psObject:
			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.NewScope(p.Dynamic, p.Lexical)

			p.NewState(psExecObject)
			p.NewState(psEvalBlock)
			continue

		case psQuote:
			SetCar(p.Scratch, Car(p.Code))

		case psReturn:
			p.Code = Car(p.Code)

			m := Car(p.Scratch).(*Applicative)
			p.Scratch = Car(m.Func.Param)
			p.Stack = Cadr(m.Func.Param)

			p.NewState(psEvalElement)
			continue

		case psSpawn:
			child := NewProcess(psNone, p.Dynamic, p.Lexical)

			child.Scratch = Cons(Null, child.Scratch)
			child.NewState(psEvalBlock)

			child.Code = p.Code

			go run(child)

		case psSplice:
			p.ReplaceState(psExecSplice)

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(psEvalElement)
			continue

		case psPipeStderr, psPipeStdout:
			p.RemoveState()
			p.SaveState(SaveDynamic)

			c := channel(p, nil, nil)

			child := NewProcess(psNone, p.Dynamic, p.Lexical)

			child.NewState(psPipeChild)

			var s *Symbol
			if state == psPipeStderr {
				s = NewSymbol("$stderr")
			} else {
				s = NewSymbol("$stdout")
			}
			child.SaveState(SaveCode, s)

			child.Code = Car(p.Code)
			child.Dynamic.Define(s, c)

			child.NewState(psEvalCommand)

			go run(child)

			p.Code = Cadr(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewScope(p.Dynamic, p.Lexical)

			p.Dynamic.Define(NewSymbol("$stdin"), c)

			p.NewState(psPipeParent)
			p.NewState(psEvalCommand)
			continue

		case psPipeChild:
			c := Resolve(p.Lexical, p.Dynamic, p.Code.(*Symbol)).GetValue()
			wpipe(c).Close()

		case psPipeParent:
			c := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdin")).GetValue()
			rpipe(c).Close()

		case psAppendStderr, psAppendStdout, psRedirectStderr,
			psRedirectStdin, psRedirectStdout:
			p.RemoveState()
			p.SaveState(SaveDynamic)

			initial := NewInteger(state)

			p.NewState(psRedirectCleanup)
			p.NewState(psEvalCommand)
			p.SaveState(SaveCode, Cadr(p.Code))
			p.NewState(psRedirectSetup)
			p.SaveState(SaveCode, initial)

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewScope(p.Dynamic, p.Lexical)

			p.NewState(psEvalElement)
			continue

		case psRedirectSetup:
			flags, name := 0, ""
			initial := p.Code.(Atom).Int()

			switch initial {
			case psAppendStderr:
				flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
				name = "$stderr"

			case psAppendStdout:
				flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
				name = "$stdout"

			case psRedirectStderr:
				flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
				name = "$stderr"

			case psRedirectStdin:
				flags = os.O_RDONLY
				name = "$stdin"

			case psRedirectStdout:
				flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
				name = "$stdout"
			}

			c, ok := Car(p.Scratch).(Interface)
			if !ok {
				n := Raw(Car(p.Scratch))

				f, err := os.OpenFile(n, flags, 0666)
				if err != nil {
					panic(err)
				}

				if name == "$stdin" {
					c = channel(p, f, nil)
				} else {
					c = channel(p, nil, f)
				}
				SetCar(p.Scratch, c)

				r := Resolve(c, nil, NewSymbol("guts"))
				ch := r.GetValue().(*Channel)

				ch.Implicit = true
			}

			p.Dynamic.Define(NewSymbol(name), c)

		case psRedirectCleanup:
			c := Cadr(p.Scratch).(Interface)
			r := Resolve(c, nil, NewSymbol("guts"))
			ch := r.GetValue().(*Channel)

			if ch.Implicit {
				ch.Close()
			}

			SetCdr(p.Scratch, Cddr(p.Scratch))

		default:
			if state >= SaveMax {
				panic(fmt.Sprintf("command not found: %s", p.Code))
			} else {
				p.RestoreState()
			}
		}

		p.RemoveState()
	}
}

func syntax(body, param Cell, scope *Scope) *Operative {
	return NewOperative(NewClosure(body, param, scope), scope)
}

func wpipe(c Cell) *os.File {
	w := Resolve(c.(Interface).Expose(), nil, NewSymbol("guts"))
	return w.GetValue().(*Channel).WriteEnd()
}

func Evaluate(c Cell) {
	SetCar(block0, c)
	SetCdr(block0, Cons(nil, Null))
	block0 = Cdr(block0)

	proc0.SaveState(SaveCode, block0)
	proc0.NewState(psEvalCommand)
	proc0.Code = c

	run(proc0)

	if proc0.Stack == Null {
		os.Exit(ExitStatus())
	}

	proc0.Scratch = Cdr(proc0.Scratch)
}

func ExitStatus() int {
	s, ok := Car(proc0.Scratch).(*Status)
	if !ok {
		return 0
	}
	return int(s.Int())
}

func Start() {
	block0 = Cons(nil, Null)

	proc0 = NewProcess(psEvalTopBlock, nil, nil)

	proc0.Code = block0
	proc0.Scratch = Cons(NewStatus(0), proc0.Scratch)

	e, s := proc0.Dynamic, proc0.Lexical.Expose()

	e.Define(NewSymbol("$stdin"), channel(proc0, os.Stdin, nil))
	e.Define(NewSymbol("$stdout"), channel(proc0, nil, os.Stdout))
	e.Define(NewSymbol("$stderr"), channel(proc0, nil, os.Stderr))

	if wd, err := os.Getwd(); err == nil {
		e.Define(NewSymbol("$cwd"), NewSymbol(wd))
	}

	s.DefineState("block", psBlock)
	s.DefineState("backtick", psBacktick)
	s.DefineState("define", psDefine)
	s.DefineState("dynamic", psDynamic)
	s.DefineState("builtin", psBuiltin)
	s.DefineState("if", psIf)
	s.DefineState("import", psImport)
	s.DefineState("source", psSource)
	s.DefineState("method", psMethod)
	s.DefineState("object", psObject)
	s.DefineState("quote", psQuote)
	s.DefineState("set", psSet)
	s.DefineState("setenv", psSetenv)
	s.DefineState("spawn", psSpawn)
	s.DefineState("splice", psSplice)
	s.DefineState("syntax", psSyntax)
	s.DefineState("while", psWhile)

	s.PublicState("public", psPublic)

	s.DefineState("background", psBackground)
	s.DefineState("pipe-stdout", psPipeStdout)
	s.DefineState("pipe-stderr", psPipeStderr)
	s.DefineState("redirect-stdin", psRedirectStdin)
	s.DefineState("redirect-stdout", psRedirectStdout)
	s.DefineState("redirect-stderr", psRedirectStderr)
	s.DefineState("append-stdout", psAppendStdout)
	s.DefineState("append-stderr", psAppendStderr)

	/* Builtins. */
	s.DefineFunction("cd", func(p *Process, args Cell) bool {
		err := os.Chdir(Raw(Car(args)))
		status := 0
		if err != nil {
			status = 1
		}
		SetCar(p.Scratch, NewStatus(int64(status)))

		if wd, err := os.Getwd(); err == nil {
			p.Dynamic.Define(NewSymbol("$cwd"), NewSymbol(wd))
		}

		return false
	})
	s.DefineFunction("debug", func(p *Process, args Cell) bool {
		debug(p, "debug")

		return false
	})
	s.DefineFunction("exit", func(p *Process, args Cell) bool {
		var status int64 = 0

		a, ok := Car(args).(Atom)
		if ok {
			status = a.Status()
		}

		p.Scratch = List(NewStatus(status))
		p.Stack = Null

		return true
	})

	s.PublicMethod("child", func(p *Process, args Cell) bool {
		o := Car(p.Scratch).(*Applicative).Self.Expose()

		SetCar(p.Scratch, NewObject(NewLexicalScope(o)))

		return false
	})
	s.PublicMethod("clone", func(p *Process, args Cell) bool {
		o := Car(p.Scratch).(*Applicative).Self.Expose()

		SetCar(p.Scratch, NewObject(o.Copy()))

		return false
	})

	s.DefineMethod("apply", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Car(args))
		next(p)

		p.Scratch = Cons(nil, p.Scratch)
		for args = Cdr(args); args != Null; args = Cdr(args) {
			p.Scratch = Cons(Car(args), p.Scratch)
		}

		return true
	})
	s.DefineMethod("append", func(p *Process, args Cell) bool {
		/*
		 * NOTE: Our append works differently than Scheme's append.
		 *       To mimic Scheme's behavior used append l1 @l2 ... @ln
		 */

		/* TODO: We should just copy this list: ... */
		l := Car(args)

		/* TODO: ... and then set it's cdr to cdr(args). */
		argv := make([]Cell, 0)
		for args = Cdr(args); args != Null; args = Cdr(args) {
			argv = append(argv, Car(args))
		}

		SetCar(p.Scratch, Append(l, argv...))

		return false
	})
	s.DefineMethod("car", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Caar(args))

		return false
	})
	s.DefineMethod("cdr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdar(args))

		return false
	})
	s.DefineMethod("caar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Caaar(args))

		return false
	})
	s.DefineMethod("cadr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cadar(args))

		return false
	})
	s.DefineMethod("cdar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdaar(args))

		return false
	})
	s.DefineMethod("cddr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cddar(args))

		return false
	})
	s.DefineMethod("caaar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Car(Caaar(args)))

		return false
	})
	s.DefineMethod("caadr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Car(Cadar(args)))

		return false
	})
	s.DefineMethod("cadar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Car(Cdaar(args)))

		return false
	})
	s.DefineMethod("caddr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Car(Cddar(args)))

		return false
	})
	s.DefineMethod("cdaar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdr(Caaar(args)))

		return false
	})
	s.DefineMethod("cdadr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdr(Cadar(args)))

		return false
	})
	s.DefineMethod("cddar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdr(Cdaar(args)))

		return false
	})
	s.DefineMethod("cdddr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdr(Cddar(args)))

		return false
	})
	s.DefineMethod("caaaar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Caar(Caaar(args)))

		return false
	})
	s.DefineMethod("caaadr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Caar(Cadar(args)))

		return false
	})
	s.DefineMethod("caadar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Caar(Cdaar(args)))

		return false
	})
	s.DefineMethod("caaddr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Caar(Cddar(args)))

		return false
	})
	s.DefineMethod("cadaar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cadr(Caaar(args)))

		return false
	})
	s.DefineMethod("cadadr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cadr(Cadar(args)))

		return false
	})
	s.DefineMethod("caddar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cadr(Cdaar(args)))

		return false
	})
	s.DefineMethod("cadddr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cadr(Cddar(args)))

		return false
	})
	s.DefineMethod("cdaaar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdar(Caaar(args)))

		return false
	})
	s.DefineMethod("cdaadr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdar(Cadar(args)))

		return false
	})
	s.DefineMethod("cdadar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdar(Cdaar(args)))

		return false
	})
	s.DefineMethod("cdaddr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdar(Cddar(args)))

		return false
	})
	s.DefineMethod("cddaar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cddr(Caaar(args)))

		return false
	})
	s.DefineMethod("cddadr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cddr(Cadar(args)))

		return false
	})
	s.DefineMethod("cdddar", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cddr(Cdaar(args)))

		return false
	})
	s.DefineMethod("cddddr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cddr(Cddar(args)))

		return false
	})
	s.DefineMethod("cons", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cons(Car(args), Cadr(args)))

		return false
	})
	s.DefineMethod("eval", func(p *Process, args Cell) bool {
		if Cdr(args) != Null {
			p.RemoveState()
			p.SaveState(SaveDynamic | SaveLexical)

			p.NewState(psEvalElement)

			p.Lexical = Car(args).(Interface)
			p.Code = Cadr(args)
		} else {
			p.ReplaceState(psEvalElement)

			p.Code = Car(args)
		}

		p.Scratch = Cdr(p.Scratch)

		return true
	})
	s.DefineMethod("length", func(p *Process, args Cell) bool {
		var l int64 = 0

		switch c := Car(args); c.(type) {
		case *String, *Symbol:
			l = int64(len(Raw(c)))
		default:
			l = Length(c)
		}

		SetCar(p.Scratch, NewInteger(l))

		return false
	})
	s.DefineMethod("is-control", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsControl(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-digit", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsDigit(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-graphic", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsGraphic(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-letter", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLetter(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-lower", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLower(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-mark", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsMark(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-print", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPrint(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-punct", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPunct(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-space", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSpace(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-symbol", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSymbol(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-title", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsTitle(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("is-upper", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsUpper(int(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("list", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, args)

		return false
	})
	s.DefineMethod("list-to-string", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		SetCar(p.Scratch, NewString(s))

		return false
	})
	s.DefineMethod("list-to-symbol", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		SetCar(p.Scratch, NewSymbol(s))

		return false
	})
	s.DefineMethod("open", func(p *Process, args Cell) bool {
		name := Raw(Car(args))
		mode := Raw(Cadr(args))

		flags := os.O_CREATE

		if strings.IndexAny(mode, "r") != -1 {
			flags |= os.O_WRONLY
		} else if strings.IndexAny(mode, "aw") != -1 {
			flags |= os.O_RDONLY
		} else {
			flags |= os.O_RDWR
		}

		if strings.IndexAny(mode, "a") != -1 {
			flags |= os.O_APPEND
		}

		f, err := os.OpenFile(name, flags, 0666)
		if err != nil {
			panic(err)
		}

		SetCar(p.Scratch, channel(p, f, f))

		return false
	})
	s.DefineMethod("reverse", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Reverse(Car(args)))

		return false
	})
	s.DefineMethod("set-car", func(p *Process, args Cell) bool {
		SetCar(Car(args), Cadr(args))
		SetCar(p.Scratch, Cadr(args))

		return false
	})
	s.DefineMethod("set-cdr", func(p *Process, args Cell) bool {
		SetCdr(Car(args), Cadr(args))
		SetCar(p.Scratch, Cadr(args))

		return false
	})
	s.DefineMethod("sprintf", func(p *Process, args Cell) bool {
		f := Raw(Car(args))

		argv := []interface{}{}
		for l := Cdr(args); l != Null; l = Cdr(l) {
			switch t := Car(l).(type) {
			case *Boolean:
				argv = append(argv, *t)
			case *Integer:
				argv = append(argv, *t)
			case *Status:
				argv = append(argv, *t)
			case *Float:
				argv = append(argv, *t)
			default:
				argv = append(argv, Raw(t))
			}
		}

		s := fmt.Sprintf(f, argv...)
		SetCar(p.Scratch, NewString(s))

		return false
	})
	s.DefineMethod("substring", func(p *Process, args Cell) bool {
		var r Cell

		s := []int(Raw(Car(args)))

		start := int(Cadr(args).(Atom).Int())
		end := len(s)

		if Cddr(args) != Null {
			end = int(Caddr(args).(Atom).Int())
		}

		switch Car(args).(type) {
		case *String:
			r = NewString(string(s[start:end]))
		case *Symbol:
			r = NewSymbol(string(s[start:end]))
		default:
			r = Null
		}
		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("text-to-list", func(p *Process, args Cell) bool {
		l := Null
		for _, char := range Raw(Car(args)) {
			l = Cons(NewInteger(int64(char)), l)
		}

		SetCar(p.Scratch, Reverse(l))

		return false
	})
	s.DefineMethod("to-lower", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToLower(int(t.Int()))))
		case *String:
			r = NewString(strings.ToLower(Raw(t)))
		case *Symbol:
			r = NewSymbol(strings.ToLower(Raw(t)))
		default:
			r = NewInteger(0)
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("to-title", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToTitle(int(t.Int()))))
		case *String:
			r = NewString(strings.ToTitle(Raw(t)))
		case *Symbol:
			r = NewSymbol(strings.ToTitle(Raw(t)))
		default:
			r = NewInteger(0)
		}

		SetCar(p.Scratch, r)

		return false
	})
	s.DefineMethod("to-upper", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToUpper(int(t.Int()))))
		case *String:
			r = NewString(strings.ToUpper(Raw(t)))
		case *Symbol:
			r = NewSymbol(strings.ToUpper(Raw(t)))
		default:
			r = NewInteger(0)
		}

		SetCar(p.Scratch, r)

		return false
	})

	/* Predicates. */
	s.DefineMethod("is-atom", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, NewBoolean(IsAtom(Car(args))))

		return false
	})
	s.DefineMethod("is-boolean",
		func(p *Process, args Cell) bool {
			_, ok := Car(args).(*Boolean)
			SetCar(p.Scratch, NewBoolean(ok))

			return false
		})
	s.DefineMethod("is-channel",
		func(p *Process, args Cell) bool {
			o, ok := Car(args).(Interface)
			if ok {
				ok = false
				c := Resolve(o.Expose(), nil, NewSymbol("guts"))
				if c != nil {
					_, ok = c.GetValue().(*Channel)
				}
			}

			SetCar(p.Scratch, NewBoolean(ok))

			return false
		})
	s.DefineMethod("is-cons", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, NewBoolean(IsCons(Car(args))))

		return false
	})
	s.DefineMethod("is-float", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Float)
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-integer",
		func(p *Process, args Cell) bool {
			_, ok := Car(args).(*Integer)
			SetCar(p.Scratch, NewBoolean(ok))

			return false
		})
	s.DefineMethod("is-list", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, NewBoolean(IsList(Car(args))))

		return false
	})
	s.DefineMethod("is-method", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Applicative)
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-null", func(p *Process, args Cell) bool {
		ok := Car(args) == Null
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-number", func(p *Process, args Cell) bool {
		_, ok := Car(args).(Number)
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-object", func(p *Process, args Cell) bool {
		_, ok := Car(args).(Interface)
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-status", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Status)
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-string", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*String)
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-symbol", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Symbol)
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("is-text", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Symbol)
		if !ok {
			_, ok = Car(args).(*String)
		}
		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})

	/* Generators. */
	s.DefineMethod("boolean", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, NewBoolean(Car(args).Bool()))

		return false
	})
	s.DefineMethod("channel", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, channel(p, nil, nil))

		return false
	})
	s.DefineMethod("float", func(p *Process, args Cell) bool {
		SetCar(p.Scratch,
			NewFloat(Car(args).(Atom).Float()))

		return false
	})
	s.DefineMethod("integer", func(p *Process, args Cell) bool {
		SetCar(p.Scratch,
			NewInteger(Car(args).(Atom).Int()))

		return false
	})
	s.DefineMethod("status", func(p *Process, args Cell) bool {
		SetCar(p.Scratch,
			NewStatus(Car(args).(Atom).Status()))

		return false
	})
	s.DefineMethod("string", func(p *Process, args Cell) bool {
		SetCar(p.Scratch,
			NewString(Car(args).String()))

		return false
	})
	s.DefineMethod("symbol", func(p *Process, args Cell) bool {
		SetCar(p.Scratch,
			NewSymbol(Raw(Car(args))))

		return false
	})

	/* Relational. */
	s.DefineMethod("eq", func(p *Process, args Cell) bool {
		prev := Car(args)

		SetCar(p.Scratch, False)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args)

			if !prev.Equal(curr) {
				return false
			}

			prev = curr
		}

		SetCar(p.Scratch, True)
		return false
	})
	s.DefineMethod("ge", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		SetCar(p.Scratch, False)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if prev.Less(curr) {
				return false
			}

			prev = curr
		}

		SetCar(p.Scratch, True)
		return false
	})
	s.DefineMethod("gt", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		SetCar(p.Scratch, False)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if !prev.Greater(curr) {
				return false
			}

			prev = curr
		}

		SetCar(p.Scratch, True)
		return false
	})
	s.DefineMethod("is", func(p *Process, args Cell) bool {
		prev := Car(args)

		SetCar(p.Scratch, False)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args)

			if prev != curr {
				return false
			}

			prev = curr
		}

		SetCar(p.Scratch, True)
		return false
	})
	s.DefineMethod("le", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		SetCar(p.Scratch, False)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if prev.Greater(curr) {
				return false
			}

			prev = curr
		}

		SetCar(p.Scratch, True)
		return false
	})
	s.DefineMethod("lt", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		SetCar(p.Scratch, False)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if !prev.Less(curr) {
				return false
			}

			prev = curr
		}

		SetCar(p.Scratch, True)
		return false
	})
	s.DefineMethod("match", func(p *Process, args Cell) bool {
		pattern := Raw(Car(args))
		text := Raw(Cadr(args))

		ok, err := path.Match(pattern, text)
		if err != nil {
			panic(err)
		}

		SetCar(p.Scratch, NewBoolean(ok))

		return false
	})
	s.DefineMethod("ne", func(p *Process, args Cell) bool {
		/*
		 * This should really check to make sure no arguments are equal.
		 * Currently it only checks whether adjacent pairs are not equal.
		 */

		prev := Car(args)

		SetCar(p.Scratch, False)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args)

			if prev.Equal(curr) {
				return false
			}

			prev = curr
		}

		SetCar(p.Scratch, True)
		return false
	})
	s.DefineMethod("not", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, NewBoolean(!Car(args).Bool()))

		return false
	})

	/* Arithmetic. */
	s.DefineMethod("add", func(p *Process, args Cell) bool {
		acc := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Add(Car(args))

		}

		SetCar(p.Scratch, acc)
		return false
	})
	s.DefineMethod("sub", func(p *Process, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Subtract(Car(args))
		}

		SetCar(p.Scratch, acc)
		return false
	})
	s.DefineMethod("div", func(p *Process, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Divide(Car(args))
		}

		SetCar(p.Scratch, acc)
		return false
	})
	s.DefineMethod("mod", func(p *Process, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Modulo(Car(args))
		}

		SetCar(p.Scratch, acc)
		return false
	})
	s.DefineMethod("mul", func(p *Process, args Cell) bool {
		acc := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Multiply(Car(args))
		}

		SetCar(p.Scratch, acc)
		return false
	})

	s.Public(NewSymbol("$dynamic"), e)
	s.Public(NewSymbol("$lexical"), s)

	e.Define(NewSymbol("$$"), NewInteger(int64(os.Getpid())))

	/* Command-line arguments */
	args := Null
	if len(os.Args) > 1 {
		e.Define(NewSymbol("$0"), NewSymbol(os.Args[1]))

		for i, v := range os.Args[2:] {
			e.Define(NewSymbol("$"+strconv.Itoa(i+1)), NewSymbol(v))
		}

		for i := len(os.Args) - 1; i > 1; i-- {
			args = Cons(NewSymbol(os.Args[i]), args)
		}
	} else {
		e.Define(NewSymbol("$0"), NewSymbol(os.Args[0]))
	}
	e.Define(NewSymbol("$args"), args)

	/* Environment variables. */
	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		e.Define(NewSymbol("$"+kv[0]), NewSymbol(kv[1]))
	}

	Parse(bufio.NewReader(strings.NewReader(`
define and: syntax e {
    define l $args
    define r false
    while (not: is-null: car l) {
        set r: eval e: car l
        if (not r): return r
        set l: cdr l
    }
    return r
}
define echo: builtin: $stdout::write @$args
define expand: builtin: return $args
define for: method l m {
    define r: cons () ()
    define c r
    while (not: is-null l) {
        set-cdr c: cons (m: car l) ()
        set c: cdr c
        set l: cdr l
    }
    return: cdr r
}
define list-ref: method k x: car: list-tail k x
define list-tail: method k x {
    if k {
        list-tail (sub k 1): cdr x
    } else {
        return x
    }
}
define or: syntax e {
    define l $args
    define r false
    while (not: is-null: car l) {
	set r: eval e: car l
        if r: return r
        set l: cdr l
    }
    return r
}
define printf: method: echo: sprintf (car $args) @(cdr $args)
define read: builtin: $stdin::read
define readline: builtin: $stdin::readline
define write: method: $stdout::write @$args
`)), Evaluate)

	/* Read and execute rc script if it exists. */
	rc := filepath.Join(os.Getenv("HOME"), ".ohrc")
	if _, err := os.Stat(rc); err == nil {
		proc0.NewState(psEvalCommand)
		proc0.Code = List(NewSymbol("source"), NewSymbol(rc))

		run(proc0)

		proc0.Scratch = Cdr(proc0.Scratch)
	}
}
