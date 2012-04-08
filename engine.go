/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

const (
	psNone = 0

	psChangeScope = SaveMax + iota

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
	psExecCommand
	psExecDefine
	psExecDynamic
	psExecFunction
	psExecIf
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
	psMethod
	psPublic
	psReturn
	psSet
	psSetenv
	psSyntax
	psSpawn
	psSplice
	psWhile

	psMax
)

var block0 Cell
var proc0 *Process

var interactive bool
var irq chan os.Signal
var next = map[int64]int64{
	psEvalArguments:   psEvalElement,
	psEvalArgumentsBC: psEvalElementBC,
	psDefine:          psExecDefine,
	psDynamic:         psExecDynamic,
	psPublic:          psExecPublic,
	psSetenv:          psExecSetenv,
}

func channel(p *Process, r, w *os.File, cap int) Interface {
	c, ch := NewLexicalScope(p.Lexical), NewChannel(r, w, cap)

	var rclose Function = func(p *Process, args Cell) bool {
		SetCar(p.Scratch, NewBoolean(ch.ReaderClose()))
		return false
	}

	var read Function = func(p *Process, args Cell) bool {
		SetCar(p.Scratch, ch.Read())
		return false
	}

	var readline Function = func(p *Process, args Cell) bool {
		SetCar(p.Scratch, ch.ReadLine())
		return false
	}

	var wclose Function = func(p *Process, args Cell) bool {
		SetCar(p.Scratch, NewBoolean(ch.WriterClose()))
		return false
	}

	var write Function = func(p *Process, args Cell) bool {
		ch.Write(args)
		SetCar(p.Scratch, True)
		return false
	}

	c.Public(NewSymbol("guts"), ch)
	c.Public(NewSymbol("reader-close"), method(rclose, Null, c))
	c.Public(NewSymbol("read"), method(read, Null, c))
	c.Public(NewSymbol("readline"), method(readline, Null, c))
	c.Public(NewSymbol("writer-close"), method(wclose, Null, c))
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
	p.Scratch = Cdr(p.Scratch)

	name, err := exec.LookPath(Raw(Car(p.Scratch)))

	SetCar(p.Scratch, False)

	if err != nil {
		panic(err)
	}

	argv := []string{name}

	for ; args != Null; args = Cdr(args) {
		argv = append(argv, Car(args).String())
	}

	c := Resolve(p.Lexical, p.Dynamic, NewSymbol("$cwd"))
	dir := c.GetValue().String()

	stdin := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdin")).GetValue()
	stdout := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdout")).GetValue()
	stderr := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stderr")).GetValue()

	fd := []*os.File{rpipe(stdin), wpipe(stdout), wpipe(stderr)}

	proc, err := os.StartProcess(name, argv, &os.ProcAttr{dir, nil, fd, nil})
	if err != nil {
		panic(err)
	}

	var status int64 = 0

	msg, err := proc.Wait()
	status = int64(msg.Sys().(syscall.WaitStatus).ExitStatus())

	SetCar(p.Scratch, NewStatus(status))

	return false
}

func function(body, param Cell, scope *Scope) *Applicative {
	return NewApplicative(NewClosure(body, param, scope), nil)
}

func head(p *Process) bool {
	switch Car(p.Scratch).(type) {
	case *Applicative:

		body := Car(p.Scratch).(*Applicative).Func.Body

		switch t := body.(type) {
		case Function:
			p.ReplaceState(psExecFunction)

		case *Integer:
			p.ReplaceState(t.Int())
			return true

		default:
			p.ReplaceState(psExecApplicative)
		}

		return false

	case *Operative:
		p.ReplaceState(psExecOperative)
		return true
	}

	return false
}

func lookup(p *Process, sym *Symbol) (bool, string) {
	c := Resolve(p.Lexical, p.Dynamic, sym)
	if c == nil {
		r := Raw(sym)
		if strict(p) && !number(r) {
			return false, r + " undefined"
		} else {
			p.Scratch = Cons(sym, p.Scratch)
		}
	} else if p.GetState() == psEvalElementBC && !IsSimple(c.GetValue()) {
		p.Scratch = Cons(sym, p.Scratch)
	} else {
		p.Scratch = Cons(c.GetValue(), p.Scratch)
	}

	return true, ""
}

