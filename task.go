// Released under an MIT-style license. See LICENSE.

package main

import (
	"bufio"
	"fmt"
	"github.com/peterh/liner"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode"
	"unsafe"
)

type Liner struct {
	*liner.State
}

func (cli *Liner) ReadString(delim byte) (line string, err error) {
	SetForegroundGroup(pgid)

	uncooked.ApplyMode()
	defer cooked.ApplyMode()

	if line, err = cli.State.Prompt("> "); err == nil {
		cli.AppendHistory(line)
		if task0.Job.command == "" {
			task0.Job.command = line
		}
		line += "\n"
	}
	return
}

type Conduit interface {
	Context

	Close()
	ReaderClose()
	Read(t *Task) Cell
	ReadLine(t *Task) Cell
	WriterClose()
	Write(c Cell)
}

type Notification struct {
	pid    int
	status syscall.WaitStatus
}

type Registration struct {
	pid int
	cb  chan Notification
}

const (
	SaveCarCode = 1 << iota
	SaveCdrCode
	SaveDynamic
	SaveLexical
	SaveScratch
	SaveMax
)

const (
	psChangeContext = SaveMax + iota

	psEvalArguments
	psEvalArgumentsBuiltin
	psEvalBlock
	psEvalCommand
	psEvalElement
	psEvalElementBuiltin
	psEvalMember

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
	psReturn

	psMax
	SaveCode = SaveCarCode | SaveCdrCode
)

var (
	cli         *Liner
	cooked      liner.ModeApplier
	env0        *Env
	envc        *Env
	envs        *Env
	external    Cell
	incoming    chan os.Signal
	interactive bool
	pgid        int
	pid         int
	register    chan Registration
	runnable    chan bool
	scope0      *Scope
	task0       *Task
	uncooked    liner.ModeApplier
)

var next = map[int64][]int64{
	psEvalArguments:        {SaveCdrCode, psEvalElement},
	psEvalArgumentsBuiltin: {SaveCdrCode, psEvalElementBuiltin},
	psExecIf:               {psEvalBlock},
	psExecWhileBody:        {psExecWhileTest, SaveCode, psEvalBlock},
}

/* Convert Context into a Conduit. (Return nil if not possible). */
func as_conduit(o Context) Conduit {
	if c, ok := o.(Conduit); ok {
		return c
	}

	return nil
}

func complete(line string, pos int) (string, []string, string) {
	head := line[:pos]
	tail := line[pos:]

	fields := strings.Fields(head)

	if len(fields) == 0 {
		return head, []string{"    "}, tail
	}

	word := fields[len(fields)-1]
	if !strings.HasSuffix(head, word) {
		return head, []string{}, tail
	}

	head = head[0 : len(head)-len(word)]

	completions := task0.Complete(word)
	completions = append(completions, files(word)...)
	if len(fields) == 1 {
		completions = append(completions, executables(word)...)
	}

	if len(completions) == 0 {
		return head, []string{word}, tail
	}

	unique := make(map[string]bool)
	for _, completion := range completions {
		unique[completion] = true
	}

	completions = make([]string, 0, len(unique))
	for completion := range unique {
		completions = append(completions, completion)
	}

	return head, completions, tail
}

func executables(word string) []string {
	completions := []string{}

	if strings.Contains(word, string(os.PathSeparator)) {
		return completions
	}

	pathenv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathenv, string(os.PathListSeparator)) {
		if dir == "" {
			dir = "."
		} else {
			dir = path.Clean(dir)
		}

		stat, err := os.Stat(dir)
		if err != nil || !stat.IsDir() {
			continue
		}

		max := strings.Count(dir, "/") + 1
		filepath.Walk(dir, func(p string, i os.FileInfo, err error) error {
			depth := strings.Count(p, "/")
			if depth > max {
				if i.IsDir() {
					return filepath.SkipDir
				}
				return nil
			} else if depth < max {
				return nil
			}

			_, basename := filepath.Split(p)

			if strings.HasPrefix(basename, word) {
				completions = append(completions, basename)
			}

			return nil
		})
	}

	return completions
}

func deref(name string, addr uintptr) Cell {
	switch {
	case name == "bound":
		return (*Bound)(unsafe.Pointer(addr))
	case name == "builtin":
		return (*Builtin)(unsafe.Pointer(addr))
	case name == "channel":
		return (*Channel)(unsafe.Pointer(addr))
	case name == "constant":
		return (*Constant)(unsafe.Pointer(addr))
	case name == "continuation":
		return (*Continuation)(unsafe.Pointer(addr))
	case name == "env":
		return (*Env)(unsafe.Pointer(addr))
	case name == "method":
		return (*Method)(unsafe.Pointer(addr))
	case name == "object":
		return (*Object)(unsafe.Pointer(addr))
	case name == "pipe":
		return (*Pipe)(unsafe.Pointer(addr))
	case name == "scope":
		return (*Scope)(unsafe.Pointer(addr))
	case name == "syntax":
		return (*Syntax)(unsafe.Pointer(addr))
	case name == "task":
		return (*Task)(unsafe.Pointer(addr))
	case name == "unbound":
		return (*Unbound)(unsafe.Pointer(addr))
	case name == "variable":
		return (*Variable)(unsafe.Pointer(addr))
	}

	return Null
}

func expand(t *Task, args Cell) Cell {
	list := Null

	for ; args != Null; args = Cdr(args) {
		c := Car(args)

		s := raw(c)
		if _, ok := c.(*Symbol); !ok {
			list = AppendTo(list, NewSymbol(s))
			continue
		}

		if s[:1] == "~" {
			s = filepath.Join(os.Getenv("HOME"), s[1:])
		}

		if strings.IndexAny(s, "*?[") == -1 {
			list = AppendTo(list, NewSymbol(s))
			continue
		}

		m, err := filepath.Glob(s)
		if err != nil || len(m) == 0 {
			panic("no matches found: " + s)
		}

		for _, v := range m {
			if v[0] != '.' || s[0] == '.' {
				e := NewString(t, v)
				list = AppendTo(list, e)
			}
		}
	}

	return list
}

func files(word string) []string {
	completions := []string{}

	candidate := word
	if candidate[:1] == "~" {
		candidate = filepath.Join(os.Getenv("HOME"), candidate[1:])
	}

	candidate = path.Clean(candidate)
	if !path.IsAbs(candidate) {
		ref := Resolve(task0.Lexical, task0.Dynamic, NewSymbol("$cwd"))
		cwd := ref.Get().String()

		candidate = path.Join(cwd, candidate)
	}

	dirname, basename := filepath.Split(candidate)
	if strings.HasSuffix(word, "/") {
		dirname, basename = path.Join(dirname, basename)+"/", ""
	}

	stat, err := os.Stat(dirname)
	if err != nil {
		return completions
	} else if len(basename) == 0 && !stat.IsDir() {
		return completions
	}

	max := strings.Count(dirname, "/")

	filepath.Walk(dirname, func(p string, i os.FileInfo, err error) error {
		depth := strings.Count(p, "/")
		if depth > max {
			if i.IsDir() {
				return filepath.SkipDir
			}
			return nil
		} else if depth < max {
			return nil
		}

		full := path.Join(dirname, basename)
		if len(basename) == 0 {
			if p == dirname {
				return nil
			}
			full += "/"
		} else if !strings.HasPrefix(p, full) {
			return nil
		}

		if i.IsDir() {
			p += "/"
		}

		if len(full) >= len(p) {
			return nil
		}

		completion := word + p[len(full):]
		completions = append(completions, completion)

		return nil
	})

	return completions
}

