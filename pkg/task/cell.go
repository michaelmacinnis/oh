// Released under an MIT-style license. See LICENSE.

package task

import (
	"bufio"
	"fmt"
	"github.com/michaelmacinnis/adapted"
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/peterh/liner"
	"math/big"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Conduit interface {
	Context

	Close()
	ReaderClose()
	ReadLine(*Task) Cell
	Read(*Task) Cell
	WriterClose()
	Write(c Cell)
}

/* Convert Context into a Conduit. (Return nil if not possible). */
func asConduit(o Context) Conduit {
	if c, ok := o.(Conduit); ok {
		return c
	}

	return nil
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

	conduit := asConduit(context)
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

func IsContinuation(c Cell) bool {
	switch c.(type) {
	case *Continuation:
		return true
	}
	return false
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
	Command string
	Group   int
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

	conduit := asConduit(context)
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
			parse(t, p.reader(), deref, func(c Cell) {
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

func NewString(t *Task, v string) *String {
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
	return adapted.Quote(s.v)
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

	if t.GetState() == psExecSyntax {
		t.ReplaceStates(SaveLexical, psEvalBlock)
		t.Lexical = NewScope(m.Ref().Scope(), nil)
	} else {
		t.ReplaceStates(SaveDynamic|SaveLexical, psEvalBlock)
		t.NewBlock(t.Dynamic, m.Ref().Scope())
	}

	t.Code = m.Ref().Body()

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

	if jobControlEnabled() {
		attr.Sys = SysProcAttr(t.Group)
	}

	proc, err := os.StartProcess(arg0, argv, attr)
	if err != nil {
		t.Unlock()
		return nil, err
	}

	if jobControlEnabled() {
		if t.Group == 0 {
			t.Group = proc.Pid
		}
	}

	t.pid = proc.Pid

	t.Unlock()

	status := JoinProcess(proc)

	if jobControlEnabled() {
		if t.Group == t.pid {
			t.Group = 0
		}
	}
	t.pid = 0

	return NewStatus(int64(status)), err
}

func (t *Task) External(args Cell) bool {
	t.Scratch = Cdr(t.Scratch)

	arg0, problem := adapted.LookPath(raw(Car(t.Scratch)))

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

	l := t.Self().Expose()
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
	} else if simple && !isSimple(c.Get()) {
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
				t.Scratch = Cons(l, t.Scratch)
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

func (t *Task) Self() Context {
	return Car(t.Scratch).(Binding).Self()
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