func method(body, param Cell, scope *Scope) *Applicative {
	return NewApplicative(NewClosure(body, param, scope), scope)
}

func module(f string) (string, error) {
	i, err := os.Stat(f)
	if err != nil {
		return "", err
	}

	m := "$" + i.Name() + "-" + strconv.FormatInt(i.Size(), 10) + "-" +
		strconv.Itoa(i.ModTime().Second()) + "." +
		strconv.Itoa(i.ModTime().Nanosecond())

	return m, nil
}

func number(s string) bool {
	m, err := regexp.MatchString(`^[0-9]+(\.[0-9]+)?$`, s)
	return err == nil && m
}

func rpipe(c Cell) *os.File {
	r := Resolve(c.(Interface).Expose(), nil, NewSymbol("guts"))
	return r.GetValue().(*Channel).ReadFd()
}

func run(p *Process) (successful bool) {
	successful = true

	defer func(saved Process) {
		r := recover()
		if r == nil {
			return
		}

		successful = false

		fmt.Printf("oh: %v\n", r)

		*p = saved

		p.Code = Null
		p.Scratch = Cons(False, p.Scratch)
		p.RemoveState()
	}(*p)

	clearing: for interactive && p == proc0 {	
		select {
		case <-irq:
			continue clearing
		default:
			break clearing
		}
	}

	for p.Stack != Null {
		select {
		case <-irq:
                        if interactive {
				panic("interrupted")
			} else {
				proc0.Stack = Null
				return true
			}
		default:
			;
		}

		state := p.GetState()

		switch state {
		case psNone:
			return

		case psBlock:
			p.ReplaceState(SaveDynamic | SaveLexical)

			p.NewScope(p.Dynamic, p.Lexical)

			p.NewState(psEvalBlock)
			continue

		case psChangeScope:
			p.Dynamic = nil
			p.Lexical = Car(p.Scratch).(Interface)
			p.Scratch = Cdr(p.Scratch)

		case psEvalTopBlock:
			if p.Code == block0 {
				return
			}

			p.NewState(SaveCode, Cdr(p.Code))
			p.NewState(psEvalCommand)

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)
			continue

		case psExecApplicative:
			args := p.Arguments()

			m := Car(p.Scratch).(*Applicative)
			s := m.Self
			if s == nil {
				args = expand(args)
				s = p.Lexical.Expose()
			}

			p.ReplaceState(SaveDynamic | SaveLexical)

			p.Code = m.Func.Body
			p.NewScope(p.Dynamic, m.Func.Lexical)

			param := m.Func.Param
			for args != Null && param != Null {
				p.Lexical.Public(Car(param), Car(args))
				args, param = Cdr(args), Cdr(param)
			}
			p.Lexical.Public(NewSymbol("$args"), args)
			p.Lexical.Public(NewSymbol("$self"), s)
			p.Lexical.Public(NewSymbol("return"),
				p.Continuation(psReturn))

			p.NewState(psEvalBlock)
			continue

		case psExecFunction:
			args := p.Arguments()

			m := Car(p.Scratch).(*Applicative)
			if m.Self == nil {
				args = expand(args)
			}

			if m.Func.Body.(Function)(p, args) {
				continue
			}

		case psExecOperative:
			args := p.Code
			env := p.Lexical

			m := Car(p.Scratch).(*Operative)

			p.ReplaceState(SaveDynamic | SaveLexical)

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

			fallthrough
		case psEvalBlock:
			if p.Code == Null || !IsCons(p.Code) || !IsCons(Car(p.Code)) {
				break
			}

			if Cdr(p.Code) == Null || !IsCons(Cadr(p.Code)) {
				p.ReplaceState(psEvalCommand)
			} else {
				p.NewState(SaveCode, Cdr(p.Code))
				p.NewState(psEvalCommand)
			}

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			fallthrough
		case psEvalCommand:
			if p.Code == Null {
				p.Scratch = Cons(p.Code, p.Scratch)
				break
			}

			p.ReplaceState(psExecCommand)
			p.NewState(SaveCode, Cdr(p.Code))

			p.Code = Car(p.Code)

			p.NewState(psEvalElement)
			continue

		case psExecCommand:
			if Car(p.Scratch) == Null {
				break
			}

			switch Car(p.Scratch).(type) {
			case *String, *Symbol:
				c := NewClosure(Function(external), Null, p.Lexical.Expose())
				a := NewApplicative(c, nil)
				p.Scratch = Cons(a, p.Scratch)

				p.ReplaceState(psExecFunction)
				p.NewState(psEvalArgumentsBC)

			default:
				if head(p) {
					continue
				}

				state = p.GetState()
				if state == psExecApplicative &&
					Car(p.Scratch).(*Applicative).Self == nil {
					p.NewState(psEvalArgumentsBC)
				} else {
					p.NewState(psEvalArguments)
				}
			}

			p.Scratch = Cons(nil, p.Scratch)

			fallthrough
		case psEvalArguments, psEvalArgumentsBC:
			if p.Code == Null {
				break
			}

			state = next[p.GetState()]

			p.NewState(SaveCode, Cdr(p.Code))
			p.Code = Car(p.Code)

			p.NewState(state)

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
				ok, msg := lookup(p, sym)
				if !ok {
					panic(msg)
				}
				break
			} else {
				p.Scratch = Cons(p.Code, p.Scratch)
				break
			}

			fallthrough
		case psEvalAccess:
			p.ReplaceState(SaveDynamic | SaveLexical)

			p.NewState(psEvalElement)
			p.NewState(psChangeScope)
			p.NewState(SaveCode, Cdr(p.Code))

			p.Code = Car(p.Code)

			p.NewState(psEvalElement)
			continue

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

		case psDefine, psPublic:
			state = next[p.GetState()]

			p.RemoveState()

			l := Car(p.Scratch).(*Applicative).Self
			if p.Lexical != l {
				p.NewState(SaveLexical)
				p.Lexical = l
			}

			p.NewState(state)

			k := Car(p.Code)

			r := Raw(k)
			if strict(p) && number(r) {
				panic(r + " cannot be used as a variable name")
			}

			p.Code = Cadr(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(SaveCode|SaveLexical, k)
			p.NewState(psEvalElement)
			continue

		case psExecDefine, psExecPublic:
			if state == psExecDefine {
				p.Lexical.Define(p.Code, Car(p.Scratch))
			} else {
				p.Lexical.Public(p.Code, Car(p.Scratch))
			}

		case psDynamic, psSetenv:
			k := Car(p.Code)

			r := Raw(k)
			if strict(p) && number(r) {
				panic(r + " cannot be used as a variable name")
			}

			if state == psSetenv {
				if !strings.HasPrefix(r, "$") {
					break
				}
			}

			p.ReplaceState(next[state])

			p.Code = Cadr(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(SaveCode|SaveDynamic, k)
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

		case psIf:
			p.ReplaceState(SaveDynamic | SaveLexical)

			p.NewScope(p.Dynamic, p.Lexical)

			p.NewState(psExecIf)
			p.NewState(SaveCode, Cdr(p.Code))
			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(psEvalElement)
			continue

		case psExecIf:
			if !Car(p.Scratch).Bool() {
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

		case psWhile:
			p.ReplaceState(SaveDynamic | SaveLexical)

			p.NewState(psEvalWhileTest)

			fallthrough
		case psEvalWhileTest:
			p.ReplaceState(psEvalWhileBody)
			p.NewState(SaveCode, p.Code)

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(psEvalElement)
			continue

		case psEvalWhileBody:
			if !Car(p.Scratch).Bool() {
				break
			}

			p.ReplaceState(psEvalWhileTest)
			p.NewState(SaveCode, p.Code)

			p.Code = Cdr(p.Code)

			p.NewState(psEvalBlock)
			continue

		case psReturn:
			p.Code = Car(p.Code)

			m := Car(p.Scratch).(*Applicative)
			p.Scratch = Car(m.Func.Param)
			p.Stack = Cadr(m.Func.Param)

			p.NewState(psEvalElement)
			continue

		case psSet:
			p.Scratch = Cdr(p.Scratch)

			s := Car(p.Code)
			if !IsCons(s) {
				p.ReplaceState(psExecSet)
				p.NewState(SaveCode, s)
			} else {
				p.ReplaceState(SaveDynamic | SaveLexical)
				p.NewState(psExecSet)
				p.NewState(SaveCode, Cdr(s))
				p.NewState(psChangeScope)
				p.NewState(psEvalElement)
				p.NewState(SaveCode, Car(s))
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

		case psSpawn:
			child := NewProcess(psNone, p.Dynamic, p.Lexical)

			child.NewState(psEvalBlock)

			child.Code = p.Code
			child.Scratch = Cons(Null, child.Scratch)

			SetCar(p.Scratch, True)

			go run(child)

		case psSplice:
			p.ReplaceState(psExecSplice)

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			p.NewState(psEvalElement)
			continue

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

		default:
			if state >= SaveMax {
				panic(fmt.Sprintf("command not found: %s", p.Code))
			} else {
				p.RestoreState()
				continue
			}
		}

		p.RemoveState()
	}

	return
}

func strict(p *Process) (ok bool) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		ok = false
	}()

	c := Resolve(p.Lexical, nil, NewSymbol("strict"))
	if c == nil {
		return false
	}

	return c.GetValue().(Atom).Bool()
}