func init() {
	pid = BecomeProcessGroupLeader()
	pgid = pid

	signals := []os.Signal{InterruptRequest, StopRequest}
	incoming = make(chan os.Signal, len(signals))

	// We assume the terminal starts in cooked mode.
	cooked, _ = liner.TerminalMode()

	interactive = len(os.Args) <= 1 && cooked != nil
	if interactive {
		InitSignalHandling()

		cli = &Liner{liner.NewLiner()}

		uncooked, _ = liner.TerminalMode()

		cli.SetCtrlCAborts(true)
		cli.SetTabCompletionStyle(liner.TabPrints)
		cli.SetWordCompleter(complete)

		signal.Notify(incoming, signals...)
	}

	active := make(chan bool)
	notify := make(chan Notification)
	register = make(chan Registration)

	go Monitor(active, notify)
	go Registrar(active, notify)

	external = NewUnbound(NewBuiltin((*Task).External, Null, Null, Null, nil))

	envc = NewEnv(nil)
	envc.Method("child", func(t *Task, args Cell) bool {
		panic("conduits cannot be parents")
	})
	envc.Method("clone", func(t *Task, args Cell) bool {
		panic("conduits cannot be cloned")
	})
	envc.Method("define", func(t *Task, args Cell) bool {
		panic("private members cannot be added to a conduit")
	})
	envc.Method("close", func(t *Task, args Cell) bool {
		to_conduit(Car(t.Scratch).(Binding).Self()).Close()
		return t.Return(True)
	})
	envc.Method("reader-close", func(t *Task, args Cell) bool {
		to_conduit(Car(t.Scratch).(Binding).Self()).ReaderClose()
		return t.Return(True)
	})
	envc.Method("read", func(t *Task, args Cell) bool {
		r := to_conduit(Car(t.Scratch).(Binding).Self()).Read(t)
		return t.Return(r)
	})
	envc.Method("readline", func(t *Task, args Cell) bool {
		r := to_conduit(Car(t.Scratch).(Binding).Self()).ReadLine(t)
		return t.Return(r)
	})
	envc.Method("writer-close", func(t *Task, args Cell) bool {
		to_conduit(Car(t.Scratch).(Binding).Self()).WriterClose()
		return t.Return(True)
	})
	envc.Method("write", func(t *Task, args Cell) bool {
		to_conduit(Car(t.Scratch).(Binding).Self()).Write(args)
		return t.Return(True)
	})

	envs = NewEnv(nil)
	envs.Method("child", func(t *Task, args Cell) bool {
		panic("strings cannot be parents")
	})
	envs.Method("clone", func(t *Task, args Cell) bool {
		panic("strings cannot be cloned")
	})
	envs.Method("define", func(t *Task, args Cell) bool {
		panic("private members cannot be added to a string")
	})
	envs.Method("join", func(t *Task, args Cell) bool {
		sep := to_string(Car(t.Scratch).(Binding).Self())
		arr := make([]string, Length(args))

		for i := 0; args != Null; i++ {
			arr[i] = string(raw(Car(args)))
			args = Cdr(args)
		}

		r := strings.Join(arr, string(raw(sep)))

		return t.Return(NewString(t, r))
	})
	envs.Method("split", func(t *Task, args Cell) bool {
		r := Null

		sep := Car(args)
		str := to_string(Car(t.Scratch).(Binding).Self())

		l := strings.Split(string(raw(str)), string(raw(sep)))

		for i := len(l) - 1; i >= 0; i-- {
			r = Cons(NewString(t, l[i]), r)
		}

		return t.Return(r)
	})
	envs.Method("sprintf", func(t *Task, args Cell) bool {
		f := raw(to_string(Car(t.Scratch).(Binding).Self()))

		argv := []interface{}{}
		for l := args; l != Null; l = Cdr(l) {
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
				argv = append(argv, raw(t))
			}
		}

		s := fmt.Sprintf(f, argv...)

		return t.Return(NewRawString(t, s))
	})

	runnable = make(chan bool)
	close(runnable)

	env0 = NewEnv(nil)
	scope0 = NewScope(nil, nil)

	env0.Add(NewSymbol("false"), False)
	env0.Add(NewSymbol("true"), True)

	/* Command-line arguments */
	args := Null
	origin := ""
	if len(os.Args) > 1 {
		origin = filepath.Dir(os.Args[1])
		env0.Add(NewSymbol("$0"), NewSymbol(os.Args[1]))

		for i, v := range os.Args[2:] {
			env0.Add(NewSymbol("$"+strconv.Itoa(i+1)), NewSymbol(v))
		}

		for i := len(os.Args) - 1; i > 1; i-- {
			args = Cons(NewSymbol(os.Args[i]), args)
		}
	} else {
		env0.Add(NewSymbol("$0"), NewSymbol(os.Args[0]))
	}
	env0.Add(NewSymbol("$args"), args)

	env0.Add(NewSymbol("$$"), NewInteger(int64(syscall.Getpid())))

	if wd, err := os.Getwd(); err == nil {
		env0.Add(NewSymbol("$cwd"), NewSymbol(wd))
		if !filepath.IsAbs(origin) {
			origin = filepath.Join(wd, origin)
		}
		env0.Add(NewSymbol("$origin"), NewSymbol(origin))
	}

	env0.Add(NewSymbol("$platform"), NewSymbol(Platform))

	env0.Add(NewSymbol("$stdin"), NewPipe(scope0, os.Stdin, nil))
	env0.Add(NewSymbol("$stdout"), NewPipe(scope0, nil, os.Stdout))
	env0.Add(NewSymbol("$stderr"), NewPipe(scope0, nil, os.Stderr))

	/* Environment variables. */
	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		env0.Add(NewSymbol("$"+kv[0]), NewSymbol(kv[1]))
	}

	scope0.DefineSyntax("block", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical, psEvalBlock)

		t.NewBlock(t.Dynamic, t.Lexical)

		return true
	})
	scope0.DefineSyntax("builtin", func(t *Task, args Cell) bool {
		return t.Closure(NewBuiltin)
	})
	scope0.DefineSyntax("define", func(t *Task, args Cell) bool {
		return t.LexicalVar(psExecDefine)
	})
	scope0.DefineSyntax("dynamic", func(t *Task, args Cell) bool {
		return t.DynamicVar(psExecDynamic)
	})
	scope0.DefineSyntax("if", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical,
			psExecIf, SaveCode, psEvalElement)

		t.NewBlock(t.Dynamic, t.Lexical)

		t.Code = Car(t.Code)
		t.Scratch = Cdr(t.Scratch)

		return true
	})
	scope0.DefineSyntax("method", func(t *Task, args Cell) bool {
		return t.Closure(NewMethod)
	})
	scope0.DefineSyntax("set", func(t *Task, args Cell) bool {
		t.Scratch = Cdr(t.Scratch)

		s := Null
		if Length(t.Code) == 3 {
			if raw(Cadr(t.Code)) != "=" {
				panic("expected '='")
			}
			s = Caddr(t.Code)
		} else {
			s = Cadr(t.Code)
		}

		t.Code = Car(t.Code)
		if !IsCons(t.Code) {
			t.ReplaceStates(psExecSet, SaveCode)
		} else {
			t.ReplaceStates(SaveDynamic|SaveLexical,
				psExecSet, SaveCdrCode,
				psChangeContext, psEvalElement,
				SaveCarCode)
		}

		t.NewStates(psEvalElement)

		t.Code = s
		return true
	})
	scope0.DefineSyntax("setenv", func(t *Task, args Cell) bool {
		return t.DynamicVar(psExecSetenv)
	})
	scope0.DefineSyntax("spawn", func(t *Task, args Cell) bool {
		child := NewTask(t.Code, NewEnv(t.Dynamic),
			NewScope(t.Lexical, nil), t)

		go child.Launch()

		SetCar(t.Scratch, child)

		return false
	})
	scope0.DefineSyntax("splice", func(t *Task, args Cell) bool {
		t.ReplaceStates(psExecSplice, psEvalElement)

		t.Code = Car(t.Code)
		t.Scratch = Cdr(t.Scratch)

		return true
	})
	scope0.DefineSyntax("syntax", func(t *Task, args Cell) bool {
		return t.Closure(NewSyntax)
	})
	scope0.DefineSyntax("while", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical, psExecWhileTest)

		return true
	})

	scope0.PublicSyntax("public", func(t *Task, args Cell) bool {
		return t.LexicalVar(psExecPublic)
	})

	/* Builtins. */
	DefineBuiltin("cd", func(t *Task, args Cell) bool {
		err := os.Chdir(raw(Car(args)))
		status := 0
		if err != nil {
			status = 1
		}

		if wd, err := os.Getwd(); err == nil {
			t.Dynamic.Add(NewSymbol("$cwd"), NewSymbol(wd))
		}

		return t.Return(NewStatus(int64(status)))
	})
	DefineBuiltin("debug", func(t *Task, args Cell) bool {
		t.Debug("debug")

		return false
	})
	DefineBuiltin("exists", func(t *Task, args Cell) bool {
		count := 0
		for ; args != Null; args = Cdr(args) {
			count++
			if _, err := os.Stat(raw(Car(args))); err != nil {
				return t.Return(False)
			}
		}

		return t.Return(NewBoolean(count > 0))
	})
	DefineBuiltin("module", func(t *Task, args Cell) bool {
		str, err := module(raw(Car(args)))

		if err != nil {
			return t.Return(Null)
		}

		sym := NewSymbol(str)
		c := Resolve(t.Lexical, t.Dynamic, sym)

		if c == nil {
			return t.Return(sym)
		}

		return t.Return(c.Get())
	})
	DefineBuiltin("run", func(t *Task, args Cell) bool {
		if args == Null {
			SetCar(t.Scratch, False)
			return false
		}
		SetCar(t.Scratch, Car(args))
		t.Scratch = Cons(external, t.Scratch)
		t.Scratch = Cons(nil, t.Scratch)
		for args = Cdr(args); args != Null; args = Cdr(args) {
			t.Scratch = Cons(Car(args), t.Scratch)
		}
		t.ReplaceStates(psExecBuiltin)
		return true
	})

	scope0.PublicMethod("child", func(t *Task, args Cell) bool {
		o := Car(t.Scratch).(Binding).Self().Expose()

		return t.Return(NewObject(NewScope(o, nil)))
	})
	scope0.PublicMethod("clone", func(t *Task, args Cell) bool {
		o := Car(t.Scratch).(Binding).Self().Expose()

		return t.Return(NewObject(o.Copy()))
	})
	scope0.PublicMethod("context", func(t *Task, args Cell) bool {
		self := Car(t.Scratch).(Binding).Self()
		bare := self.Expose()
		if self == bare {
			self = NewObject(bare)
		}
		return t.Return(self)
	})
	scope0.DefineMethod("exit", func(t *Task, args Cell) bool {
		t.Scratch = List(Car(args))

		t.Stop()

		return true
	})
	scope0.PublicMethod("get-slot", func(t *Task, args Cell) bool {
		o := Car(t.Scratch).(Binding).Self()

		s := raw(Car(args))
		k := NewSymbol(s)

		c := Resolve(o, nil, k)
		if c == nil {
			panic(s + " undefined")
		} else if a, ok := c.Get().(Binding); ok {
			return t.Return(a.Bind(t.Lexical))
		} else {
			return t.Return(c.Get())
		}
	})
	scope0.PublicMethod("has", func(t *Task, args Cell) bool {
		l := Car(t.Scratch).(Binding).Self()
		c := Resolve(l, t.Dynamic, NewSymbol(raw(Car(args))))

		return t.Return(NewBoolean(c != nil))
	})
	scope0.PublicMethod("set-slot", func(t *Task, args Cell) bool {
		o := Car(t.Scratch).(Binding).Self()

		s := raw(Car(args))
		v := Cadr(args)

		k := NewSymbol(s)

		o.Public(k, v)
		return t.Return(v)
	})
	scope0.PublicMethod("unset", func(t *Task, args Cell) bool {
		l := Car(t.Scratch).(Binding).Self()
		r := l.Remove(NewSymbol(raw(Car(args))))

		return t.Return(NewBoolean(r))
	})

	scope0.DefineMethod("append", func(t *Task, args Cell) bool {
		/*
		 * NOTE: Our append works differently than Scheme's append.
		 *       To mimic Scheme's behavior use: append l1 @l2 ... @ln
		 */

		l := Car(args)
		n := Cons(Car(l), Null)
		s := n
		for l = Cdr(l); l != Null; l = Cdr(l) {
			SetCdr(n, Cons(Car(l), Null))
			n = Cdr(n)
		}
		SetCdr(n, Cdr(args))

		return t.Return(s)
	})
	scope0.DefineMethod("car", func(t *Task, args Cell) bool {
		return t.Return(Caar(args))
	})
	scope0.DefineMethod("cdr", func(t *Task, args Cell) bool {
		return t.Return(Cdar(args))
	})
	scope0.DefineMethod("cons", func(t *Task, args Cell) bool {
		return t.Return(Cons(Car(args), Cadr(args)))
	})
	scope0.PublicMethod("eval", func(t *Task, args Cell) bool {
		scope := Car(t.Scratch).(Binding).Self().Expose()
		t.RemoveState()
		if t.Lexical != scope {
			t.NewStates(SaveLexical)
			t.Lexical = scope
		}
		t.NewStates(psEvalElement)
		t.Code = Car(args)
		t.Scratch = Cdr(t.Scratch)

		return true
	})
	scope0.DefineMethod("length", func(t *Task, args Cell) bool {
		var l int64

		switch c := Car(args); c.(type) {
		case *String, *Symbol:
			l = int64(len(raw(c)))
		default:
			l = Length(c)
		}

		return t.Return(NewInteger(l))
	})
	scope0.DefineMethod("list", func(t *Task, args Cell) bool {
		return t.Return(args)
	})
	scope0.DefineMethod("open", func(t *Task, args Cell) bool {
		mode := raw(Car(args))
		path := raw(Cadr(args))
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

		f, err := os.OpenFile(path, flags, 0666)
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

		return t.Return(NewPipe(t.Lexical, r, w))
	})
	scope0.DefineMethod("reverse", func(t *Task, args Cell) bool {
		return t.Return(Reverse(Car(args)))
	})
	scope0.DefineMethod("set-car", func(t *Task, args Cell) bool {
		SetCar(Car(args), Cadr(args))

		return t.Return(Cadr(args))
	})
	scope0.DefineMethod("set-cdr", func(t *Task, args Cell) bool {
		SetCdr(Car(args), Cadr(args))

		return t.Return(Cadr(args))
	})
	scope0.DefineMethod("wait", func(t *Task, args Cell) bool {
		if args == Null {
			t.Wait()
		}
		list := args
		for ; args != Null; args = Cdr(args) {
			child := Car(args).(*Task)
			<-child.Done
			SetCar(args, Car(child.Scratch))
		}
		return t.Return(list)
	})

	bind_predicates(scope0)

	/* Generators. */
	scope0.DefineMethod("boolean", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(Car(args).Bool()))
	})
	scope0.DefineMethod("channel", func(t *Task, args Cell) bool {
		cap := 0
		if args != Null {
			cap = int(Car(args).(Atom).Int())
		}

		return t.Return(NewChannel(t, cap))
	})
	scope0.DefineMethod("float", func(t *Task, args Cell) bool {
		return t.Return(NewFloat(Car(args).(Atom).Float()))
	})
	scope0.DefineMethod("integer", func(t *Task, args Cell) bool {
		return t.Return(NewInteger(Car(args).(Atom).Int()))
	})
	scope0.DefineMethod("pipe", func(t *Task, args Cell) bool {
		return t.Return(NewPipe(t.Lexical, nil, nil))
	})
	scope0.DefineMethod("rational", func(t *Task, args Cell) bool {
		return t.Return(NewRational(Car(args).(Atom).Rat()))
	})
	scope0.DefineMethod("status", func(t *Task, args Cell) bool {
		return t.Return(NewStatus(Car(args).(Atom).Status()))
	})
	scope0.DefineMethod("string", func(t *Task, args Cell) bool {
		return t.Return(NewString(t, Car(args).String()))
	})
	scope0.DefineMethod("symbol", func(t *Task, args Cell) bool {
		return t.Return(NewSymbol(raw(Car(args))))
	})

	/* Relational. */
	scope0.DefineMethod("eq", func(t *Task, args Cell) bool {
		prev := Car(args)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args)

			if !prev.Equal(curr) {
				return t.Return(False)
			}

			prev = curr
		}

		return t.Return(True)
	})
	scope0.DefineMethod("ge", func(t *Task, args Cell) bool {
		prev := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Number)

			if prev.Less(curr) {
				return t.Return(False)
			}

			prev = curr
		}

		return t.Return(True)
	})
	scope0.DefineMethod("gt", func(t *Task, args Cell) bool {
		prev := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Number)

			if !prev.Greater(curr) {
				return t.Return(False)
			}

			prev = curr
		}

		return t.Return(True)
	})
	scope0.DefineMethod("is", func(t *Task, args Cell) bool {
		prev := Car(args)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args)

			if prev != curr {
				return t.Return(False)
			}

			prev = curr
		}

		return t.Return(True)
	})
	scope0.DefineMethod("le", func(t *Task, args Cell) bool {
		prev := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Number)

			if prev.Greater(curr) {
				return t.Return(False)
			}

			prev = curr
		}

		return t.Return(True)
	})
	scope0.DefineMethod("lt", func(t *Task, args Cell) bool {
		prev := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			curr := Car(args).(Number)

			if !prev.Less(curr) {
				return t.Return(False)
			}

			prev = curr
		}

		return t.Return(True)
	})
	scope0.DefineMethod("match", func(t *Task, args Cell) bool {
		pattern := raw(Car(args))
		text := raw(Cadr(args))

		ok, err := path.Match(pattern, text)
		if err != nil {
			panic(err)
		}

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("ne", func(t *Task, args Cell) bool {
		for l1 := args; l1 != Null; l1 = Cdr(l1) {
			for l2 := Cdr(l1); l2 != Null; l2 = Cdr(l2) {
				v1 := Car(l1)
				v2 := Car(l2)

				if v1.Equal(v2) {
					return t.Return(False)
				}
			}
		}

		return t.Return(True)
	})
	scope0.DefineMethod("not", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(!Car(args).Bool()))
	})

	/* Arithmetic. */
	scope0.DefineMethod("add", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Add(Car(args))

		}

		return t.Return(acc)
	})
	scope0.DefineMethod("sub", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Subtract(Car(args))
		}

		return t.Return(acc)
	})
	scope0.DefineMethod("div", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Divide(Car(args))
		}

		return t.Return(acc)
	})
	scope0.DefineMethod("mod", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Modulo(Car(args))
		}

		return t.Return(acc)
	})
	scope0.DefineMethod("mul", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Multiply(Car(args))
		}

		return t.Return(acc)
	})

	scope0.PublicMethod("interpolate", func(t *Task, args Cell) bool {
		original := raw(Car(args))

		l := Car(t.Scratch).(Binding).Self()
		if t.Lexical == l.Expose() {
			l = t.Lexical
		}

		f := func(old string) string {
			if old == "$$" {
				return "$"
			}

			name := old[2 : len(old)-1]
			sym := NewSymbol(name)

			c := Resolve(l, t.Dynamic, sym)
			if c == nil {
				return "${" + name + "}"
			}

			return raw(c.Get())
		}

		r := regexp.MustCompile("(?:\\$\\$)|(?:\\${.*?})")
		modified := r.ReplaceAllStringFunc(original, f)

		return t.Return(NewString(t, modified))
	})

	scope0.DefineMethod("list-to-string", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewRawString(t, s))
	})
	scope0.DefineMethod("list-to-symbol", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewSymbol(s))
	})

	/* Standard namespaces. */
	text := NewObject(NewScope(scope0, nil))
	scope0.Define(NewSymbol("$text"), text)

	text.PublicMethod("is-control", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsControl(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-digit", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsDigit(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-graphic", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsGraphic(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-letter", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLetter(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-lower", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsLower(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-mark", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsMark(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-print", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPrint(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-punct", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsPunct(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-space", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSpace(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-symbol", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsSymbol(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-title", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsTitle(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("is-upper", func(t *Task, args Cell) bool {
		var r Cell

		switch t := Car(args).(type) {
		case *Integer:
			r = NewBoolean(unicode.IsUpper(rune(t.Int())))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("join", func(t *Task, args Cell) bool {
		str := false
		sep := Car(args)
		list := Cdr(args)

		arr := make([]string, Length(list))

		for i := 0; list != Null; i++ {
			_, str = Car(list).(*String)
			arr[i] = string(raw(Car(list)))
			list = Cdr(list)
		}

		r := strings.Join(arr, string(raw(sep)))

		if str {
			return t.Return(NewString(t, r))
		}
		return t.Return(NewSymbol(r))
	})
	text.PublicMethod("split", func(t *Task, args Cell) bool {
		r := Null

		sep := Car(args)
		str := Cadr(args)

		l := strings.Split(string(raw(str)), string(raw(sep)))

		for i := len(l) - 1; i >= 0; i-- {
			switch str.(type) {
			case *Symbol:
				r = Cons(NewSymbol(l[i]), r)
			case *String:
				r = Cons(NewString(t, l[i]), r)
			}
		}

		return t.Return(r)
	})
	text.PublicMethod("substring", func(t *Task, args Cell) bool {
		var r Cell

		s := []rune(raw(Car(args)))

		start := int(Cadr(args).(Atom).Int())
		end := len(s)

		if Cddr(args) != Null {
			end = int(Caddr(args).(Atom).Int())
		}

		switch Car(args).(type) {
		case *String:
			r = NewString(t, string(s[start:end]))
		case *Symbol:
			r = NewSymbol(string(s[start:end]))
		default:
			r = Null
		}

		return t.Return(r)
	})
	text.PublicMethod("to-list", func(t *Task, args Cell) bool {
		l := Null
		for _, char := range raw(Car(args)) {
			l = Cons(NewInteger(int64(char)), l)
		}

		return t.Return(Reverse(l))
	})
	text.PublicMethod("lower", func(t *Task, args Cell) bool {
		var r Cell

		switch k := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToLower(rune(k.Int()))))
		case *String:
			r = NewString(t, strings.ToLower(raw(k)))
		case *Symbol:
			r = NewSymbol(strings.ToLower(raw(k)))
		default:
			r = NewInteger(0)
		}

		return t.Return(r)
	})
	text.PublicMethod("title", func(t *Task, args Cell) bool {
		var r Cell

		switch k := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToTitle(rune(k.Int()))))
		case *String:
			r = NewString(t, strings.ToTitle(raw(k)))
		case *Symbol:
			r = NewSymbol(strings.ToTitle(raw(k)))
		default:
			r = NewInteger(0)
		}

		return t.Return(r)
	})
	text.PublicMethod("upper", func(t *Task, args Cell) bool {
		var r Cell

		switch k := Car(args).(type) {
		case *Integer:
			r = NewInteger(int64(unicode.ToUpper(rune(k.Int()))))
		case *String:
			r = NewString(t, strings.ToUpper(raw(k)))
		case *Symbol:
			r = NewSymbol(strings.ToUpper(raw(k)))
		default:
			r = NewInteger(0)
		}

		return t.Return(r)
	})

	scope0.Public(NewSymbol("$root"), scope0)
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

func raw(c Cell) string {
	if s, ok := c.(*String); ok {
		return s.Raw()
	}

	return c.String()
}

func rpipe(c Cell) *os.File {
	return to_conduit(c.(Context)).(*Pipe).ReadFd()

}

func status(c Cell) int {
	a, ok := c.(Atom)
	if !ok {
		return 0
	}
	return int(a.Status())
}

/* Convert Context into a Conduit. */
func to_conduit(o Context) Conduit {
	conduit := as_conduit(o)
	if conduit == nil {
		panic("not a conduit")
	}

	return conduit
}

/* Convert Context into a String. */
func to_string(o Context) *String {
	if s, ok := o.(*String); ok {
		return s
	}

	panic("not a string")
}

func wpipe(c Cell) *os.File {
	return to_conduit(c.(Context)).(*Pipe).WriteFd()
}

func DefineBuiltin(k string, a Function) {
	scope0.DefineBuiltin(k, a)
}

func ForegroundTask() *Task {
	return task0
}

func Incoming() chan os.Signal {
	return incoming
}

func IsSimple(c Cell) bool {
	return IsAtom(c) || IsCons(c)
}

func JobControlEnabled() bool {
	return interactive && JobControlSupported()
}

func LaunchForegroundTask() {
	if task0 != nil {
		mode, _ := liner.TerminalMode()
		task0.Job.mode = mode
	}
	task0 = NewTask(Cons(nil, Null), nil, nil, nil)
	go task0.Listen()
}

func Pid() int {
	return pid
}

func SetForegroundTask(t *Task) {
	if t.Job.group != 0 {
		SetForegroundGroup(t.Job.group)
		t.Job.mode.ApplyMode()
	}
	task0, t = t, task0
	t.Stop()
	task0.Continue()
}

func Start(definitions string, eval func(c Cell)) {
	Parse(nil, bufio.NewReader(strings.NewReader(definitions)), deref, eval)

	if interactive {
		Parse(nil, cli, deref, eval)

		cli.Close()
		fmt.Printf("\n")
	} else if len(os.Args) > 1 {
		eval(List(NewSymbol("source"), NewSymbol(os.Args[1])))
	} else {
		eval(List(NewSymbol("source"), NewSymbol("/dev/stdin")))
	}

	os.Exit(0)
}

/* Channel cell definition. */

type Channel struct {
	*Scope
	v chan Cell
}

func IsChannel(c Cell) bool {
	context, ok := c.(Context)
	if !ok {
		return false
	}

	conduit := as_conduit(context)
	if conduit == nil {
		return false
	}

	switch conduit.(type) {
	case *Channel:
		return true
	}
	return false
}

func NewChannel(t *Task, cap int) Context {
	return &Channel{
		NewScope(t.Lexical.Expose(), envc),
		make(chan Cell, cap),
	}
}

func (ch *Channel) String() string {
	return fmt.Sprintf("%%channel %p%%", ch)
}

func (ch *Channel) Equal(c Cell) bool {
	return ch == c
}

func (ch *Channel) Close() {
	ch.WriterClose()
}

func (ch *Channel) Expose() Context {
	return ch
}

func (ch *Channel) ReaderClose() {
	return
}

func (ch *Channel) Read(t *Task) Cell {
	v := <-ch.v
	if v == nil {
		return Null
	}
	return v
}

func (ch *Channel) ReadLine(t *Task) Cell {
	v := <-ch.v
	if v == nil {
		return False
	}
	return NewString(t, v.String())
}

func (ch *Channel) WriterClose() {
	close(ch.v)
}

func (ch *Channel) Write(c Cell) {
	ch.v <- c
}

/* Continuation cell definition. */

type Continuation struct {
	Scratch Cell
	Stack   Cell
}

func NewContinuation(scratch Cell, stack Cell) *Continuation {
	return &Continuation{Scratch: scratch, Stack: stack}
}

func (ct *Continuation) Bool() bool {
	return true
}

func (ct *Continuation) Equal(c Cell) bool {
	return ct == c
}

func (ct *Continuation) String() string {
	return fmt.Sprintf("%%continuation %p%%", ct)
}

/* Job definition. */

type Job struct {
	*sync.Mutex
	command string
	group   int
	mode    liner.ModeApplier
}

func NewJob() *Job {
	mode, _ := liner.TerminalMode()
	return &Job{&sync.Mutex{}, "", 0, mode}
}

/* Pipe cell definition. */

type Pipe struct {
	*Scope
	b *bufio.Reader
	c chan Cell
	d chan bool
	r *os.File
	w *os.File
}

func IsPipe(c Cell) bool {
	context, ok := c.(Context)
	if !ok {
		return false
	}

	conduit := as_conduit(context)
	if conduit == nil {
		return false
	}

	switch conduit.(type) {
	case *Pipe:
		return true
	}
	return false
}

func NewPipe(l Context, r *os.File, w *os.File) Context {
	p := &Pipe{
		Scope: NewScope(l.Expose(), envc),
		b:     nil, c: nil, d: nil, r: r, w: w,
	}

	if r == nil && w == nil {
		var err error

		if p.r, p.w, err = os.Pipe(); err != nil {
			p.r, p.w = nil, nil
		}
	}

	runtime.SetFinalizer(p, (*Pipe).Close)

	return p
}

func (p *Pipe) String() string {
	return fmt.Sprintf("%%pipe %p%%", p)
}

func (p *Pipe) Equal(c Cell) bool {
	return p == c
}

func (p *Pipe) Close() {
	if p.r != nil && len(p.r.Name()) > 0 {
		p.ReaderClose()
	}

	if p.w != nil && len(p.w.Name()) > 0 {
		p.WriterClose()
	}
}

func (p *Pipe) Expose() Context {
	return p
}

func (p *Pipe) reader() *bufio.Reader {
	if p.b == nil {
		p.b = bufio.NewReader(p.r)
	}

	return p.b
}

func (p *Pipe) ReaderClose() {
	if p.r != nil {
		p.r.Close()
		p.r = nil
	}
}

func (p *Pipe) Read(t *Task) Cell {
	if p.r == nil {
		return Null
	}

	if p.c == nil {
		p.c = make(chan Cell)
		p.d = make(chan bool)
		go func() {
			Parse(t, p.reader(), deref, func(c Cell) {
				p.c <- c
				<-p.d
			})
			p.c <- Null
		}()
	} else {
		p.d <- true
	}

	return <-p.c
}

func (p *Pipe) ReadLine(t *Task) Cell {
	s, err := p.reader().ReadString('\n')
	if err != nil && len(s) == 0 {
		p.b = nil
		return Null
	}

	return NewString(t, strings.TrimRight(s, "\n"))
}

func (p *Pipe) WriterClose() {
	if p.w != nil {
		p.w.Close()
		p.w = nil
	}
}

func (p *Pipe) Write(c Cell) {
	if p.w == nil {
		panic("write to closed pipe")
	}

	fmt.Fprintln(p.w, c)
}

/* Pipe-specific functions */

func (p *Pipe) ReadFd() *os.File {
	return p.r
}

func (p *Pipe) WriteFd() *os.File {
	return p.w
}

/* Registers cell definition. */

type Registers struct {
	Continuation // Stack and Dump

	Code    Cell // Control
	Dynamic *Env
	Lexical Context
}

/* Registers-specific functions. */

func (r *Registers) Arguments() Cell {
	e := Car(r.Scratch)
	l := Null

	for e != nil {
		l = Cons(e, l)

		r.Scratch = Cdr(r.Scratch)
		e = Car(r.Scratch)
	}

	r.Scratch = Cdr(r.Scratch)

	return l
}

func (r *Registers) Complete(word string) []string {
	completions := r.Lexical.Complete(word)
	return append(completions, r.Dynamic.Complete(word)...)
}

func (r *Registers) GetState() int64 {
	if r.Stack == Null {
		return 0
	}
	return Car(r.Stack).(Atom).Int()
}

func (r *Registers) NewBlock(dynamic *Env, lexical Context) {
	r.Dynamic = NewEnv(dynamic)
	r.Lexical = NewScope(lexical, nil)
}

func (r *Registers) NewStates(l ...int64) {
	for _, f := range l {
		if f >= SaveMax {
			r.Stack = Cons(NewInteger(f), r.Stack)
			continue
		}

		if s := r.GetState(); s < SaveMax && f&s == f {
			continue
		}

		if f&SaveCode > 0 {
			if f&SaveCode == SaveCode {
				r.Stack = Cons(r.Code, r.Stack)
			} else if f&SaveCarCode > 0 {
				r.Stack = Cons(Car(r.Code), r.Stack)
			} else if f&SaveCdrCode > 0 {
				r.Stack = Cons(Cdr(r.Code), r.Stack)
			}
		}

		if f&SaveDynamic > 0 {
			r.Stack = Cons(r.Dynamic, r.Stack)
		}

		if f&SaveLexical > 0 {
			r.Stack = Cons(r.Lexical, r.Stack)
		}

		if f&SaveScratch > 0 {
			r.Stack = Cons(r.Scratch, r.Stack)
		}

		r.Stack = Cons(NewInteger(f), r.Stack)
	}
}

func (r *Registers) RemoveState() {
	f := r.GetState()

	r.Stack = Cdr(r.Stack)
	if f >= SaveMax {
		return
	}

	if f&SaveScratch > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if f&SaveLexical > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if f&SaveDynamic > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if f&SaveCode > 0 {
		r.Stack = Cdr(r.Stack)
	}
}

func (r *Registers) ReplaceStates(l ...int64) {
	r.RemoveState()
	r.NewStates(l...)
}

func (r *Registers) RestoreState() {
	f := r.GetState()

	if f == 0 || f >= SaveMax {
		return
	}

	if f&SaveScratch > 0 {
		r.Stack = Cdr(r.Stack)
		r.Scratch = Car(r.Stack)
	}

	if f&SaveLexical > 0 {
		r.Stack = Cdr(r.Stack)
		r.Lexical = Car(r.Stack).(Context)
	}

	if f&SaveDynamic > 0 {
		r.Stack = Cdr(r.Stack)
		r.Dynamic = Car(r.Stack).(*Env)
	}

	if f&SaveCode > 0 {
		r.Stack = Cdr(r.Stack)
		r.Code = Car(r.Stack)
	}

	r.Stack = Cdr(r.Stack)
}

func (r *Registers) Return(rv Cell) bool {
	SetCar(r.Scratch, rv)

	return false
}

/* String cell definition. */

type String struct {
	*Scope
	v string
}

func IsString(c Cell) bool {
	switch c.(type) {
	case *String:
		return true
	}
	return false
}

func NewRawString(t *Task, v string) *String {
	p, ok := str[v]

	if ok {
		return p
	}

	l := scope0
	if t != nil {
		l = NewScope(t.Lexical.Expose(), envs)
	} else if task0 != nil {
		l = NewScope(task0.Lexical.Expose(), envs)
	} else {
		l = NewScope(l, envs)
	}

	s := String{l, v}
	p = &s

	if len(v) <= 8 {
		str[v] = p
	}

	return p
}

func NewString(t *Task, q string) Context {
	v, _ := strconv.Unquote("\"" + q + "\"")

	return NewRawString(t, v)
}

func (s *String) Bool() bool {
	return true
}

func (s *String) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(s.v) == a.String()
	}
	return false
}

func (s *String) String() string {
	return strconv.Quote(string(s.v))
}

func (s *String) Float() (f float64) {
	var err error
	if f, err = strconv.ParseFloat(string(s.v), 64); err != nil {
		panic(err)
	}
	return f
}

func (s *String) Int() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(s.v), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *String) Rat() *big.Rat {
	r := new(big.Rat)
	if _, err := fmt.Sscan(string(s.v), r); err != nil {
		panic(err)
	}
	return r
}

func (s *String) Status() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(s.v), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *String) Expose() Context {
	return s
}

