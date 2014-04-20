/*
Oh is a Unix shell.  It is similar in spirit but different in detail from
other Unix shells. The following commands behave as expected:

    date
    cat /usr/share/dict/words
    who >user.names
    who >>user.names
    wc <file
    echo [a-f]*.c
    who | wc
    who; date
    cc *.c &
    mkdir junk && cd junk
    cd ..
    rm -r junk || echo "rm failed!"

For more detail, see: https://github.com/michaelmacinnis/oh

Oh is released under an MIT-style license.
*/
package main

import (
	"bufio"
	"fmt"
	"github.com/michaelmacinnis/liner"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

const (
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
	psReturn

	psMax
)

type Liner struct {
	*liner.State
}

func (cli *Liner) ReadString(delim byte) (line string, err error) {
	cli.LineEditingMode()
	defer cli.OriginalTerminalMode()

	if line, err = cli.State.Prompt("> "); err == nil {
		cli.AppendHistory(line)
		if command == "" {
			command = line
		}
		line += "\n"
	}
	return
}

func complete(line string) []string {
	fields := strings.Fields(line)

	if len(fields) == 0 {
		return []string{"    " + line}
	}

	prefix := fields[len(fields)-1]
	if !strings.HasSuffix(line, prefix) {
		return []string{line}
	}

	trimmed := line[0 : len(line)-len(prefix)]

	completions := files(trimmed, prefix)
	completions = append(completions, task0.Complete(trimmed, prefix)...)

	if len(completions) == 0 {
		return []string{line}
	}

	return completions
}

func files(line, prefix string) []string {
	completions := []string{}

	prfx := path.Clean(prefix)
	if !path.IsAbs(prfx) {
		ref := Resolve(task0.Lexical, task0.Dynamic, NewSymbol("$cwd"))
		cwd := ref.Get().String()

		prfx = path.Join(cwd, prfx)
	}

	root, prfx := filepath.Split(prfx)
	if strings.HasSuffix(prefix, "/") {
		root, prfx = path.Join(root, prfx)+"/", ""
	}
	max := strings.Count(root, "/")

	filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		depth := strings.Count(p, "/")
		if depth > max {
			if i.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		} else if depth == max {
			full := path.Join(root, prfx)
			if len(prfx) == 0 {
				full += "/"
			} else if !strings.HasPrefix(p, full) {
				return nil
			}

			completion := line + prefix + p[len(full):]
			completions = append(completions, completion)
		}

		return nil
	})

	return completions
}

var cli *Liner

var done0 chan Cell
var eval0 chan Cell

var command = ""
var ext Cell
var interactive bool
var jobs = map[*Task]int{}
var lines = map[*Task]string{}

var next = map[int64][]int64{
	psEvalArguments:        {SaveCdrCode, psEvalElement},
	psEvalArgumentsBuiltin: {SaveCdrCode, psEvalElementBuiltin},
	psExecIf:               {psEvalBlock},
	psExecWhileBody:        {psExecWhileTest, SaveCode, psEvalBlock},
}

var task0 *Task

func apply(t *Task, args Cell) bool {
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

func combiner(t *Task, n NewCombiner) bool {
	label := Null
	params := Car(t.Code)
	for t.Code != Null && Raw(Cadr(t.Code)) != "as" {
		label = params
		params = Cadr(t.Code)
		t.Code = Cdr(t.Code)
	}

	if t.Code == Null {
		panic("expected 'as'")
	}

	body := Cddr(t.Code)
	scope := t.Lexical

	c := n(apply, body, label, params, scope)
	if label == Null {
		SetCar(t.Scratch, NewUnbound(c))
	} else {
		SetCar(t.Scratch, NewBound(c, scope))
	}

	return false
}

func debug(t *Task, s string) {
	fmt.Printf("%s: t.Code = %v, t.Scratch = %v\n", s, t.Code, t.Scratch)
}

func dynamic(t *Task, state int64) bool {
	r := Raw(Car(t.Code))
	if strict(t) && number(r) {
		panic(r + " cannot be used as a variable name")
	}

	if state == psExecSetenv {
		if !strings.HasPrefix(r, "$") {
			panic("environment variable names must begin with '$'")
		}
	}

	t.ReplaceStates(state, SaveCarCode|SaveDynamic, psEvalElement)

	t.Code = Cadr(t.Code)
	t.Scratch = Cdr(t.Scratch)

	return true
}

func evaluate(c Cell) {
	eval0 <- c
	<-done0
	command = ""
}

func expand(args Cell) Cell {
	list := Null

	for ; args != Null; args = Cdr(args) {
		c := Car(args)

		s := Raw(c)
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
				e := NewSymbol(v)
				list = AppendTo(list, e)
			}
		}
	}

	return list
}

