// Released under an MIT-style license. See LICENSE.

package task

import (
	"bufio"
	"fmt"
	"github.com/michaelmacinnis/adapted"
	"github.com/michaelmacinnis/oh/pkg/boot"
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/common"
	"github.com/peterh/liner"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type ui interface {
	Close() error
	Exists() bool
	ReadString(delim byte) (line string, err error)
}

type reader func(*Task, common.ReadStringer,
	func(string, uintptr) Cell, func(Cell))

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
	env0        *Env
	external    Cell
	interactive bool
	jobs        = map[int]*Task{}
	parse       reader
	pgid        int
	pid         int
	runnable    chan bool
	scope0      *Scope
	task0       *Task
)

var next = map[int64][]int64{
	psEvalArguments:        {SaveCdrCode, psEvalElement},
	psEvalArgumentsBuiltin: {SaveCdrCode, psEvalElementBuiltin},
	psExecIf:               {psEvalBlock},
	psExecWhileBody:        {psExecWhileTest, SaveCode, psEvalBlock},
}

/* Convert Context into a Conduit. (Return nil if not possible). */
func asConduit(o Context) Conduit {
	if c, ok := o.(Conduit); ok {
		return c
	}

	return nil
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

func init() {
	CacheSymbols(common.Symbols...)

	runnable = make(chan bool)
	close(runnable)

	builtin := NewBuiltin((*Task).External, Null, Null, Null, nil)
	external = NewUnbound(builtin)

	/* Root Scope. */
	scope0 = NewScope(nil, nil)

	/* Arithmetic. */
	bindArithmetic(scope0)

	/* Builtins. */
	scope0.DefineBuiltin("cd", func(t *Task, args Cell) bool {
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
	scope0.DefineBuiltin("debug", func(t *Task, args Cell) bool {
		t.Debug("debug")

		return false
	})
	scope0.DefineBuiltin("exists", func(t *Task, args Cell) bool {
		count := 0
		for ; args != Null; args = Cdr(args) {
			count++
			if _, err := os.Stat(raw(Car(args))); err != nil {
				return t.Return(False)
			}
		}

		return t.Return(NewBoolean(count > 0))
	})
	scope0.DefineBuiltin("fg", func(t *Task, args Cell) bool {
		if !jobControlEnabled() || t != task0 {
			return false
		}

		index := 0
		if args != Null {
			if a, ok := Car(args).(Atom); ok {
				index = int(a.Int())
			}
		} else {
			for k := range jobs {
				if k > index {
					index = k
				}
			}
		}

		found, ok := jobs[index]

		if !ok {
			return false
		}

		delete(jobs, index)

		setForegroundTask(found)

		return true
	})
	scope0.DefineBuiltin("jobs", func(t *Task, args Cell) bool {
		if !jobControlEnabled() || t != task0 ||
			len(jobs) == 0 {
			return false
		}

		i := make([]int, 0, len(jobs))
		for k := range jobs {
			i = append(i, k)
		}
		sort.Ints(i)
		for k, v := range i {
			if k != len(jobs)-1 {
				fmt.Printf("[%d] \t%d\t%s\n", v,
					jobs[v].Job.Group,
					jobs[v].Job.Command)
			} else {
				fmt.Printf("[%d]+\t%d\t%s\n", v,
					jobs[v].Job.Group,
					jobs[v].Job.Command)
			}
		}
		return false
	})
	scope0.DefineBuiltin("module", func(t *Task, args Cell) bool {
		str, err := module(raw(Car(args)))
		if err != nil {
			panic(err)
		}

		sym := NewSymbol(str)
		c := Resolve(t.Lexical, t.Dynamic, sym)

		if c == nil {
			return t.Return(sym)
		}

		return t.Return(c.Get())
	})
	scope0.DefineBuiltin("run", func(t *Task, args Cell) bool {
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

	/* Generators. */
	bindGenerators(scope0)

	scope0.DefineMethod("channel", func(t *Task, args Cell) bool {
		cap := 0
		if args != Null {
			cap = int(Car(args).(Atom).Int())
		}

		return t.Return(NewChannel(t, cap))
	})

	/* Predicates. */
	bindPredicates(scope0)

	/* Relational. */
	bindRelational(scope0)

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

	/* Standard Functions. */
	scope0.DefineMethod("append", func(t *Task, args Cell) bool {
		/*
		 * NOTE: oh's append works differently than Scheme's append.
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
	scope0.DefineMethod("exit", func(t *Task, args Cell) bool {
		t.Scratch = List(Car(args))

		t.Stop()

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
	scope0.DefineMethod("list-to-string", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewString(t, s))
	})
	scope0.DefineMethod("list-to-symbol", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewSymbol(s))
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
	scope0.DefineMethod("set-car", func(t *Task, args Cell) bool {
		SetCar(Car(args), Cadr(args))

		return t.Return(Cadr(args))
	})
	scope0.DefineMethod("set-cdr", func(t *Task, args Cell) bool {
		SetCdr(Car(args), Cadr(args))

		return t.Return(Cadr(args))
	})
	scope0.DefineMethod("temp-fifo", func(t *Task, args Cell) bool {
		name, err := adapted.TempFifo("fifo-")
		if err != nil {
			panic(err)
		}

		return t.Return(NewSymbol(name))
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

	/* Standard Methods. */
	scope0.PublicMethod("child", func(t *Task, args Cell) bool {
		return t.Return(NewObject(NewScope(t.Self().Expose(), nil)))
	})
	scope0.PublicMethod("clone", func(t *Task, args Cell) bool {
		return t.Return(NewObject(t.Self().Expose().Copy()))
	})
	scope0.PublicMethod("context", func(t *Task, args Cell) bool {
		self := t.Self()
		bare := self.Expose()
		if self == bare {
			self = NewObject(bare)
		}
		return t.Return(self)
	})
	scope0.PublicMethod("eval", func(t *Task, args Cell) bool {
		scope := t.Self().Expose()
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
	scope0.PublicMethod("get-slot", func(t *Task, args Cell) bool {
		s := raw(Car(args))
		k := NewSymbol(s)

		c := Resolve(t.Self(), nil, k)
		if c == nil {
			panic(s + " undefined")
		} else if a, ok := c.Get().(Binding); ok {
			return t.Return(a.Bind(t.Lexical))
		} else {
			return t.Return(c.Get())
		}
	})
	scope0.PublicMethod("has", func(t *Task, args Cell) bool {
		c := Resolve(t.Self(), t.Dynamic, NewSymbol(raw(Car(args))))

		return t.Return(NewBoolean(c != nil))
	})
	scope0.PublicMethod("interpolate", func(t *Task, args Cell) bool {
		original := raw(Car(args))

		l := t.Self()
		if t.Lexical == l.Expose() {
			l = t.Lexical
		}

		f := func(ref string) string {
			if ref == "$$" {
				return "$"
			}

			name := ref[2 : len(ref)-1]
			sym := NewSymbol(name)

			c := Resolve(l, t.Dynamic, sym)
			if c == nil {
				sym := NewSymbol("$" + name)
				c = Resolve(l, t.Dynamic, sym)
			}
			if c == nil {
				return "${" + name + "}"
			}

			return raw(c.Get())
		}

		r := regexp.MustCompile("(?:\\$\\$)|(?:\\${.+?})")
		modified := r.ReplaceAllStringFunc(original, f)

		return t.Return(NewString(t, modified))
	})
	scope0.PublicMethod("set-slot", func(t *Task, args Cell) bool {
		s := raw(Car(args))
		v := Cadr(args)

		k := NewSymbol(s)

		t.Self().Public(k, v)
		return t.Return(v)
	})
	scope0.PublicMethod("unset", func(t *Task, args Cell) bool {
		r := t.Self().Remove(NewSymbol(raw(Car(args))))

		return t.Return(NewBoolean(r))
	})

	/* Syntax. */
	scope0.DefineSyntax("block", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical, psEvalBlock)

		t.NewBlock(t.Dynamic, t.Lexical)

		return true
	})
	scope0.DefineSyntax("if", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical,
			psExecIf, SaveCode, psEvalElement)

		t.NewBlock(t.Dynamic, t.Lexical)

		t.Code = Car(t.Code)
		t.Scratch = Cdr(t.Scratch)

		return true
	})
	scope0.DefineSyntax("make-env", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic, psEvalBlock)

		t.Dynamic = NewEnv(t.Dynamic)

		return true
	})
	scope0.DefineSyntax("make-scope", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveLexical, psEvalBlock)

		t.Lexical = NewScope(t.Lexical, nil)

		return true
	})
	scope0.DefineSyntax("set", func(t *Task, args Cell) bool {
		t.Scratch = Cdr(t.Scratch)

		s := Null
		if Length(t.Code) == 3 {
			if raw(Cadr(t.Code)) != "=" {
				return t.Throw("error/syntax", "expected '='")
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
	scope0.DefineSyntax("while", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical, psExecWhileTest)

		return true
	})

	/* The rest. */
	bindTheRest(scope0)

	scope0.Public(NewSymbol("$root"), scope0)

	/* Root Environment. */
	env0 = NewEnv(nil)

	env0.Add(NewSymbol("false"), False)
	env0.Add(NewSymbol("true"), True)

	env0.Add(NewSymbol("$$"), NewInteger(int64(os.Getpid())))
	env0.Add(NewSymbol("$platform"), NewSymbol(Platform))
	env0.Add(NewSymbol("$stdin"), NewPipe(scope0, os.Stdin, nil))
	env0.Add(NewSymbol("$stdout"), NewPipe(scope0, nil, os.Stdout))
	env0.Add(NewSymbol("$stderr"), NewPipe(scope0, nil, os.Stderr))

	/* Environment variables. */
	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		env0.Add(NewSymbol("$"+kv[0]), NewSymbol(kv[1]))
	}

}

func isSimple(c Cell) bool {
	return IsAtom(c) || IsCons(c)
}

func jobControlEnabled() bool {
	return interactive && JobControlSupported()
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
	return toConduit(c.(Context)).(*Pipe).ReadFd()

}

func setForegroundTask(t *Task) {
	if t.Job.Group != 0 {
		SetForegroundGroup(t.Job.Group)
		t.Job.mode.ApplyMode()
	}
	task0, t = t, task0
	t.Stop()
	task0.Continue()
}

func status(c Cell) int {
	a, ok := c.(Atom)
	if !ok {
		return 0
	}
	return int(a.Status())
}

/* Convert Context into a Conduit. */
func toConduit(o Context) Conduit {
	conduit := asConduit(o)
	if conduit == nil {
		panic("not a conduit")
	}

	return conduit
}

/* Convert Context into a String. */
func toString(o Context) *String {
	if s, ok := o.(*String); ok {
		return s
	}

	panic("not a string")
}

func wpipe(c Cell) *os.File {
	return toConduit(c.(Context)).(*Pipe).WriteFd()
}

func ForegroundTask() *Task {
	return task0
}

func IsContext(c Cell) bool {
	switch c.(type) {
	case Context:
		return true
	}
	return false
}

func LaunchForegroundTask() {
	if task0 != nil {
		mode, _ := liner.TerminalMode()
		task0.Job.mode = mode
	}
	task0 = NewTask(Cons(nil, Null), nil, nil, nil)
	go task0.Listen()
}

func Pgid() int {
	return pgid
}

func Resolve(s Context, e *Env, k *Symbol) (v Reference) {
	v = nil

	if v = s.Access(k); v == nil {
		if e == nil {
			return v
		}

		v = e.Access(k)
	}

	return v
}

func Start(parser reader, cli ui) {
	LaunchForegroundTask()

	parse = parser
	eval := func(c Cell) {
		task0.Eval <- c
		<-task0.Done
	}

	b := bufio.NewReader(strings.NewReader(boot.Script))
	parse(nil, b, deref, eval)

	/* Command-line arguments */
	args := Null
	origin := ""
	if len(os.Args) > 1 {
		origin = filepath.Dir(os.Args[1])
		env0.Add(NewSymbol("$0"), NewSymbol(os.Args[1]))

		for i, v := range os.Args[2:] {
			k := "$" + strconv.Itoa(i+1)
			env0.Add(NewSymbol(k), NewSymbol(v))
		}

		for i := len(os.Args) - 1; i > 1; i-- {
			args = Cons(NewSymbol(os.Args[i]), args)
		}
	} else {
		env0.Add(NewSymbol("$0"), NewSymbol(os.Args[0]))
	}
	env0.Add(NewSymbol("$args"), args)

	if wd, err := os.Getwd(); err == nil {
		env0.Add(NewSymbol("$cwd"), NewSymbol(wd))
		if !filepath.IsAbs(origin) {
			origin = filepath.Join(wd, origin)
		}
		env0.Add(NewSymbol("$origin"), NewSymbol(origin))
	}

	interactive = false
	if len(os.Args) > 1 {
		eval(List(NewSymbol("source"), NewSymbol(os.Args[1])))
	} else if cli.Exists() {
		interactive = true

		InitSignalHandling()

		pgid = BecomeProcessGroupLeader()

		parse(nil, cli, deref, evaluate)

		cli.Close()
		fmt.Printf("\n")
	} else {
		eval(List(NewSymbol("source"), NewSymbol("/dev/stdin")))
	}

	os.Exit(0)
}

//go:generate ./generate.oh
//go:generate go fmt generated.go