/* String-specific functions. */

func (s *String) Raw() string {
	return string(s.v)
}

/* Task cell definition. */

type Task struct {
	*Job
	*Registers
	Done      chan Cell
	Eval      chan Cell
	children  map[*Task]bool
	parent    *Task
	pid       int
	suspended chan bool
}

func NewTask(c Cell, d *Env, l Context, p *Task) *Task {
	if d == nil {
		d = env0
	}

	if l == nil {
		l = scope0
	}

	var j *Job
	if p == nil {
		j = NewJob()
	} else {
		j = p.Job
	}

	t := &Task{
		Job: j,
		Registers: &Registers{
			Continuation: Continuation{
				Scratch: List(NewStatus(0)),
				Stack:   List(NewInteger(psEvalBlock)),
			},
			Code:    c,
			Dynamic: d,
			Lexical: l,
		},
		Done:      make(chan Cell, 1),
		Eval:      make(chan Cell, 1),
		children:  make(map[*Task]bool),
		parent:    p,
		pid:       0,
		suspended: runnable,
	}

	if p != nil {
		p.children[t] = true
	}

	return t
}

func (t *Task) Bool() bool {
	return true
}

func (t *Task) String() string {
	return fmt.Sprintf("%%task %p%%", t)
}

func (t *Task) Equal(c Cell) bool {
	return t == c
}