func syntax(body, param Cell, scope *Scope) *Operative {
	return NewOperative(NewClosure(body, param, scope), scope)
}

func wpipe(c Cell) *os.File {
	w := Resolve(c.(Interface).Expose(), nil, NewSymbol("guts"))
	return w.GetValue().(*Channel).WriteFd()
}

func Evaluate(c Cell) {
	saved := block0

	SetCar(block0, c)
	SetCdr(block0, Cons(nil, Null))
	block0 = Cdr(block0)

	proc0.NewState(SaveCode, block0)
	proc0.NewState(psEvalCommand)
	proc0.Code = c

	if !run(proc0) {
		block0 = saved
		SetCar(block0, nil)
		SetCdr(block0, Null)

		proc0.Code = block0
		proc0.RemoveState()
	} else {
		if proc0.Stack == Null {
			os.Exit(ExitStatus())
		}
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

func Start(i bool) {
	interactive = i

	irq = make(chan os.Signal, 1)
	signal.Notify(irq, syscall.SIGINT)

	proc0 = NewProcess(psEvalTopBlock, nil, nil)

	block0 = Cons(nil, Null)
	proc0.Code = block0

	proc0.Scratch = Cons(NewStatus(0), proc0.Scratch)

	e, s := proc0.Dynamic, proc0.Lexical.Expose()

	e.Define(NewSymbol("false"), False)
	e.Define(NewSymbol("true"), True)

	e.Define(NewSymbol("$stdin"), channel(proc0, os.Stdin, nil, -1))
	e.Define(NewSymbol("$stdout"), channel(proc0, nil, os.Stdout, -1))
	e.Define(NewSymbol("$stderr"), channel(proc0, nil, os.Stderr, -1))

	if wd, err := os.Getwd(); err == nil {
		e.Define(NewSymbol("$cwd"), NewSymbol(wd))
	}

	s.DefineState("block", psBlock)
	s.DefineState("builtin", psBuiltin)
	s.DefineState("define", psDefine)
	s.DefineState("dynamic", psDynamic)
	s.DefineState("if", psIf)
	s.DefineState("method", psMethod)
	s.DefineState("set", psSet)
	s.DefineState("setenv", psSetenv)
	s.DefineState("spawn", psSpawn)
	s.DefineState("splice", psSplice)
	s.DefineState("syntax", psSyntax)
	s.DefineState("while", psWhile)

	s.PublicState("public", psPublic)

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
	s.DefineFunction("module", func(p *Process, args Cell) bool {
		str, err := module(Raw(Car(args)))

		if err != nil {
			SetCar(p.Scratch, Null)

			return false
		}

		sym := NewSymbol(str)
		c := Resolve(p.Lexical, p.Dynamic, sym)

		if c == nil {
			SetCar(p.Scratch, sym)

			return false
		}

		SetCar(p.Scratch, c.GetValue())

		return false
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
	s.DefineMethod("apply", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Car(args))
		head(p)

		p.Scratch = Cons(nil, p.Scratch)
		for args = Cdr(args); args != Null; args = Cdr(args) {
			p.Scratch = Cons(Car(args), p.Scratch)
		}

		return true
	})
	s.DefineMethod("car", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Caar(args))

		return false
	})
	s.DefineMethod("cdr", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cdar(args))

		return false
	})
	s.DefineMethod("cons", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, Cons(Car(args), Cadr(args)))

		return false
	})
	s.DefineMethod("eval", func(p *Process, args Cell) bool {
		if Cdr(args) != Null {
			p.ReplaceState(SaveDynamic | SaveLexical)

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
	s.DefineMethod("list", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, args)

		return false
	})
	s.DefineMethod("open", func(p *Process, args Cell) bool {
		name := Raw(Car(args))
		mode := Raw(Cadr(args))

		flags := os.O_CREATE

		read := false
		if strings.IndexAny(mode, "r") != -1 {
			read = true
		}

		write := false
		if strings.IndexAny(mode, "w") != -1 {
			write = true
			if strings.IndexAny(mode, "a") == -1 {
				flags |= os.O_TRUNC
			}
		}

		if strings.IndexAny(mode, "a") != -1 {
			write = true
			flags |= os.O_APPEND
		}

		if read == write {
			read = true
			write = true
			flags |= os.O_RDWR
		} else if write {
			flags |= os.O_WRONLY
		}

		f, err := os.OpenFile(name, flags, 0666)
		if err != nil {
			panic(err)
		}

		r := f
		if !read {
			r = nil
		}

		w := f
		if !write {
			w = nil
		}

		SetCar(p.Scratch, channel(p, r, w, -1))

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
		c := 0
		if args != Null {
			c = int(Car(args).(Atom).Int())
		}
		SetCar(p.Scratch, channel(p, nil, nil, c))

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
	s.DefineMethod("pipe", func(p *Process, args Cell) bool {
		SetCar(p.Scratch, channel(p, nil, nil, -1))

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

	/* Standard namespaces. */
	list := NewObject(NewLexicalScope(s))
	s.Define(NewSymbol("$list"), list)

	list.PublicMethod("to-string", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		SetCar(p.Scratch, NewString(s))

		return false
	})
	list.PublicMethod("to-symbol", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		SetCar(p.Scratch, NewSymbol(s))

		return false
	})

	text := NewObject(NewLexicalScope(s))
	s.Define(NewSymbol("$text"), text)

	text.PublicMethod("is-control", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsControl(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-digit", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsDigit(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-graphic", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsGraphic(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-letter", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLetter(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-lower", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLower(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-mark", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsMark(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-print", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPrint(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-punct", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPunct(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-space", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSpace(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-symbol", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSymbol(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-title", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsTitle(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("is-upper", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsUpper(rune(t.Int())))
		default:
			r = Null
		}

		SetCar(p.Scratch, r)

		return false
	})
	text.PublicMethod("sprintf", func(p *Process, args Cell) bool {
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
	text.PublicMethod("substring", func(p *Process, args Cell) bool {
		var r Cell

		s := []rune(Raw(Car(args)))

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
	text.PublicMethod("to-list", func(p *Process, args Cell) bool {
		l := Null
		for _, char := range Raw(Car(args)) {
			l = Cons(NewInteger(int64(char)), l)
		}

		SetCar(p.Scratch, Reverse(l))

		return false
	})
	text.PublicMethod("to-lower", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToLower(rune(t.Int()))))
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
	text.PublicMethod("to-title", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToTitle(rune(t.Int()))))
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
	text.PublicMethod("to-upper", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToUpper(rune(t.Int()))))
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

	s.Public(NewSymbol("$root"), s)

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
define caar: method l: car: car l
define cadr: method l: car: cdr l
define cdar: method l: cdr: car l
define cddr: method l: cdr: cdr l
define caaar: method l: car: caar l
define caadr: method l: car: cadr l
define cadar: method l: car: cdar l
define caddr: method l: car: cddr l
define cdaar: method l: cdr: caar l
define cdadr: method l: cdr: cadr l
define cddar: method l: cdr: cdar l
define cdddr: method l: cdr: cddr l
define caaaar: method l: caar: caar l
define caaadr: method l: caar: cadr l
define caadar: method l: caar: cdar l
define caaddr: method l: caar: cddr l
define cadaar: method l: cadr: caar l
define cadadr: method l: cadr: cadr l
define caddar: method l: cadr: cdar l
define cadddr: method l: cadr: cddr l
define cdaaar: method l: cdar: caar l
define cdaadr: method l: cdar: cadr l
define cdadar: method l: cdar: cdar l
define cdaddr: method l: cdar: cddr l
define cddaar: method l: cddr: caar l
define cddadr: method l: cddr: cadr l
define cdddar: method l: cddr: cdar l
define cddddr: method l: cddr: cddr l
define $connect: syntax {
    define type: eval: car $args
    define out: cadr $args
    define close: eval: caddr $args
    syntax e {
        define p: type
        define left: car $args
        define right: cadr $args
        spawn {
            eval: list 'dynamic out 'p
            eval e left
            if close: p::writer-close
        }

        dynamic $stdin p
        eval e right
        if close: p::reader-close
    }
}
define $redirect: syntax {
    define chan: car $args
    define mode: cadr $args
    define mthd: caddr $args
    syntax e {
        define c: eval e: car $args
        define f '()
        if (not: is-channel c) {
            set f: open c mode
            set c f
        }
        eval: list 'dynamic chan 'c
        eval e: cadr $args
        if (not: is-null f): eval: cons 'f mthd
    }
}
define and: syntax e {
    define r false
    while (not: is-null: car $args) {
        set r: eval e: car $args
        if (not r): return r
        set $args: cdr $args
    }
    return r
}
define append-stderr: $redirect $stderr "a" writer-close
define append-stdout: $redirect $stdout "a" writer-close
define backtick: syntax e {
    define p: pipe
    define r '()
    spawn {
        dynamic $stdout p
        eval e: car $args
        p::writer-close
    }
    define l: p::readline
    while l {
        set r: append r l
        set l: p::readline
    }
    p::reader-close
    return r
}
define channel-stderr: $connect channel $stderr true
define channel-stdout: $connect channel $stdout true
define echo: builtin: $stdout::write @$args
define for: method l m {
    define r: cons '() '()
    define c r
    while (not: is-null l) {
        set-cdr c: cons (m: car l) '()
        set c: cdr c
        set l: cdr l
    }
    return: cdr r
}
define glob: builtin: return $args
define import: syntax e {
    define m: module: car $args
    if (or (is-null m) (is-object m)) {
        return m
    }

    define l: list 'source: car $args
    set l: cons 'object: cons l '()
    set l: list '$root::define m l
    eval e l
}
$list::public ref: method k x: car: $list::tail k x
$list::public tail: method k x {
    if k {
        $list::tail (sub k 1): cdr x
    } else {
        return x
    }
}
define object: syntax e {
    eval e: cons 'block: append $args '(clone)
}
define or: syntax e {
    define r false
    while (not: is-null: car $args) {
	set r: eval e: car $args
        if r: return r
        set $args: cdr $args
    }
    return r
}
define pipe-stderr: $connect pipe $stderr true
define pipe-stdout: $connect pipe $stdout true
define printf: method: echo: $text::sprintf (car $args) @(cdr $args)
define quote: syntax: car $args
define read: builtin: $stdin::read
define readline: builtin: $stdin::readline
define redirect-stderr: $redirect $stderr "w" writer-close
define redirect-stdin: $redirect $stdin "r" reader-close
define redirect-stdout: $redirect $stdout "w" writer-close
define source: syntax e {
        define f: open (eval e: car $args) "r"
        define l: f::read
        while l {
                eval e l
                set l: f::read
        }
        f::reader-close
}
define write: method: $stdout::write @$args
[ -r (add $HOME /.ohrc) ] && source (add $HOME /.ohrc)
`)), Evaluate)
}
