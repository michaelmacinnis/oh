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

//#include <signal.h>
//void ignore(void) {
//	signal(SIGTTOU, SIG_IGN);
//	signal(SIGTTIN, SIG_IGN);
//}
import "C"

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode"
	"unsafe"
)

var ext Cell
var group int
var pid int

var jobs = map[int]*Task{}

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

func init() {
	pid = syscall.Getpid()
	pgrp := syscall.Getpgrp()
	if pid != pgrp {
		syscall.Setpgid(0, 0)
	}
	group = pid
	C.ignore()
}

func main() {
	StartBroker(pid, env0, scope0)

	env0.Add(NewSymbol("False"), False)
	env0.Add(NewSymbol("True"), True)

	/* Command-line arguments */
	args := Null
	if !interactive {
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

	env0.Add(NewSymbol("$$"), NewInteger(int64(pid)))

	if wd, err := os.Getwd(); err == nil {
		env0.Add(NewSymbol("$cwd"), NewSymbol(wd))
	}

	fg := ForegroundTask()
	env0.Add(NewSymbol("$stdin"), NewPipe(fg, os.Stdin, nil))
	env0.Add(NewSymbol("$stdout"), NewPipe(fg, nil, os.Stdout))
	env0.Add(NewSymbol("$stderr"), NewPipe(fg, nil, os.Stderr))

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
	scope0.DefineSyntax("setenv", func(t *Task, args Cell) bool {
		return t.DynamicVar(psExecSetenv)
	})
	scope0.DefineSyntax("spawn", func(t *Task, args Cell) bool {
		child := NewTask(psEvalBlock, t.Code, NewEnv(t.Dynamic),
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
	scope0.DefineBuiltin("cd", func(t *Task, args Cell) bool {
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
	scope0.DefineBuiltin("debug", func(t *Task, args Cell) bool {
		t.Debug("debug")

		return false
	})
	scope0.DefineBuiltin("fg", func(t *Task, args Cell) bool {
		if !interactive || t != ForegroundTask() {
			return false
		}

		index := 0
		if args != Null {
			if a, ok := Car(args).(Atom); ok {
				index = int(a.Int())
			}
		} else {
			for k, _ := range jobs {
				if k > index {
					index = k
				}
			}
		}

		found, ok := jobs[index]

		if !ok {
			return false
		}

		t.Stop()

		if found.Job.group != 0 {
			foreground := found.Job.group
			syscall.Syscall(syscall.SYS_IOCTL,
				uintptr(syscall.Stdin),
				syscall.TIOCSPGRP,
				uintptr(unsafe.Pointer(&foreground)))
			found.Job.mode.ApplyMode()
		}

		SetForegroundTask(found)

		delete(jobs, index)

		return true
	})
	scope0.DefineBuiltin("jobs", func(t *Task, args Cell) bool {
		if !interactive || t != ForegroundTask() {
			return false
		}

		i := make([]int, 0, len(jobs))
		for k, _ := range jobs {
			i = append(i, k)
		}
		sort.Ints(i)
		for k, v := range i {
			if k != len(jobs)-1 {
				fmt.Printf("[%d] \t%s\n", v, jobs[v].Job.command)
			} else {
				fmt.Printf("[%d]+\t%s\n", v, jobs[v].Job.command)
			}
		}
		return false
	})
	scope0.DefineBuiltin("module", func(t *Task, args Cell) bool {
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
	scope0.DefineBuiltin("run", func(t *Task, args Cell) bool {
		if args == Null {
			SetCar(t.Scratch, False)
			return false
		}
		SetCar(t.Scratch, Car(args))
		t.Scratch = Cons(ext, t.Scratch)
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
	scope0.PublicMethod("exists", func(t *Task, args Cell) bool {
		l := Car(t.Scratch).(Binding).Self()
		c := Resolve(l, t.Dynamic, NewSymbol(Raw(Car(args))))

		return t.Return(NewBoolean(c != nil))
	})
	scope0.DefineMethod("exit", func(t *Task, args Cell) bool {
		t.Scratch = List(Car(args))

		t.Stop()

		return true
	})
	scope0.PublicMethod("unset", func(t *Task, args Cell) bool {
		l := Car(t.Scratch).(Binding).Self()
		r := l.Remove(NewSymbol(Raw(Car(args))))

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
		var l int64 = 0

		switch c := Car(args); c.(type) {
		case *String, *Symbol:
			l = int64(len(Raw(c)))
		default:
			l = Length(c)
		}

		return t.Return(NewInteger(l))
	})
	scope0.DefineMethod("list", func(t *Task, args Cell) bool {
		return t.Return(args)
	})
	scope0.DefineMethod("open", func(t *Task, args Cell) bool {
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

	/* Predicates. */
	scope0.DefineMethod("is-atom", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(IsAtom(Car(args))))
	})
	scope0.DefineMethod("is-boolean", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Boolean)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-builtin", func(t *Task, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Builtin)
		}

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-channel", func(t *Task, args Cell) bool {
		_, ok := GetConduit(Car(args).(Context)).(*Channel)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-cons", func(t *Task, args Cell) bool {
		return t.Return(NewBoolean(IsCons(Car(args))))
	})
	scope0.DefineMethod("is-float", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Float)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-integer", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Integer)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-method", func(t *Task, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Method)
		}

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-null", func(t *Task, args Cell) bool {
		ok := Car(args) == Null

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-number", func(t *Task, args Cell) bool {
		_, ok := Car(args).(Number)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-object", func(t *Task, args Cell) bool {
		_, ok := Car(args).(Context)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-pipe", func(t *Task, args Cell) bool {
		_, ok := GetConduit(Car(args).(Context)).(*Pipe)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-status", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Status)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-string", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*String)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-symbol", func(t *Task, args Cell) bool {
		_, ok := Car(args).(*Symbol)

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("is-syntax", func(t *Task, args Cell) bool {
		b, ok := Car(args).(Binding)
		if ok {
			_, ok = b.Ref().(*Syntax)
		}

		return t.Return(NewBoolean(ok))
	})

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
		return t.Return(NewPipe(t, nil, nil))
	})
	scope0.DefineMethod("status", func(t *Task, args Cell) bool {
		return t.Return(NewStatus(Car(args).(Atom).Status()))
	})
	scope0.DefineMethod("string", func(t *Task, args Cell) bool {
		return t.Return(NewString(Car(args).String()))
	})
	scope0.DefineMethod("symbol", func(t *Task, args Cell) bool {
		return t.Return(NewSymbol(Raw(Car(args))))
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
		pattern := Raw(Car(args))
		text := Raw(Cadr(args))

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

	/* Standard namespaces. */
	list := NewObject(NewScope(scope0, nil))
	scope0.Define(NewSymbol("List"), list)

	list.PublicMethod("to-string", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewRawString(s))
	})
	list.PublicMethod("to-symbol", func(t *Task, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

		return t.Return(NewSymbol(s))
	})

	text := NewObject(NewScope(scope0, nil))
	scope0.Define(NewSymbol("Text"), text)

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

	scope0.Public(NewSymbol("Root"), scope0)

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
`)), Evaluate)

	StartInterface()
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

func status(c Cell) int {
	a, ok := c.(Atom)
	if !ok {
		return 0
	}
	return int(a.Status())
}

func wpipe(c Cell) *os.File {
	return GetConduit(c.(Context)).(*Pipe).WriteFd()
}
