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

	psChangeContext = SaveMax + iota

	psEvalArguments
	psEvalArgumentsBuiltin
	psEvalBlock
	psEvalCommand
	psEvalElement
	psEvalElementBuiltin

	psExecBuiltin
	psExecCommand
	psExecDefine
	psExecDynamic
	psExecIf
	psExecMethod
	psExecPublic
	psExecSet
	psExecSetenv
	psExecSplice
	psExecSyntax
	psExecWhileBody
	psExecWhileTest

	psMax
)

var block0 Cell
var proc0 *Process

var con Cell
var ext Cell
var interactive bool
var incoming chan os.Signal
var next = map[int64][]int64{
	psEvalArguments:        {SaveCdrCode, psEvalElement},
	psEvalArgumentsBuiltin: {SaveCdrCode, psEvalElementBuiltin},
	psExecIf:               {psEvalBlock},
	psExecWhileBody:        {psExecWhileTest, SaveCode, psEvalBlock},
}

func apply(p *Process, args Cell) bool {
	m := Car(p.Scratch).(Binding)

	p.ReplaceStates(SaveDynamic|SaveLexical, psEvalBlock)

	p.Code = m.Ref().Code()
	p.NewScope(p.Dynamic, m.Ref().Lexical())

	label := m.Ref().Label()
	if label != Null {
		p.Lexical.Public(label, m.Self())
	}

	formal := m.Ref().Formal()
	for args != Null && formal != Null && IsAtom(Car(formal)) {
		p.Lexical.Public(Car(formal), Car(args))
		args, formal = Cdr(args), Cdr(formal)
	}
	if IsCons(Car(formal)) {
		p.Lexical.Public(Caar(formal), args)
	}

	con := NewUnbound(NewMethod(tinue, Cdr(p.Scratch), p.Stack, Null, nil))
	p.Lexical.Public(NewSymbol("return"), con)

	return true
}

func channel(p *Process, r, w *os.File, cap int) Context {
	c, ch := NewLexicalScope(p.Lexical), NewChannel(r, w, cap)

	var rclose Function = func(p *Process, args Cell) bool {
		return p.Return(NewBoolean(ch.ReaderClose()))
	}

	var read Function = func(p *Process, args Cell) bool {
		return p.Return(ch.Read())
	}

	var readline Function = func(p *Process, args Cell) bool {
		return p.Return(ch.ReadLine())
	}

	var wclose Function = func(p *Process, args Cell) bool {
		return p.Return(NewBoolean(ch.WriterClose()))
	}

	var write Function = func(p *Process, args Cell) bool {
		ch.Write(args)
		return p.Return(True)
	}

	c.Public(NewSymbol("guts"), ch)
	c.Public(NewSymbol("reader-close"), method(rclose, c))
	c.Public(NewSymbol("read"), method(read, c))
	c.Public(NewSymbol("readline"), method(readline, c))
	c.Public(NewSymbol("writer-close"), method(wclose, c))
	c.Public(NewSymbol("write"), method(write, c))

	return NewObject(c)
}

func combiner(p *Process, n NewClosure) bool {
	label := Null
	formal := Car(p.Code)
	for p.Code != Null && Raw(Cadr(p.Code)) != "as" {
		label = formal
		formal = Cadr(p.Code)
		p.Code = Cdr(p.Code)
	}

	if p.Code == Null {
		panic("expected 'as'")
	}

	block := Cddr(p.Code)
	scope := p.Lexical.Expose()

	c := n(apply, block, formal, label, scope)
	if label == Null {
		SetCar(p.Scratch, NewUnbound(c))
	} else {
		SetCar(p.Scratch, NewBound(c, scope))
	}

	return false
}

func debug(p *Process, s string) {
	fmt.Printf("%s: p.Code = %v, p.Scratch = %v\n", s, p.Code, p.Scratch)
}