/* Task-specific functions. */

func (t *Task) Apply(args Cell) bool {
	m := Car(t.Scratch).(Binding)

	t.ReplaceStates(SaveDynamic|SaveLexical, psEvalBlock)

	t.Code = m.Ref().Body()
	t.NewBlock(t.Dynamic, m.Ref().Scope())

	label := m.Ref().Label()
	if label != Null {
		t.Lexical.Public(label, m.Self().Expose())
	}

	params := m.Ref().Params()
	for args != Null && params != Null && IsAtom(Car(params)) {
		t.Lexical.Public(Car(params), Car(args))
		args, params = Cdr(args), Cdr(params)
	}
	if IsCons(Car(params)) {
		t.Lexical.Public(Caar(params), args)
	}

	cc := NewContinuation(Cdr(t.Scratch), t.Stack)
	t.Lexical.Public(NewSymbol("return"), cc)

	return true
}

func (t *Task) Closure(n ClosureGenerator) bool {
	label := Null
	params := Car(t.Code)
	for t.Code != Null && raw(Cadr(t.Code)) != "as" {
		label = params
		params = Cadr(t.Code)
		t.Code = Cdr(t.Code)
	}

	if t.Code == Null {
		panic("expected 'as'")
	}

	body := Cddr(t.Code)
	scope := t.Lexical

	c := n((*Task).Apply, body, label, params, scope)
	if label == Null {
		SetCar(t.Scratch, NewUnbound(c))
	} else {
		SetCar(t.Scratch, NewBound(c, scope))
	}

	return false
}