func external(t *Task, args Cell) bool {
	t.Scratch = Cdr(t.Scratch)

	name, problem := exec.LookPath(Raw(Car(t.Scratch)))

	SetCar(t.Scratch, False)

	if problem != nil {
		panic(problem)
	}

	argv := []string{name}

	for ; args != Null; args = Cdr(args) {
		argv = append(argv, Car(args).String())
	}

	c := Resolve(t.Lexical, t.Dynamic, NewSymbol("$cwd"))
	dir := c.Get().String()

	in := rpipe(Resolve(t.Lexical, t.Dynamic, NewSymbol("$stdin")).Get())
	out := wpipe(Resolve(t.Lexical, t.Dynamic, NewSymbol("$stdout")).Get())
	err := wpipe(Resolve(t.Lexical, t.Dynamic, NewSymbol("$stderr")).Get())

	fd := []*os.File{in, out, err}
	attr := &os.ProcAttr{Dir: dir, Env: nil, Files: fd}
	proc, problem := os.StartProcess(name, argv, attr)
	if problem != nil {
		panic(problem)
	}

	t.Pid(proc.Pid)

	var status int64 = 0
	msg, problem := proc.Wait()
	if problem != nil {
		panic(problem)
	}

	status = int64(msg.Sys().(syscall.WaitStatus).ExitStatus())

	t.Pid(0)

	return t.Return(NewStatus(status))
}

func init() {
	runtime.LockOSThread()
	syscall.Setpgid(0, 0)
}

func launch(task *Task) {
	run(task, nil)
	close(task.Done)
}

func listen(task *Task) {
	for c := range task.Eval {
		saved := *(task.Registers)

		end := Cons(nil, Null)

		SetCar(task.Code, c)
		SetCdr(task.Code, end)

		task.Code = end
		task.NewStates(SaveCode, psEvalCommand)

		task.Code = c
		if !run(task, end) {
			*(task.Registers) = saved

			SetCar(task.Code, nil)
			SetCdr(task.Code, Null)
		}

		task.Done <- nil
	}
}

func lexical(t *Task, state int64) bool {
	t.RemoveState()

	l := Car(t.Scratch).(Binding).Self().Expose()
	if t.Lexical != l {
		t.NewStates(SaveLexical)
		t.Lexical = l
	}

	t.NewStates(state)

	r := Raw(Car(t.Code))
	if strict(t) && number(r) {
		panic(r + " cannot be used as a variable name")
	}

	t.NewStates(SaveCarCode|SaveLexical, psEvalElement)

	t.Code = Cadr(t.Code)
	t.Scratch = Cdr(t.Scratch)

	return true
}

func lookup(t *Task, sym *Symbol, simple bool) (bool, string) {
	c := Resolve(t.Lexical, t.Dynamic, sym)
	if c == nil {
		r := Raw(sym)
		if strict(t) && !number(r) {
			return false, r + " undefined"
		} else {
			t.Scratch = Cons(sym, t.Scratch)
		}
	} else if simple && !IsSimple(c.Get()) {
		t.Scratch = Cons(sym, t.Scratch)
	} else if a, ok := c.Get().(Binding); ok {
		t.Scratch = Cons(a.Bind(t.Lexical), t.Scratch)
	} else {
		t.Scratch = Cons(c.Get(), t.Scratch)
	}

	return true, ""
}