func dynamic(p *Process, state int64) bool {
	r := Raw(Car(p.Code))
	if strict(p) && number(r) {
		panic(r + " cannot be used as a variable name")
	}

	if state == psExecSetenv {
		if !strings.HasPrefix(r, "$") {
			/* TODO: We should probably panic here. */
			return false
		}
	}

	p.ReplaceStates(state, SaveCarCode|SaveDynamic, psEvalElement)

	p.Code = Cadr(p.Code)
	p.Scratch = Cdr(p.Scratch)

	return true
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

				for _, v := range m {
					if v[0] != '.' || s[0] == '.' {
						e := NewSymbol(v)
						list = AppendTo(list, e)
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
	dir := c.Get().String()

	stdin := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdin")).Get()
	stdout := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdout")).Get()
	stderr := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stderr")).Get()

	fd := []*os.File{rpipe(stdin), wpipe(stdout), wpipe(stderr)}

	attr := &os.ProcAttr{Dir: dir, Env: nil, Files: fd}
	proc, err := os.StartProcess(name, argv, attr)
	if err != nil {
		panic(err)
	}

	var status int64 = 0

	msg, err := proc.Wait()
	status = int64(msg.Sys().(syscall.WaitStatus).ExitStatus())

	return p.Return(NewStatus(status))
}

func lookup(p *Process, sym *Symbol, simple bool) (bool, string) {
	c := Resolve(p.Lexical, p.Dynamic, sym)
	if c == nil {
		r := Raw(sym)
		if strict(p) && !number(r) {
			return false, r + " undefined"
		} else {
			p.Scratch = Cons(sym, p.Scratch)
		}
	} else if simple && !IsSimple(c.Get()) {
		p.Scratch = Cons(sym, p.Scratch)
	} else if a, ok := c.Get().(Binding); ok {
		p.Scratch = Cons(a.Bind(p.Lexical.Expose()), p.Scratch)
	} else {
		p.Scratch = Cons(c.Get(), p.Scratch)
	}

	return true, ""
}

func lexical(p *Process, state int64) bool {
	p.RemoveState()

	l := Car(p.Scratch).(Binding).Self()
	if p.Lexical != l {
		p.NewStates(SaveLexical)
		p.Lexical = l
	}

	p.NewStates(state)

	r := Raw(Car(p.Code))
	if strict(p) && number(r) {
		panic(r + " cannot be used as a variable name")
	}

	p.NewStates(SaveCarCode|SaveLexical, psEvalElement)

	p.Code = Cadr(p.Code)
	p.Scratch = Cdr(p.Scratch)

	return true
}

func method(body Function, scope *Scope) Binding {
	return NewBound(NewMethod(body, Null, Null, Null, scope), scope)
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
	r := Resolve(c.(Context).Expose(), nil, NewSymbol("guts"))
	return r.Get().(*Channel).ReadFd()
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

	p.ClearSignals()
	for p.Stack != Null {
		if p.HandleSignal(func(p *Process, args Cell) bool {
			if interactive {
				panic("interrupted")
			}

			proc0.Stack = Null
			return true
		}) {
			continue
		}

		state := p.GetState()

		switch state {
		case psNone:
			return

		case psChangeContext:
			p.Dynamic = nil
			p.Lexical = Car(p.Scratch).(Context)
			p.Scratch = Cdr(p.Scratch)

		case psExecBuiltin, psExecMethod:
			args := p.Arguments()

			if state == psExecBuiltin {
				args = expand(args)
			}

			p.Code = args

			fallthrough
		case psExecSyntax:
			m := Car(p.Scratch).(Binding)

			if m.Ref().Body()(p, p.Code) {
				continue
			}

		case psExecIf, psExecWhileBody:
			if !Car(p.Scratch).Bool() {
				p.Code = Cdr(p.Code)

				for Car(p.Code) != Null &&
					!IsAtom(Car(p.Code)) {
					p.Code = Cdr(p.Code)
				}

				if Car(p.Code) != Null &&
					Raw(Car(p.Code)) != "else" {
					panic("expected 'else'")
				}
			}

			if Cdr(p.Code) == Null {
				break
			}

			p.ReplaceStates(next[p.GetState()]...)

			p.Code = Cdr(p.Code)

			fallthrough
		case psEvalBlock:
			if p.Code == block0 {
				return
			}

			if p.Code == Null ||
				!IsCons(p.Code) || !IsCons(Car(p.Code)) {
				break
			}

			if Cdr(p.Code) == Null || !IsCons(Cadr(p.Code)) {
				p.ReplaceStates(psEvalCommand)
			} else {
				p.NewStates(SaveCdrCode, psEvalCommand)
			}

			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			fallthrough
		case psEvalCommand:
			if p.Code == Null {
				p.Scratch = Cons(p.Code, p.Scratch)
				break
			}

			p.ReplaceStates(psExecCommand,
				SaveCdrCode,
				psEvalElement)
			p.Code = Car(p.Code)

			continue

		case psExecCommand:
			switch t := Car(p.Scratch).(type) {
			case *String, *Symbol:
				p.Scratch = Cons(ext, p.Scratch)

				p.ReplaceStates(psExecBuiltin,
					psEvalArgumentsBuiltin)
			case Binding:
				switch t.Ref().(type) {
				case *Builtin:
					p.ReplaceStates(psExecBuiltin,
						psEvalArgumentsBuiltin)

				case *Method:
					p.ReplaceStates(psExecMethod,
						psEvalArguments)
				case *Syntax:
					p.ReplaceStates(psExecSyntax)
					continue
				}

			default:
				panic(fmt.Sprintf("can't evaluate: %v", t))
			}

			p.Scratch = Cons(nil, p.Scratch)

			fallthrough
		case psEvalArguments, psEvalArgumentsBuiltin:
			if p.Code == Null {
				break
			}

			p.NewStates(next[p.GetState()]...)

			p.Code = Car(p.Code)

			fallthrough
		case psEvalElement, psEvalElementBuiltin:
			if p.Code == Null {
				p.Scratch = Cons(p.Code, p.Scratch)
				break
			} else if IsCons(p.Code) {
				if IsAtom(Cdr(p.Code)) {
					p.ReplaceStates(SaveDynamic|SaveLexical,
						psEvalElement,
						psChangeContext,
						SaveCdrCode,
						psEvalElement)
					p.Code = Car(p.Code)
				} else {
					p.ReplaceStates(psEvalCommand)
				}
				continue
			} else if sym, ok := p.Code.(*Symbol); ok {
				simple := p.GetState() == psEvalElementBuiltin
				ok, msg := lookup(p, sym, simple)
				if !ok {
					panic(msg)
				}
				break
			} else {
				p.Scratch = Cons(p.Code, p.Scratch)
				break
			}

		case psExecDefine:
			p.Lexical.Define(p.Code, Car(p.Scratch))

		case psExecPublic:
			p.Lexical.Public(p.Code, Car(p.Scratch))

		case psExecDynamic, psExecSetenv:
			k := p.Code
			v := Car(p.Scratch)

			if state == psExecSetenv {
				s := Raw(v)
				os.Setenv(strings.TrimLeft(k.String(), "$"), s)
			}

			p.Dynamic.Define(k, v)

		case psExecSet:
			k := p.Code.(*Symbol)
			r := Resolve(p.Lexical, p.Dynamic, k)
			if r == nil {
				panic("'" + k.String() + "' is not defined")
			}

			r.Set(Car(p.Scratch))

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

		case psExecWhileTest:
			p.ReplaceStates(psExecWhileBody,
				SaveCode,
				psEvalElement)
			p.Code = Car(p.Code)
			p.Scratch = Cdr(p.Scratch)

			continue

		default:
			if state >= SaveMax {
				panic(fmt.Sprintf("command not found: %s",
					p.Code))
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

	return c.Get().(Atom).Bool()
}

func tinue(p *Process, args Cell) bool {
	p.Code = Car(p.Code)

	m := Car(p.Scratch).(Binding)
	p.Scratch = m.Ref().Code()
	p.Stack = m.Ref().Formal()

	p.Scratch = Cons(Car(args), p.Scratch)

	return false
}

func wpipe(c Cell) *os.File {
	w := Resolve(c.(Context).Expose(), nil, NewSymbol("guts"))
	return w.Get().(*Channel).WriteFd()
}

func Evaluate(c Cell) {
	saved := block0

	proc0.Code = block0
	SetCar(block0, c)
	SetCdr(block0, Cons(nil, Null))
	block0 = Cdr(block0)

	proc0.NewStates(SaveCdrCode, psEvalCommand)
	proc0.Code = Car(proc0.Code)

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

	ext = NewUnbound(NewBuiltin(Function(external), Null, Null, Null, nil))

	incoming = make(chan os.Signal, 1)
	signal.Notify(incoming, syscall.SIGINT, syscall.SIGTSTP)
	go func() {
		for sig := range incoming {
			switch sig {
			case syscall.SIGINT:
				proc0.Interrupt()
			default:
				// SIGTSTP - Do nothing.
			}
		}
	}()

	proc0 = NewProcess(psEvalBlock, nil, nil)

	block0 = Cons(nil, Null)
	proc0.Code = block0

	proc0.Scratch = Cons(NewStatus(0), proc0.Scratch)

	e, s := proc0.Dynamic, proc0.Lexical.Expose()

	e.Define(NewSymbol("False"), False)
	e.Define(NewSymbol("True"), True)

	e.Define(NewSymbol("$stdin"), channel(proc0, os.Stdin, nil, -1))
	e.Define(NewSymbol("$stdout"), channel(proc0, nil, os.Stdout, -1))
	e.Define(NewSymbol("$stderr"), channel(proc0, nil, os.Stderr, -1))

	if wd, err := os.Getwd(); err == nil {
		e.Define(NewSymbol("$cwd"), NewSymbol(wd))
	}

	s.DefineSyntax("block", func(p *Process, args Cell) bool {
		p.ReplaceStates(SaveDynamic|SaveLexical, psEvalBlock)

		p.NewScope(p.Dynamic, p.Lexical)

		return true
	})
	s.DefineSyntax("builtin", func(p *Process, args Cell) bool {
		return combiner(p, NewBuiltin)
	})
	s.DefineSyntax("define", func(p *Process, args Cell) bool {
		return lexical(p, psExecDefine)
	})
	s.DefineSyntax("dynamic", func(p *Process, args Cell) bool {
		return dynamic(p, psExecDynamic)
	})
	s.DefineSyntax("if", func(p *Process, args Cell) bool {
		p.ReplaceStates(SaveDynamic|SaveLexical,
			psExecIf, SaveCode, psEvalElement)

		p.NewScope(p.Dynamic, p.Lexical)

		p.Code = Car(p.Code)
		p.Scratch = Cdr(p.Scratch)

		return true
	})
	s.DefineSyntax("method", func(p *Process, args Cell) bool {
		return combiner(p, NewMethod)
	})
	s.DefineSyntax("set", func(p *Process, args Cell) bool {
		p.Scratch = Cdr(p.Scratch)

		s := Cadr(p.Code)
		p.Code = Car(p.Code)
		if !IsCons(p.Code) {
			p.ReplaceStates(psExecSet, SaveCode)
		} else {
			p.ReplaceStates(SaveDynamic|SaveLexical,
				psExecSet, SaveCdrCode,
				psChangeContext, psEvalElement,
				SaveCarCode)
		}

		p.NewStates(psEvalElement)

		p.Code = s
		return true
	})
	s.DefineSyntax("setenv", func(p *Process, args Cell) bool {
		return dynamic(p, psExecSetenv)
	})
	s.DefineSyntax("spawn", func(p *Process, args Cell) bool {
		child := NewProcess(psNone, p.Dynamic, p.Lexical)

		child.NewStates(psEvalBlock)

		child.Code = p.Code
		child.Scratch = Cons(Null, child.Scratch)

		SetCar(p.Scratch, child)

		go run(child)

		return false
	})
	s.DefineSyntax("splice", func(p *Process, args Cell) bool {
		p.ReplaceStates(psExecSplice, psEvalElement)

		p.Code = Car(p.Code)
		p.Scratch = Cdr(p.Scratch)

		return true
	})
	s.DefineSyntax("syntax", func(p *Process, args Cell) bool {
		return combiner(p, NewSyntax)
	})

	s.PublicSyntax("public", func(p *Process, args Cell) bool {
		return lexical(p, psExecPublic)
	})
	s.PublicSyntax("while", func(p *Process, args Cell) bool {
		p.ReplaceStates(SaveDynamic|SaveLexical, psExecWhileTest)

		return true
	})

	/* Builtins. */
	s.DefineBuiltin("cd", func(p *Process, args Cell) bool {
		err := os.Chdir(Raw(Car(args)))
		status := 0
		if err != nil {
			status = 1
		}

		if wd, err := os.Getwd(); err == nil {
			p.Dynamic.Define(NewSymbol("$cwd"), NewSymbol(wd))
		}

		return p.Return(NewStatus(int64(status)))
	})
	s.DefineBuiltin("debug", func(p *Process, args Cell) bool {
		debug(p, "debug")

		return false
	})
	s.DefineBuiltin("exit", func(p *Process, args Cell) bool {
		var status int64 = 0

		a, ok := Car(args).(Atom)
		if ok {
			status = a.Status()
		}

		p.Scratch = List(NewStatus(status))
		p.Stack = Null

		return true
	})
	s.DefineBuiltin("module", func(p *Process, args Cell) bool {
		str, err := module(Raw(Car(args)))

		if err != nil {
			return p.Return(Null)
		}

		sym := NewSymbol(str)
		c := Resolve(p.Lexical, p.Dynamic, sym)

		if c == nil {
			return p.Return(sym)
		}

		return p.Return(c.Get())
	})

	s.PublicMethod("child", func(p *Process, args Cell) bool {
		o := Car(p.Scratch).(Binding).Self()

		return p.Return(NewObject(NewLexicalScope(o)))
	})
	s.PublicMethod("clone", func(p *Process, args Cell) bool {
		o := Car(p.Scratch).(Binding).Self()

		return p.Return(NewObject(o.Copy()))
	})
	s.PublicMethod("exists", func(p *Process, args Cell) bool {
		l := Car(p.Scratch).(Binding).Self()
		c := Resolve(l, p.Dynamic, NewSymbol(Raw(Car(args))))

		return p.Return(NewBoolean(c != nil))
	})
	s.PublicMethod("unset", func(p *Process, args Cell) bool {
		l := Car(p.Scratch).(Binding).Self()
		r := l.Remove(NewSymbol(Raw(Car(args))))

		return p.Return(NewBoolean(r))
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

		return p.Return(Append(l, argv...))
	})
	s.DefineMethod("car", func(p *Process, args Cell) bool {
		return p.Return(Caar(args))
	})
	s.DefineMethod("cdr", func(p *Process, args Cell) bool {
		return p.Return(Cdar(args))
	})
	s.DefineMethod("cons", func(p *Process, args Cell) bool {
		return p.Return(Cons(Car(args), Cadr(args)))
	})
	s.PublicMethod("eval", func(p *Process, args Cell) bool {
		scope := Car(p.Scratch).(Binding).Self()
		p.RemoveState()
		if p.Lexical != scope {
			p.NewStates(SaveLexical)
			p.Lexical = scope
		}
		p.NewStates(psEvalElement)
		p.Code = Car(args)
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

		return p.Return(NewInteger(l))
	})
	s.DefineMethod("list", func(p *Process, args Cell) bool {
		return p.Return(args)
	})
	s.DefineMethod("open", func(p *Process, args Cell) bool {
		name := Raw(Car(args))
		mode := Raw(Cadr(args))
		flags := 0

		if strings.IndexAny(mode, "-") == -1 {
			flags = os.O_CREATE
		}

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

		return p.Return(channel(p, r, w, -1))
	})
	s.DefineMethod("reverse", func(p *Process, args Cell) bool {
		return p.Return(Reverse(Car(args)))
	})
	s.DefineMethod("set-car", func(p *Process, args Cell) bool {
		SetCar(Car(args), Cadr(args))

		return p.Return(Cadr(args))
	})
	s.DefineMethod("set-cdr", func(p *Process, args Cell) bool {
		SetCdr(Car(args), Cadr(args))

		return p.Return(Cadr(args))
	})

	/* Predicates. */
	s.DefineMethod("is-atom", func(p *Process, args Cell) bool {
		return p.Return(NewBoolean(IsAtom(Car(args))))
	})
	s.DefineMethod("is-boolean", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Boolean)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-builtin", func(p *Process, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Builtin)
		}

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-channel", func(p *Process, args Cell) bool {
		o, ok := Car(args).(Context)
		if !ok {
			return p.Return(False)
		}

		g := Resolve(o.Expose(), nil, NewSymbol("guts"))
		if g == nil {
			return p.Return(False)
		}

		c, ok := g.Get().(*Channel)
		if !ok {
			return p.Return(False)
		}

		ok = (c.ReadFd() == nil && c.WriteFd() == nil)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-cons", func(p *Process, args Cell) bool {
		return p.Return(NewBoolean(IsCons(Car(args))))
	})
	s.DefineMethod("is-float", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Float)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-integer", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Integer)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-list", func(p *Process, args Cell) bool {
		return p.Return(NewBoolean(IsList(Car(args))))
	})
	s.DefineMethod("is-method", func(p *Process, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Method)
		}

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-null", func(p *Process, args Cell) bool {
		ok := Car(args) == Null

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-number", func(p *Process, args Cell) bool {
		_, ok := Car(args).(Number)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-object", func(p *Process, args Cell) bool {
		_, ok := Car(args).(Context)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-pipe", func(p *Process, args Cell) bool {
		o, ok := Car(args).(Context)
		if !ok {
			return p.Return(False)
		}

		g := Resolve(o.Expose(), nil, NewSymbol("guts"))
		if g == nil {
			return p.Return(False)
		}

		c, ok := g.Get().(*Channel)
		if !ok {
			return p.Return(False)
		}

		ok = (c.ReadFd() != nil || c.WriteFd() != nil)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-status", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Status)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-string", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*String)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-symbol", func(p *Process, args Cell) bool {
		_, ok := Car(args).(*Symbol)

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-syntax", func(p *Process, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Syntax)
		}

		return p.Return(NewBoolean(ok))
	})

	/* Generators. */
	s.DefineMethod("boolean", func(p *Process, args Cell) bool {
		return p.Return(NewBoolean(Car(args).Bool()))
	})
	s.DefineMethod("channel", func(p *Process, args Cell) bool {
		c := 0
		if args != Null {
			c = int(Car(args).(Atom).Int())
		}

		return p.Return(channel(p, nil, nil, c))
	})
	s.DefineMethod("float", func(p *Process, args Cell) bool {
		return p.Return(NewFloat(Car(args).(Atom).Float()))
	})
	s.DefineMethod("integer", func(p *Process, args Cell) bool {
		return p.Return(NewInteger(Car(args).(Atom).Int()))
	})
	s.DefineMethod("pipe", func(p *Process, args Cell) bool {
		return p.Return(channel(p, nil, nil, -1))
	})
	s.DefineMethod("status", func(p *Process, args Cell) bool {
		return p.Return(NewStatus(Car(args).(Atom).Status()))
	})
	s.DefineMethod("string", func(p *Process, args Cell) bool {
		return p.Return(NewString(Car(args).String()))
	})
	s.DefineMethod("symbol", func(p *Process, args Cell) bool {
		return p.Return(NewSymbol(Raw(Car(args))))
	})

	/* Relational. */
	s.DefineMethod("eq", func(p *Process, args Cell) bool {
		prev := Car(args)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args)

			if !prev.Equal(curr) {
				return p.Return(False)
			}

			prev = curr
		}

		return p.Return(True)
	})
	s.DefineMethod("ge", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if prev.Less(curr) {
				return p.Return(False)
			}

			prev = curr
		}

		return p.Return(True)
	})
	s.DefineMethod("gt", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if !prev.Greater(curr) {
				return p.Return(False)
			}

			prev = curr
		}

		return p.Return(True)
	})
	s.DefineMethod("is", func(p *Process, args Cell) bool {
		prev := Car(args)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args)

			if prev != curr {
				return p.Return(False)
			}

			prev = curr
		}

		return p.Return(True)
	})
	s.DefineMethod("le", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if prev.Greater(curr) {
				return p.Return(False)
			}

			prev = curr
		}

		return p.Return(True)
	})
	s.DefineMethod("lt", func(p *Process, args Cell) bool {
		prev := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Atom)

			if !prev.Less(curr) {
				return p.Return(False)
			}

			prev = curr
		}

		return p.Return(True)
	})
	s.DefineMethod("match", func(p *Process, args Cell) bool {
		pattern := Raw(Car(args))
		text := Raw(Cadr(args))

		ok, err := path.Match(pattern, text)
		if err != nil {
			panic(err)
		}

		return p.Return(NewBoolean(ok))
	})
	s.DefineMethod("ne", func(p *Process, args Cell) bool {
		for l1 := args; l1 != Null; l1 = Cdr(l1) {
			for l2 := Cdr(l1); l2 != Null; l2 = Cdr(l2) {
				v1 := Car(l1)
				v2 := Car(l2)

				if v1.Equal(v2) {
					return p.Return(False)
				}
			}
		}

		return p.Return(True)
	})
	s.DefineMethod("not", func(p *Process, args Cell) bool {
		return p.Return(NewBoolean(!Car(args).Bool()))
	})

	/* Arithmetic. */
	s.DefineMethod("add", func(p *Process, args Cell) bool {
		acc := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Add(Car(args))

		}

		return p.Return(acc)
	})
	s.DefineMethod("sub", func(p *Process, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Subtract(Car(args))
		}

		return p.Return(acc)
	})
	s.DefineMethod("div", func(p *Process, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Divide(Car(args))
		}

		return p.Return(acc)
	})
	s.DefineMethod("mod", func(p *Process, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Modulo(Car(args))
		}

		return p.Return(acc)
	})
	s.DefineMethod("mul", func(p *Process, args Cell) bool {
		acc := Car(args).(Atom)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Multiply(Car(args))
		}

		return p.Return(acc)
	})

	/* Standard namespaces. */
	list := NewObject(NewLexicalScope(s))
	s.Define(NewSymbol("List"), list)

	list.PublicMethod("to-string", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return p.Return(NewString(s))
	})
	list.PublicMethod("to-symbol", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return p.Return(NewSymbol(s))
	})

	text := NewObject(NewLexicalScope(s))
	s.Define(NewSymbol("Text"), text)

	text.PublicMethod("is-control", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsControl(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-digit", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsDigit(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-graphic", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsGraphic(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-letter", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLetter(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-lower", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLower(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-mark", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsMark(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-print", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPrint(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-punct", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPunct(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-space", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSpace(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-symbol", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSymbol(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-title", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsTitle(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("is-upper", func(p *Process, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsUpper(rune(t.Int())))
		default:
			r = Null
		}

		return p.Return(r)
	})
	text.PublicMethod("join", func(p *Process, args Cell) bool {
		str := false
		sep := Car(args)
		list := Cdr(args)

		arr := make([]string, Length(list))

		for i := 0; list != Null; i++ {
			_, str = Car(list).(*String)
			arr[i] = string(Raw(Car(list)))
			list = Cdr(list)
		}

		r := strings.Join(arr, string(Raw(sep)))

		if str {
			return p.Return(NewString(r))
		}
		return p.Return(NewSymbol(r))
	})
	text.PublicMethod("split", func(p *Process, args Cell) bool {
		var r Cell = Null

		sep := Car(args)
		str := Cadr(args)

		l := strings.Split(string(Raw(str)), string(Raw(sep)))

		for i := len(l) - 1; i >= 0; i-- {
			switch str.(type) {
			case *Symbol:
				r = Cons(NewSymbol(l[i]), r)
			case *String:
				r = Cons(NewString(l[i]), r)
			}
		}

		return p.Return(r)
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

		return p.Return(NewString(s))
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

		return p.Return(r)
	})
	text.PublicMethod("to-list", func(p *Process, args Cell) bool {
		l := Null
		for _, char := range Raw(Car(args)) {
			l = Cons(NewInteger(int64(char)), l)
		}

		return p.Return(Reverse(l))
	})
	text.PublicMethod("lower", func(p *Process, args Cell) bool {
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

		return p.Return(r)
	})
	text.PublicMethod("title", func(p *Process, args Cell) bool {
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

		return p.Return(r)
	})
	text.PublicMethod("upper", func(p *Process, args Cell) bool {
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

		return p.Return(r)
	})

	s.Public(NewSymbol("Root"), s)

	e.Define(NewSymbol("$$"), NewInteger(int64(os.Getpid())))

	/* Command-line arguments */
	args := Null
	if len(os.Args) > 1 {
		e.Define(NewSymbol("$0"), NewSymbol(os.Args[1]))

		for i, v := range os.Args[2:] {
			e.Define(NewSymbol("$"+strconv.Itoa(i+1)),
				NewSymbol(v))
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
define caar: method (l) as: car: car l
define cadr: method (l) as: car: cdr l
define cdar: method (l) as: cdr: car l
define cddr: method (l) as: cdr: cdr l
define caaar: method (l) as: car: caar l
define caadr: method (l) as: car: cadr l
define cadar: method (l) as: car: cdar l
define caddr: method (l) as: car: cddr l
define cdaar: method (l) as: cdr: caar l
define cdadr: method (l) as: cdr: cadr l
define cddar: method (l) as: cdr: cdar l
define cdddr: method (l) as: cdr: cddr l
define caaaar: method (l) as: caar: caar l
define caaadr: method (l) as: caar: cadr l
define caadar: method (l) as: caar: cdar l
define caaddr: method (l) as: caar: cddr l
define cadaar: method (l) as: cadr: caar l
define cadadr: method (l) as: cadr: cadr l
define caddar: method (l) as: cadr: cdar l
define cadddr: method (l) as: cadr: cddr l
define cdaaar: method (l) as: cdar: caar l
define cdaadr: method (l) as: cdar: cadr l
define cdadar: method (l) as: cdar: cdar l
define cdaddr: method (l) as: cdar: cddr l
define cddaar: method (l) as: cddr: caar l
define cddadr: method (l) as: cddr: cadr l
define cdddar: method (l) as: cddr: cdar l
define cddddr: method (l) as: cddr: cddr l
define $connect: syntax (type out close) as {
    set type: eval type
    set close: eval close
    syntax e (left right) as {
        define p: type
        spawn {
            eval: list 'dynamic out 'p
            e::eval left
            if close: p::writer-close
        }

        dynamic $stdin p
        e::eval right
        if close: p::reader-close
    }
}
define $redirect: syntax (chan mode mthd) as {
    syntax e (c cmd) as {
        define c: e::eval c
        define f '()
        if (not: or (is-channel c) (is-pipe c)) {
            set f: open c mode
            set c f
        }
        eval: list 'dynamic chan 'c
        e::eval cmd
        if (not: is-null f): eval: cons 'f mthd
    }
}
define and: syntax e (: lst) as {
    define r False
    while (not: is-null: car lst) {
        set r: e::eval: car lst
        if (not r): return r
        set lst: cdr lst
    }
    return r
}
define append-stderr: $redirect $stderr "a" writer-close
define append-stdout: $redirect $stdout "a" writer-close
define apply: method (f: args) as: f @args
define backtick: syntax e (cmd) as {
    define p: pipe
    define r '()
    spawn {
        dynamic $stdout p
        e::eval cmd
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
define channel-stderr: $connect channel $stderr True
define channel-stdout: $connect channel $stdout True
define echo: builtin (: args) as: $stdout::write @args
define for: method (l m) as {
    define r: cons '() '()
    define c r
    while (not: is-null l) {
        set-cdr c: cons (m: car l) '()
        set c: cdr c
        set l: cdr l
    }
    return: cdr r
}
define glob: builtin (: args) as: return args
define import: syntax e (name) as {
    define m: module name
    if (or (is-null m) (is-object m)) {
        return m
    }

    define l: list 'source name
    set l: cons 'object: cons l '()
    set l: list 'Root::define m l
    e::eval l
}
define is-text: method (t) as: or (is-string t) (is-symbol t)
define object: syntax e (: body) as {
    e::eval: cons 'block: append body '(clone)
}
define or: syntax e (: lst) as {
    define r False
    while (not: is-null: car lst) {
	set r: e::eval: car lst
        if r: return r
        set lst: cdr lst 
    }
    return r
}
define pipe-stderr: $connect pipe $stderr True
define pipe-stdout: $connect pipe $stdout True
define printf: method (: args) as: echo: Text::sprintf (car args) @(cdr args)
define quote: syntax (cell) as: return cell
define read: builtin () as: $stdin::read
define readline: builtin () as: $stdin::readline
define redirect-stderr: $redirect $stderr "w" writer-close
define redirect-stdin: $redirect $stdin "r" reader-close
define redirect-stdout: $redirect $stdout "w" writer-close
define source: syntax e (name) as {
	define basename: e::eval name
	define paths: Text::split ":" $OHPATH
	define name basename

	while (and (not: is-null paths) (not: test -r name)) {
		set name: Text::join / (car paths) basename
		set paths: cdr paths
	}

	if (not: test -r name): set name basename

	define f: open name "r-"
	define l: f::read
	while l {
		e::eval l
		set l: f::read
	}
	f::reader-close
}
define write: method (: args) as: $stdout::write @args

List::public ref: method (k x) as: car: List::tail k x
List::public tail: method (k x) as {
    if k {
        List::tail (sub k 1): cdr x
    } else {
        return x
    }
}

test -r (Text::join / $HOME .ohrc) && source (Text::join / $HOME .ohrc)
`)), Evaluate)
}