func (t *Task) Continue() {
	if t.pid > 0 {
		ContinueProcess(t.pid)
	}

	for k, v := range t.children {
		if v {
			k.Continue()
		}
	}

	close(t.suspended)
}

func (t *Task) Debug(s string) {
	fmt.Printf("%s: t.Code = %v, t.Scratch = %v\n", s, t.Code, t.Scratch)
}

func (t *Task) DynamicVar(state int64) bool {
	r := raw(Car(t.Code))
	if t.Strict() && number(r) {
		panic(r + " cannot be used as a variable name")
	}

	if state == psExecSetenv {
		if !strings.HasPrefix(r, "$") {
			panic("environment variable names must begin with '$'")
		}
	}

	t.ReplaceStates(state, SaveCarCode|SaveDynamic, psEvalElement)

	if Length(t.Code) == 3 {
		if raw(Cadr(t.Code)) != "=" {
			panic("expected '=' after " + r)
		}
		t.Code = Caddr(t.Code)
	} else {
		t.Code = Cadr(t.Code)
	}

	t.Scratch = Cdr(t.Scratch)

	return true
}

func (t *Task) Execute(arg0 string, argv []string, attr *os.ProcAttr) (*Status, error) {

	t.Lock()

	if JobControlEnabled() {
		attr.Sys = SysProcAttr(t.group)
	}

	proc, err := os.StartProcess(arg0, argv, attr)
	if err != nil {
		t.Unlock()
		return nil, err
	}

	if JobControlEnabled() {
		if t.group == 0 {
			t.group = proc.Pid
		}
	}

	t.pid = proc.Pid

	t.Unlock()

	status := JoinProcess(proc)

	if JobControlEnabled() {
		if t.group == t.pid {
			t.group = 0
		}
	}
	t.pid = 0

	return NewStatus(int64(status)), err
}