func main() {
	interactive = (len(os.Args) <= 1)

	ext = NewUnbound(NewBuiltin(external, Null, Null, Null, nil))

	e, s := NewEnv(nil), NewScope(nil, nil)

	task0 = NewTask(psEvalBlock, Cons(nil, Null), e, s, nil)

	e.Add(NewSymbol("False"), False)
	e.Add(NewSymbol("True"), True)

	e.Add(NewSymbol("$stdin"), NewPipe(task0, os.Stdin, nil))
	e.Add(NewSymbol("$stdout"), NewPipe(task0, nil, os.Stdout))
	e.Add(NewSymbol("$stderr"), NewPipe(task0, nil, os.Stderr))

	if wd, err := os.Getwd(); err == nil {
		e.Add(NewSymbol("$cwd"), NewSymbol(wd))
	}

	s.DefineSyntax("block", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical, psEvalBlock)

		t.NewBlock(t.Dynamic, t.Lexical)

		return true
	})
	s.DefineSyntax("builtin", func(t *Task, args Cell) bool {
		return combiner(t, NewBuiltin)
	})
	s.DefineSyntax("define", func(t *Task, args Cell) bool {
		return lexical(t, psExecDefine)
	})
	s.DefineSyntax("dynamic", func(t *Task, args Cell) bool {
		return dynamic(t, psExecDynamic)
	})
	s.DefineSyntax("if", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical,
			psExecIf, SaveCode, psEvalElement)

		t.NewBlock(t.Dynamic, t.Lexical)

		t.Code = Car(t.Code)
		t.Scratch = Cdr(t.Scratch)

		return true
	})
	s.DefineSyntax("method", func(t *Task, args Cell) bool {
		return combiner(t, NewMethod)
	})
	s.DefineSyntax("set", func(t *Task, args Cell) bool {
		t.Scratch = Cdr(t.Scratch)

		s := Cadr(t.Code)
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
	s.DefineSyntax("setenv", func(t *Task, args Cell) bool {
		return dynamic(t, psExecSetenv)
	})
	s.DefineSyntax("spawn", func(t *Task, args Cell) bool {
		child := NewTask(psEvalBlock, t.Code, NewEnv(t.Dynamic),
			NewScope(t.Lexical, nil), t)

		go launch(child)

		SetCar(t.Scratch, child)

		return false
	})
	s.DefineSyntax("splice", func(t *Task, args Cell) bool {
		t.ReplaceStates(psExecSplice, psEvalElement)

		t.Code = Car(t.Code)
		t.Scratch = Cdr(t.Scratch)

		return true
	})
	s.DefineSyntax("syntax", func(t *Task, args Cell) bool {
		return combiner(t, NewSyntax)
	})
	s.DefineSyntax("while", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveDynamic|SaveLexical, psExecWhileTest)

		return true
	})

	s.PublicSyntax("public", func(t *Task, args Cell) bool {
		return lexical(t, psExecPublic)
	})

	/* Builtins. */
	s.DefineBuiltin("cd", func(t *Task, args Cell) bool {
		err := os.Chdir(Raw(Car(args)))
		status := 0
		if err != nil {
			status = 1
		}

		if wd, err := os.Getwd(); err == nil {
			t.Dynamic.Add(NewSymbol("$cwd"), NewSymbol(wd))
		}

		return t.Return(NewStatus(int64(status)))
	})
	s.DefineBuiltin("debug", func(t *Task, args Cell) bool {
		debug(t, "debug")

		return false
	})
	s.DefineBuiltin("fg", func(t *Task, args Cell) bool {
		if !interactive || t != task0 {
			return false
		}

		index := 0
		if args != Null {
			if a, ok := Car(args).(Atom); ok {
				index = int(a.Int())
			}
		}

		found := task0
		for k, v := range jobs {
			if args == Null {
				if v > index {
					index = v
				}

			}
			if v == index {
				found = k
			}
		}

		if found == task0 {
			return false
		}

		delete(jobs, found)
		delete(lines, found)

		task0 = found

		t.Stop()
		task0.Continue()

		return true
	})
	s.DefineBuiltin("jobs", func(t *Task, args Cell) bool {
		if !interactive || t != task0 {
			return false
		}

		m := make(map[int]*Task)
		i := make([]int, 0, len(jobs))
		for k, v := range jobs {
			i = append(i, v)
			m[v] = k
		}
		sort.Ints(i)
		for k, v := range i {
			if k != len(jobs)-1 {
				fmt.Printf("[%d] \t%s\n", v, lines[m[v]])
			} else {
				fmt.Printf("[%d]+\t%s\n", v, lines[m[v]])
			}
		}
		return false
	})
	s.DefineBuiltin("module", func(t *Task, args Cell) bool {
		str, err := module(Raw(Car(args)))

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

	s.PublicMethod("child", func(t *Task, args Cell) bool {
		o := Car(t.Scratch).(Binding).Self().Expose()

		return t.Return(NewObject(NewScope(o, nil)))
	})
	s.PublicMethod("clone", func(t *Task, args Cell) bool {
		o := Car(t.Scratch).(Binding).Self().Expose()

		return t.Return(NewObject(o.Copy()))
	})
	s.PublicMethod("exists", func(t *Task, args Cell) bool {
		l := Car(t.Scratch).(Binding).Self()
		c := Resolve(l, t.Dynamic, NewSymbol(Raw(Car(args))))

		return t.Return(NewBoolean(c != nil))
	})
	s.DefineMethod("exit", func(t *Task, args Cell) bool {
		t.Scratch = List(Car(args))

		t.Stop()

		return true
	})
	s.PublicMethod("unset", func(t *Task, args Cell) bool {
		l := Car(t.Scratch).(Binding).Self()
		r := l.Remove(NewSymbol(Raw(Car(args))))

		return t.Return(NewBoolean(r))
	})

	s.DefineMethod("append", func(t *Task, args Cell) bool {
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
	s.DefineMethod("car", func(t *Task, args Cell) bool {
		return t.Return(Caar(args))
	})
	s.DefineMethod("cdr", func(t *Task, args Cell) bool {
		return t.Return(Cdar(args))
	})
	s.DefineMethod("cons", func(t *Task, args Cell) bool {
		return t.Return(Cons(Car(args), Cadr(args)))
	})
	s.PublicMethod("eval", func(t *Task, args Cell) bool {
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
	s.DefineMethod("length", func(t *Task, args Cell) bool {
		var l int64 = 0

		switch c := Car(args); c.(type) {
		case *String, *Symbol:
			l = int64(len(Raw(c)))
		default:
			l = Length(c)
		}

		return t.Return(NewInteger(l))
	})
	s.DefineMethod("list", func(t *Task, args Cell) bool {
		return t.Return(args)
	})
	s.DefineMethod("open", func(t *Task, args Cell) bool {
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

		return t.Return(NewPipe(t, r, w))
	})
	s.DefineMethod("reverse", func(t *Task, args Cell) bool {
		return t.Return(Reverse(Car(args)))
	})
	s.DefineMethod("set-car", func(t *Task, args Cell) bool {
		SetCar(Car(args), Cadr(args))

		return t.Return(Cadr(args))
	})
	s.DefineMethod("set-cdr", func(t *Task, args Cell) bool {
		SetCdr(Car(args), Cadr(args))

		return t.Return(Cadr(args))
	})
	s.DefineMethod("wait", func(t *Task, args Cell) bool {
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

	/* Predicates. */
	s.DefineMethod("is-atom", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(IsAtom(Car(args))))
	})
	s.DefineMethod("is-boolean", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Boolean)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-builtin", func(t *Task, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Builtin)
		}

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-channel", func(t *Task, args Cell) bool {
		_, ok := GetConduit(Car(args).(Context)).(*Channel)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-cons", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(IsCons(Car(args))))
	})
	s.DefineMethod("is-float", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Float)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-integer", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Integer)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-method", func(t *Task, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Method)
		}

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-null", func(t *Task, args Cell) bool {
		ok := Car(args) == Null

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-number", func(t *Task, args Cell) bool {
		_, ok := Car(args).(Number)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-object", func(t *Task, args Cell) bool {
		_, ok := Car(args).(Context)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-pipe", func(t *Task, args Cell) bool {
		_, ok := GetConduit(Car(args).(Context)).(*Pipe)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-status", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Status)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-string", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*String)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-symbol", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Symbol)

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("is-syntax", func(t *Task, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Syntax)
		}

		return t.Return(NewBoolean(ok))
	})

	/* Generators. */
	s.DefineMethod("boolean", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(Car(args).Bool()))
	})
	s.DefineMethod("channel", func(t *Task, args Cell) bool {
		cap := 0
		if args != Null {
			cap = int(Car(args).(Atom).Int())
		}

		return t.Return(NewChannel(t, cap))
	})
	s.DefineMethod("float", func(t *Task, args Cell) bool {
		return t.Return(NewFloat(Car(args).(Atom).Float()))
	})
	s.DefineMethod("integer", func(t *Task, args Cell) bool {
		return t.Return(NewInteger(Car(args).(Atom).Int()))
	})
	s.DefineMethod("pipe", func(t *Task, args Cell) bool {
		return t.Return(NewPipe(t, nil, nil))
	})
	s.DefineMethod("status", func(t *Task, args Cell) bool {
		return t.Return(NewStatus(Car(args).(Atom).Status()))
	})
	s.DefineMethod("string", func(t *Task, args Cell) bool {
		return t.Return(NewString(Car(args).String()))
	})
	s.DefineMethod("symbol", func(t *Task, args Cell) bool {
		return t.Return(NewSymbol(Raw(Car(args))))
	})

	/* Relational. */
	s.DefineMethod("eq", func(t *Task, args Cell) bool {
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
	s.DefineMethod("ge", func(t *Task, args Cell) bool {
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
	s.DefineMethod("gt", func(t *Task, args Cell) bool {
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
	s.DefineMethod("is", func(t *Task, args Cell) bool {
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
	s.DefineMethod("le", func(t *Task, args Cell) bool {
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
	s.DefineMethod("lt", func(t *Task, args Cell) bool {
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
	s.DefineMethod("match", func(t *Task, args Cell) bool {
		pattern := Raw(Car(args))
		text := Raw(Cadr(args))

		ok, err := path.Match(pattern, text)
		if err != nil {
			panic(err)
		}

		return t.Return(NewBoolean(ok))
	})
	s.DefineMethod("ne", func(t *Task, args Cell) bool {
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
	s.DefineMethod("not", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(!Car(args).Bool()))
	})

	/* Arithmetic. */
	s.DefineMethod("add", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Add(Car(args))

		}

		return t.Return(acc)
	})
	s.DefineMethod("sub", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Subtract(Car(args))
		}

		return t.Return(acc)
	})
	s.DefineMethod("div", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Divide(Car(args))
		}

		return t.Return(acc)
	})
	s.DefineMethod("mod", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Modulo(Car(args))
		}

		return t.Return(acc)
	})
	s.DefineMethod("mul", func(t *Task, args Cell) bool {
		acc := Car(args).(Number)

		for Cdr(args) != Null {
			args = Cdr(args)
			acc = acc.Multiply(Car(args))
		}

		return t.Return(acc)
	})

	/* Standard namespaces. */
	list := NewObject(NewScope(s, nil))
	s.Define(NewSymbol("List"), list)

	list.PublicMethod("to-string", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewString(s))
	})
	list.PublicMethod("to-symbol", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewSymbol(s))
	})

	text := NewObject(NewScope(s, nil))
	s.Define(NewSymbol("Text"), text)

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
			arr[i] = string(Raw(Car(list)))
			list = Cdr(list)
		}

		r := strings.Join(arr, string(Raw(sep)))

		if str {
			return t.Return(NewString(r))
		}
		return t.Return(NewSymbol(r))
	})
	text.PublicMethod("split", func(t *Task, args Cell) bool {
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

		return t.Return(r)
	})
	text.PublicMethod("sprintf", func(t *Task, args Cell) bool {
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

		return t.Return(NewString(s))
	})
	text.PublicMethod("substring", func(t *Task, args Cell) bool {
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

		return t.Return(r)
	})
	text.PublicMethod("to-list", func(t *Task, args Cell) bool {
		l := Null
		for _, char := range Raw(Car(args)) {
			l = Cons(NewInteger(int64(char)), l)
		}

		return t.Return(Reverse(l))
	})
	text.PublicMethod("lower", func(t *Task, args Cell) bool {
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

		return t.Return(r)
	})
	text.PublicMethod("title", func(t *Task, args Cell) bool {
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

		return t.Return(r)
	})
	text.PublicMethod("upper", func(t *Task, args Cell) bool {
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

		return t.Return(r)
	})

	s.Public(NewSymbol("Root"), s)

	pid := os.Getpid()
	e.Add(NewSymbol("$$"), NewInteger(int64(pid)))

	/* Command-line arguments */
	args := Null
	if len(os.Args) > 1 {
		e.Add(NewSymbol("$0"), NewSymbol(os.Args[1]))

		for i, v := range os.Args[2:] {
			e.Add(NewSymbol("$"+strconv.Itoa(i+1)), NewSymbol(v))
		}

		for i := len(os.Args) - 1; i > 1; i-- {
			args = Cons(NewSymbol(os.Args[i]), args)
		}
	} else {
		e.Add(NewSymbol("$0"), NewSymbol(os.Args[0]))
	}
	e.Add(NewSymbol("$args"), args)

	/* Environment variables. */
	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		e.Add(NewSymbol("$"+kv[0]), NewSymbol(kv[1]))
	}

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTSTP}
	done0 = make(chan Cell)
	eval0 = make(chan Cell)
	incoming := make(chan os.Signal, len(signals))
	signal.Notify(incoming, signals...)
	go func() {
		go listen(task0)
		var c Cell = nil
		for c == nil && task0.Stack != Null {
			for c == nil {
				select {
				case <-incoming:
					// Ignore signals.
				case c = <-eval0:
				}
			}
			task0.Eval <- c
			for c != nil {
				prev := task0
				select {
				case sig := <-incoming:
					// Handle signals.
					switch sig {
					case syscall.SIGTSTP:
						if !interactive {
							syscall.Kill(pid, syscall.SIGSTOP)
							continue
						}
						task0.Suspend()
						last := 0
						for _, v := range jobs {
							if v > last {
								last = v
							}
						}
						last++
						jobs[task0] = last
						lines[task0] = command

						fallthrough
					case syscall.SIGINT:
						if !interactive {
							os.Exit(130)
						}
						if sig == syscall.SIGINT {
							task0.Stop()
						}
						fmt.Printf("\n")
						task0 = NewTask(psEvalBlock, Cons(nil, Null), e, s, nil)
						go listen(task0)
						c = nil
					}

				case c = <-task0.Done:
					if task0 != prev {
						c = Null
						continue
					}
				}
			}
			done0 <- c
		}
		os.Exit(status(Car(task0.Scratch)))
	}()

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
	block {
            dynamic $stdin p
            e::eval right
            if close: p::reader-close
	}
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
    spawn {
        dynamic $stdout p
        e::eval cmd
        p::writer-close
    }
    define r: cons '() '()
    define c r
    define l: p::readline
    while l {
	set-cdr c: cons l '()
	set c: cdr c
        set l: p::readline
    }
    p::reader-close
    return: cdr r
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
define is-list: method (l) as {
    if (is-null l): return False
    if (not: is-cons l): return False
    if (is-null: cdr l): return True
    is-list: cdr l
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
	define paths '()
	define name basename

	if (exists $OHPATH): set paths: Text::split ":" $OHPATH
	while (and (not: is-null paths) (not: test -r name)) {
		set name: Text::join / (car paths) basename
		set paths: cdr paths
	}

	if (not: test -r name): set name basename

        define r: cons '() '()
        define c r
	define f: open name "r-"
	define l: f::read
	while l {
		set-cdr c: cons l '()
		set c: cdr c
		set l: f::read
	}
	set c: cdr r
	f::close
	define eval-list: method (rval first rest) as {
		if (is-null first): return rval
		eval-list (e::eval first) (car rest) (cdr rest)
	}
	eval-list (status 0) (car c) (cdr c)
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
`)), evaluate)

	if len(os.Args) <= 1 {
		cli = &Liner{liner.NewLiner()}
		cli.SetCompleter(complete)

		Parse(cli, evaluate)

		cli.Close()
		fmt.Printf("\n")
	} else {
		evaluate(List(NewSymbol("source"), NewString(os.Args[1])))
	}

	os.Exit(0)
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
	return GetConduit(c.(Context)).(*Pipe).ReadFd()
}

func run(t *Task, end Cell) (successful bool) {
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
				args = expand(args)
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
					Raw(Car(t.Code)) != "else" {
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
				t.Scratch = Cons(ext, t.Scratch)

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
		case psEvalElement, psEvalElementBuiltin:
			if t.Code == Null {
				t.Scratch = Cons(t.Code, t.Scratch)
				break
			} else if IsCons(t.Code) {
				if IsAtom(Cdr(t.Code)) {
					t.ReplaceStates(SaveDynamic|SaveLexical,
						psEvalElement,
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
				ok, msg := lookup(t, sym, simple)
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
				s := Raw(v)
				os.Setenv(strings.TrimLeft(k.String(), "$"), s)
			}

			t.Dynamic.Add(k, v)

		case psExecSet:
			k := t.Code.(*Symbol)
			r := Resolve(t.Lexical, t.Dynamic, k)
			if r == nil {
				panic("'" + k.String() + "' is not defined")
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

func status(c Cell) int {
	a, ok := c.(Atom)
	if !ok {
		return 0
	}
	return int(a.Status())
}

func strict(t *Task) (ok bool) {
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

func wpipe(c Cell) *os.File {
	return GetConduit(c.(Context)).(*Pipe).WriteFd()
}