func (t *Task) External(args Cell) bool {
	t.Scratch = Cdr(t.Scratch)

	arg0, problem := exec.LookPath(raw(Car(t.Scratch)))

	SetCar(t.Scratch, False)

	if problem != nil {
		panic(problem)
	}

	argv := []string{arg0}

	for ; args != Null; args = Cdr(args) {
		argv = append(argv, raw(Car(args)))
	}

	c := Resolve(t.Lexical, t.Dynamic, NewSymbol("$cwd"))
	dir := c.Get().String()

	in := Resolve(t.Lexical, t.Dynamic, NewSymbol("$stdin")).Get()
	out := Resolve(t.Lexical, t.Dynamic, NewSymbol("$stdout")).Get()
	err := Resolve(t.Lexical, t.Dynamic, NewSymbol("$stderr")).Get()

	files := []*os.File{rpipe(in), wpipe(out), wpipe(err)}

	attr := &os.ProcAttr{Dir: dir, Env: nil, Files: files}

	status, problem := t.Execute(arg0, argv, attr)
	if problem != nil {
		panic(problem)
	}

	return t.Return(status)
}

func (t *Task) Launch() {
	t.Run(nil)
	close(t.Done)
}

func (t *Task) Listen() {
	for c := range t.Eval {
		saved := *(t.Registers)

		end := Cons(nil, Null)

		SetCar(t.Code, c)
		SetCdr(t.Code, end)

		t.Code = end
		t.NewStates(SaveCode, psEvalCommand)

		t.Code = c
		if !t.Run(end) {
			*(t.Registers) = saved

			SetCar(t.Code, nil)
			SetCdr(t.Code, Null)
		}

		t.Done <- nil
	}
}

func (t *Task) LexicalVar(state int64) bool {
	t.RemoveState()

	l := Car(t.Scratch).(Binding).Self().Expose()
	if t.Lexical != l {
		t.NewStates(SaveLexical)
		t.Lexical = l
	}

	t.NewStates(state)

	r := raw(Car(t.Code))
	if t.Strict() && number(r) {
		panic(r + " cannot be used as a variable name")
	}

	t.NewStates(SaveCarCode|SaveLexical, psEvalElement)

	if Length(t.Code) == 3 {
		if raw(Cadr(t.Code)) != "=" {
			panic("expected '=' after " + r)
		}
		t.Code = Caddr(t.Code)
	} else {
		t.Code = Cadr(t.Code)
	}

	t.Scratch = Cdr(t.Scratch)

	return true
}

func (t *Task) Lookup(sym *Symbol, simple bool) (bool, string) {
	c := Resolve(t.Lexical, t.Dynamic, sym)
	if c == nil {
		r := raw(sym)
		if t.GetState() == psEvalMember || (t.Strict() && !number(r)) {
			return false, r + " undefined"
		}
		t.Scratch = Cons(sym, t.Scratch)
	} else if simple && !IsSimple(c.Get()) {
		t.Scratch = Cons(sym, t.Scratch)
	} else if a, ok := c.Get().(Binding); ok {
		t.Scratch = Cons(a.Bind(t.Lexical), t.Scratch)
	} else {
		t.Scratch = Cons(c.Get(), t.Scratch)
	}

	return true, ""
}

func (t *Task) Run(end Cell) (successful bool) {
	successful = true

	defer func() {
		r := recover()
		if r == nil {
			return
		}

		fmt.Printf("oh: %v\n", r)

		successful = false
	}()

	for t.Runnable() && t.Stack != Null {
		state := t.GetState()

		switch state {
		case psChangeContext:
			t.Dynamic = nil
			t.Lexical = Car(t.Scratch).(Context)
			t.Scratch = Cdr(t.Scratch)

		case psExecBuiltin, psExecMethod:
			args := t.Arguments()

			if state == psExecBuiltin {
				args = expand(t, args)
			}

			t.Code = args

			fallthrough
		case psExecSyntax:
			m := Car(t.Scratch).(Binding)

			if m.Ref().Applier()(t, t.Code) {
				continue
			}

		case psExecIf, psExecWhileBody:
			if !Car(t.Scratch).Bool() {
				t.Code = Cdr(t.Code)

				for Car(t.Code) != Null &&
					!IsAtom(Car(t.Code)) {
					t.Code = Cdr(t.Code)
				}

				if Car(t.Code) != Null &&
					raw(Car(t.Code)) != "else" {
					panic("expected 'else'")
				}
			}

			if Cdr(t.Code) == Null {
				break
			}

			t.ReplaceStates(next[t.GetState()]...)

			t.Code = Cdr(t.Code)

			fallthrough
		case psEvalBlock:
			if t.Code == end {
				t.Scratch = Cdr(t.Scratch)
				return
			}

			if t.Code == Null ||
				!IsCons(t.Code) || !IsCons(Car(t.Code)) {
				break
			}

			if Cdr(t.Code) == Null || !IsCons(Cadr(t.Code)) {
				t.ReplaceStates(psEvalCommand)
			} else {
				t.NewStates(SaveCdrCode, psEvalCommand)
			}

			t.Code = Car(t.Code)
			t.Scratch = Cdr(t.Scratch)

			fallthrough
		case psEvalCommand:
			if t.Code == Null {
				t.Scratch = Cons(t.Code, t.Scratch)
				break
			}

			t.ReplaceStates(psExecCommand,
				SaveCdrCode,
				psEvalElement)
			t.Code = Car(t.Code)

			continue

		case psExecCommand:
			switch k := Car(t.Scratch).(type) {
			case *String, *Symbol:
				t.Scratch = Cons(external, t.Scratch)

				t.ReplaceStates(psExecBuiltin,
					psEvalArgumentsBuiltin)
			case Binding:
				switch k.Ref().(type) {
				case *Builtin:
					t.ReplaceStates(psExecBuiltin,
						psEvalArgumentsBuiltin)

				case *Method:
					t.ReplaceStates(psExecMethod,
						psEvalArguments)
				case *Syntax:
					t.ReplaceStates(psExecSyntax)
					continue
				}

			case *Continuation:
				t.ReplaceStates(psReturn, psEvalArguments)

			default:
				panic(fmt.Sprintf("can't evaluate: %v", t))
			}

			t.Scratch = Cons(nil, t.Scratch)

			fallthrough
		case psEvalArguments, psEvalArgumentsBuiltin:
			if t.Code == Null {
				break
			}

			t.NewStates(next[t.GetState()]...)

			t.Code = Car(t.Code)

			fallthrough
		case psEvalElement, psEvalElementBuiltin, psEvalMember:
			if t.Code == Null {
				t.Scratch = Cons(t.Code, t.Scratch)
				break
			} else if IsCons(t.Code) {
				if IsAtom(Cdr(t.Code)) {
					t.ReplaceStates(SaveDynamic|SaveLexical,
						psEvalMember,
						psChangeContext,
						SaveCdrCode,
						psEvalElement)
					t.Code = Car(t.Code)
				} else {
					t.ReplaceStates(psEvalCommand)
				}
				continue
			} else if sym, ok := t.Code.(*Symbol); ok {
				simple := t.GetState() == psEvalElementBuiltin
				ok, msg := t.Lookup(sym, simple)
				if !ok {
					panic(msg)
				}
				break
			} else {
				t.Scratch = Cons(t.Code, t.Scratch)
				break
			}

		case psExecDefine:
			t.Lexical.Define(t.Code, Car(t.Scratch))

		case psExecPublic:
			t.Lexical.Public(t.Code, Car(t.Scratch))

		case psExecDynamic, psExecSetenv:
			k := t.Code
			v := Car(t.Scratch)

			if state == psExecSetenv {
				s := raw(v)
				os.Setenv(strings.TrimLeft(k.String(), "$"), s)
			}

			t.Dynamic.Add(k, v)

		case psExecSet:
			k := t.Code.(*Symbol)
			r := Resolve(t.Lexical, t.Dynamic, k)
			if r == nil {
				panic("'" + k.String() + "' undefined")
			}

			r.Set(Car(t.Scratch))

		case psExecSplice:
			l := Car(t.Scratch)
			t.Scratch = Cdr(t.Scratch)

			if !IsCons(l) {
				break
			}

			for l != Null {
				t.Scratch = Cons(Car(l), t.Scratch)
				l = Cdr(l)
			}

		case psExecWhileTest:
			t.ReplaceStates(psExecWhileBody,
				SaveCode,
				psEvalElement)
			t.Code = Car(t.Code)
			t.Scratch = Cdr(t.Scratch)

			continue

		case psReturn:
			args := t.Arguments()

			t.Continuation = *Car(t.Scratch).(*Continuation)
			t.Scratch = Cons(Car(args), t.Scratch)

			break

		default:
			if state >= SaveMax {
				panic(fmt.Sprintf("command not found: %s",
					t.Code))
			} else {
				t.RestoreState()
				continue
			}
		}

		t.RemoveState()
	}

	return
}

func (t *Task) Runnable() bool {
	return !<-t.suspended
}

func (t *Task) Stop() {
	t.Stack = Null
	close(t.Eval)

	select {
	case <-t.suspended:
	default:
		close(t.suspended)
	}

	if t.pid > 0 {
		TerminateProcess(t.pid)
	}

	for k, v := range t.children {
		if v {
			k.Stop()
		}
	}
}

func (t *Task) Strict() (ok bool) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		ok = false
	}()

	c := Resolve(t.Lexical, nil, NewSymbol("strict"))
	if c == nil {
		return false
	}

	return c.Get().(Cell).Bool()
}

func (t *Task) Suspend() {
	//	if t.pid > 0 {
	//		syscall.Kill(t.pid, syscall.SIGSTOP)
	//	}

	for k, v := range t.children {
		if v {
			k.Suspend()
		}
	}

	t.suspended = make(chan bool)
}

func (t *Task) Wait() {
	for k, v := range t.children {
		if v {
			<-k.Done
		}
		delete(t.children, k)
	}
}
