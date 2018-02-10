// Released under an MIT license. See LICENSE.

package task

import (
	"bufio"
	"fmt"
	"github.com/michaelmacinnis/adapted"
	"github.com/michaelmacinnis/oh/pkg/boot"
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/parser"
	"github.com/michaelmacinnis/oh/pkg/system"
	"github.com/peterh/liner"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type binding interface {
	Cell

	bind(c Cell) binding
	ref() closure
	self() Cell
}

type closure interface {
	Cell

	Applier() function
	Body() Cell
	CallerLabel() Cell
	Params() Cell
	Scope() context
	SelfLabel() Cell
}

type closurer func(a function, b, c, o, p Cell, s context) closure

type context interface {
	Cell

	Access(key string) Reference
	Copy() context
	Complete(word string) []string
	Define(key string, value Cell)
	Exported() map[string]Cell
	Expose() context
	Faces() *Env
	Prev() context
	Public(key string, value Cell)
	Visibility() *Env

	DefineBuiltin(k string, f function)
	DefineMethod(k string, f function)
	DefineSyntax(k string, f function)
	PublicMethod(k string, f function)
	PublicSyntax(k string, f function)
}

type function func(t *Task, args Cell) bool

type validator func(c Cell) bool

const (
	svCarCode = 1 << iota
	svCdrCode
	svDump
	svFrame
	svLexical
	svMax

	svCode = svCarCode | svCdrCode
)

const (
	psChangeContext = svMax + iota

	psEvalArguments
	psEvalBlock
	psEvalCommand
	psEvalElement
	psEvalHead
	psEvalMember

	psExecBuiltin
	psExecCommand
	psExecDefine
	psExecIf
	psExecMethod
	psExecPublic
	psExecSet
	psExecSplice
	psExecSyntax
	psExecWhileBody
	psExecWhileTest

	psFatal
	psNoOp
	psReturn
)

var (
	envc        context
	envp        context
	envs        context
	frame0      Cell
	external    Cell
	interactive = false
	jobs        = map[int]*Task{}
	jobsl       = &sync.RWMutex{}
	namespace   context
	scope0      *scope
	sys         context
	task0       *Task
	task0l      = &sync.RWMutex{}
	taskc       *Task
)

/* Bound cell definition. */

type bound struct {
	r closure
	s Cell
}

func NewBound(c closure, self Cell) *bound {
	return &bound{c, self}
}

func (b *bound) Bool() bool {
	return true
}

func (b *bound) Equal(c Cell) bool {
	if m, ok := c.(*bound); ok {
		return b.r == m.ref() && b.s == m.self()
	}
	return false
}

func (b *bound) String() string {
	return fmt.Sprintf("%%bound %p%%", b)
}

/* Bound-specific functions */

func (b *bound) bind(c Cell) binding {
	if c == b.s {
		return b
	}
	return NewBound(b.r, c)
}

func (b *bound) ref() closure {
	return b.r
}

func (b *bound) self() Cell {
	return b.s
}

/* Builtin cell definition. */

type builtin struct {
	command
}

func IsBuiltin(c Cell) bool {
	b, ok := c.(binding)
	if !ok {
		return false
	}

	switch b.ref().(type) {
	case *builtin:
		return true
	}
	return false
}

func NewBuiltin(a function, b, c, o, p Cell, s context) closure {
	return &builtin{
		command{
			applier: a,
			body:    b,
			clabel:  c,
			olabel:  o,
			params:  p,
			scope:   s,
		},
	}
}

func NewUnboundBuiltin(a function, s context) binding {
	return NewUnbound(NewBuiltin(a, Null, Null, Null, Null, s))
}

func (b *builtin) Equal(c Cell) bool {
	return b == c
}

func (b *builtin) String() string {
	return fmt.Sprintf("%%builtin %p%%", b)
}

/* Command cell definition. */

type command struct {
	applier function
	body    Cell
	clabel  Cell
	olabel  Cell
	params  Cell
	scope   context
}

func (c *command) Bool() bool {
	return true
}

func (c *command) Applier() function {
	return c.applier
}

func (c *command) Body() Cell {
	return c.body
}

func (c *command) CallerLabel() Cell {
	return c.clabel
}

func (c *command) Params() Cell {
	return c.params
}

func (c *command) Scope() context {
	return c.scope
}

func (c *command) SelfLabel() Cell {
	return c.olabel
}

/* Job definition. */

type Job struct {
	sync.Mutex
	command string
	group   int
	mode    liner.ModeApplier
	pids    map[int]struct{}
}

func NewJob() *Job {
	mode, _ := liner.TerminalMode()
	return &Job{sync.Mutex{}, "", 0, mode, map[int]struct{}{}}
}

func (j *Job) SetCommand(cmd string) {
	j.Lock()
	defer j.Unlock()

	if j.command != "" {
		return
	}
	j.command = cmd
}

func (j *Job) assignedGroup() int {
	if !jobControlEnabled() {
		return 0
	}

	j.Lock()
	return j.group
}

func (j *Job) commandAndGroup() (string, int) {
	j.Lock()
	defer j.Unlock()

	return j.command, j.group
}

func (j *Job) isForegroundJob(pid int) bool {
	j.Lock()
	defer j.Unlock()

	return j.group == pid
}

func (j *Job) moveToForeground() {
	j.Lock()
	defer j.Unlock()

	if j.group != 0 {
		system.SetForegroundGroup(j.group)
		j.mode.ApplyMode()
	}
}

func (j *Job) registerPid(pid int) {
	if !jobControlEnabled() {
		return
	}

	defer j.Unlock()

	j.pids[pid] = struct{}{}
	if j.group == 0 {
		j.group = pid
	}
}

func (j *Job) resetCommandAndGroup() {
	j.Lock()
	defer j.Unlock()

	j.command = ""
	j.group = 0
}

func (j *Job) saveMode() {
	j.Lock()
	defer j.Unlock()

	mode, _ := liner.TerminalMode()
	j.mode = mode
}

func (j *Job) unregisterPid(pid int) {
	if !jobControlEnabled() {
		return
	}

	j.Lock()
	defer j.Unlock()

	delete(j.pids, pid)
	if len(j.pids) == 0 {
		j.group = 0
	}
}

/* Method cell definition. */

type method struct {
	command
}

func IsMethod(c Cell) bool {
	b, ok := c.(binding)
	if !ok {
		return false
	}

	switch b.ref().(type) {
	case *method:
		return true
	}
	return false
}

func NewMethod(a function, b, c, o, p Cell, s context) closure {
	return &method{
		command{
			applier: a,
			body:    b,
			clabel:  c,
			olabel:  o,
			params:  p,
			scope:   s,
		},
	}
}

func NewBoundMethod(a function, s context) binding {
	return NewBound(NewMethod(a, Null, Null, Null, Null, s), s)
}

func (m *method) Equal(c Cell) bool {
	return m == c
}

func (m *method) String() string {
	return fmt.Sprintf("%%method %p%%", m)
}

/*
 * Object cell definition.
 * (An object cell only allows access to a context's public members).
 */

type object struct {
	context
}

func NewObject(v context) *object {
	return &object{v.Expose()}
}

func (o *object) Equal(c Cell) bool {
	if o == c {
		return true
	}
	if o, ok := c.(*object); ok {
		return o.context == o.Expose()
	}
	return false
}

func (o *object) String() string {
	return fmt.Sprintf("%%object %p%%", o)
}

/* Object-specific functions */

func (o *object) Access(key string) Reference {
	var obj context
	for obj = o; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().Prev().Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (o *object) Complete(word string) []string {
	cl := []string{}

	var obj context
	for obj = o; obj != nil; obj = obj.Prev() {
		cl = append(cl, obj.Faces().Prev().Complete(false, word)...)
	}

	return cl
}

func (o *object) Copy() context {
	return &object{
		&scope{o.Expose().Faces().Copy(), o.context.Prev()},
	}
}

func (o *object) Expose() context {
	return o.context
}

func (o *object) Define(key string, value Cell) {
	panic("private members cannot be added to an object")
}

/* Registers cell definition. */

type registers struct {
	Continuation // Stack and Dump

	Code    Cell // Control
	Lexical Cell
}

/* Registers-specific functions. */

func (r *registers) Arguments() Cell {
	e := Car(r.Dump)
	l := Null

	for e != nil {
		l = Cons(e, l)

		r.Dump = Cdr(r.Dump)
		e = Car(r.Dump)
	}

	r.Dump = Cdr(r.Dump)

	return l
}

func (r *registers) Complete(first string, word string) (cmpltns []string) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		cmpltns = []string{word}
	}()

	prefix, name, suffix := extractName(false, word)

	if first != "" && prefix == "" {
		return []string{}
	}

	cl := toContext(r.Lexical).Complete(name)
	for f := r.Frame; f != Null; f = Cdr(f) {
		o := Car(f).(context)
		cl = append(cl, o.Complete(name)...)
	}

	for k, v := range cl {
		cl[k] = prefix + v + suffix
	}

	return cl
}

func (r *registers) CurrentContinuation() *Continuation {
	cc := r.Continuation
	cc.Dump = Cdr(cc.Dump)
	return &cc
}

func (r *registers) GetState() int64 {
	if r.Stack == Null {
		return 0
	}
	return Car(r.Stack).(Atom).Int()
}

func (r *registers) MakeEnv() []string {
	e := toContext(r.Lexical).Exported()

	for f := r.Frame; f != Null; f = Cdr(f) {
		o := Car(f).(context)
		for k, v := range o.Exported() {
			if _, ok := e[k]; !ok {
				e[k] = v
			}
		}
	}

	l := make([]string, 0, len(e))

	for k, v := range e {
		l = append(l, k+"="+Raw(v))
	}

	return l
}

func (r *registers) NewBlock(lexical context) {
	r.Lexical = NewScope(lexical, nil)
}

func (r *registers) NewFrame(lexical context) {
	state := int64(svLexical)

	c := toContext(r.Lexical)
	v := c.Visibility()
	if v != nil && v != Car(r.Frame).(context).Visibility() {
		state |= svFrame
	}

	r.ReplaceStates(state, psEvalBlock)

	if state&svFrame > 0 {
		r.Frame = Cons(NewObject(c), r.Frame)
	}

	r.Lexical = NewScope(lexical, nil)
}

func (r *registers) NewStates(l ...int64) {
	for _, f := range l {
		if f >= svMax {
			r.Stack = Cons(NewInteger(f), r.Stack)
			continue
		}

		p := *r

		s := r.GetState()
		if s < svMax && f < svMax {
			// Previous and current states are save states.
			c := f & s
			if f&svCode > 0 || s&svCode > 0 {
				c |= svCode
			}
			if c&f == f {
				// Nothing new to save.
				continue
			} else if c&s == s {
				// Previous save state is a subset.
				p.RestoreState()
				r.Stack = p.Stack
				if c&svCode > 0 {
					f |= svCode
				}
			}
		}

		if f&svCode > 0 {
			if f&svCode == svCode {
				r.Stack = Cons(p.Code, r.Stack)
			} else if f&svCarCode > 0 {
				r.Stack = Cons(Car(p.Code), r.Stack)
			} else if f&svCdrCode > 0 {
				r.Stack = Cons(Cdr(p.Code), r.Stack)
			}
		}

		if f&svDump > 0 {
			r.Stack = Cons(p.Dump, r.Stack)
		}

		if f&svFrame > 0 {
			r.Stack = Cons(p.Frame, r.Stack)
		}

		if f&svLexical > 0 {
			r.Stack = Cons(p.Lexical, r.Stack)
		}

		r.Stack = Cons(NewInteger(f), r.Stack)
	}
}

func (r *registers) RemoveState() (s int64) {
	s = r.GetState()

	r.Stack = Cdr(r.Stack)
	if s >= svMax {
		return
	}

	if s&svLexical > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if s&svFrame > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if s&svDump > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if s&svCode > 0 {
		r.Stack = Cdr(r.Stack)
	}

	return
}

func (r *registers) ReplaceStates(l ...int64) {
	r.RemoveState()
	r.NewStates(l...)
}

func (r *registers) RestoreState() {
	s := r.GetState()

	if s == 0 || s >= svMax {
		return
	}

	if s&svLexical > 0 {
		r.Stack = Cdr(r.Stack)
		r.Lexical = Car(r.Stack).(context)
	}

	if s&svFrame > 0 {
		r.Stack = Cdr(r.Stack)
		r.Frame = Car(r.Stack)
	}

	if s&svDump > 0 {
		r.Stack = Cdr(r.Stack)
		r.Dump = Car(r.Stack)
	}

	if s&svCode > 0 {
		r.Stack = Cdr(r.Stack)
		r.Code = Car(r.Stack)
	}

	r.Stack = Cdr(r.Stack)
}

func (r *registers) Return(rv Cell) bool {
	SetCar(r.Dump, rv)

	return false
}

/*
 * Scope cell definition.
 * (A scope cell allows access to a context's public and private members).
 */

type scope struct {
	env  *Env
	prev context
}

func NewScope(prev context, fixed *Env) *scope {
	return &scope{NewEnv(NewEnv(fixed)), prev}
}

func (s *scope) Bool() bool {
	return true
}

func (s *scope) Equal(c Cell) bool {
	return s == c
}

func (s *scope) String() string {
	return fmt.Sprintf("%%scope %p%%", s)
}

/* Scope-specific functions */

func (s *scope) Access(key string) Reference {
	var obj context
	for obj = s; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (s *scope) Complete(word string) []string {
	cl := []string{}

	var obj context
	for obj = s; obj != nil; obj = obj.Prev() {
		cl = append(cl, obj.Faces().Complete(false, word)...)
	}

	return cl
}

func (s *scope) Copy() context {
	return &scope{s.env.Copy(), s.prev}
}

func (s *scope) Exported() map[string]Cell {
	return s.env.Prev().Prefixed(true, "")
}

func (s *scope) Expose() context {
	return s
}

func (s *scope) Faces() *Env {
	return s.env
}

func (s *scope) Prev() context {
	return s.prev
}

func (s *scope) Define(key string, value Cell) {
	s.env.Add(key, value)
}

func (s *scope) Public(key string, value Cell) {
	s.env.Prev().Add(key, value)
}

func (s *scope) Visibility() *Env {
	var obj context
	for obj = s; obj != nil; obj = obj.Prev() {
		env := obj.Faces().Prev()
		if !env.Empty() {
			return env
		}
	}

	return nil
}

func (s *scope) DefineBuiltin(k string, a function) {
	s.Define(k, NewUnboundBuiltin(a, s))
}

func (s *scope) DefineMethod(k string, a function) {
	s.Define(k, NewBoundMethod(a, s))
}

func (s *scope) PublicMethod(k string, a function) {
	s.Public(k, NewBoundMethod(a, s))
}

func (s *scope) DefineSyntax(k string, a function) {
	s.Define(k, NewBoundSyntax(a, s))
}

func (s *scope) PublicSyntax(k string, a function) {
	s.Public(k, NewBoundSyntax(a, s))
}

/* Syntax cell definition. */

type syntax struct {
	command
}

func IsSyntax(c Cell) bool {
	b, ok := c.(binding)
	if !ok {
		return false
	}

	switch b.ref().(type) {
	case *syntax:
		return true
	}
	return false
}

func NewSyntax(a function, b, c, o, p Cell, s context) closure {
	return &syntax{
		command{
			applier: a,
			body:    b,
			clabel:  c,
			olabel:  o,
			params:  p,
			scope:   s,
		},
	}
}

func NewBoundSyntax(a function, s context) binding {
	return NewBound(NewSyntax(a, Null, Null, Null, Null, s), s)
}

func (m *syntax) Equal(c Cell) bool {
	return m == c
}

func (m *syntax) String() string {
	return fmt.Sprintf("%%syntax %p%%", m)
}

/* Task cell definition. */

type Task struct {
	*action
	sync.Mutex
	*Job
	registers
	Done      chan Cell
	Eval      chan Cell
	children  map[*Task]bool
	childrenl *sync.RWMutex
	parent    *Task
	pid       int
}

func NewTask(c Cell, l context, p *Task) *Task {
	if l == nil {
		l = scope0
	}

	var frame Cell
	var j *Job
	if p == nil {
		frame = frame0
		j = NewJob()
	} else {
		frame = p.Frame
		j = p.Job
	}

	t := &Task{
		action: NewAction(),
		Mutex:  sync.Mutex{},
		Job:    j,
		registers: registers{
			Continuation: Continuation{
				Dump:  List(ExitSuccess),
				Frame: frame,
				Stack: List(NewInteger(psEvalBlock)),
				File:  "oh",
				Line:  0,
			},
			Code:    c,
			Lexical: l,
		},
		Done:      make(chan Cell, 1),
		Eval:      make(chan Cell, 1),
		children:  make(map[*Task]bool),
		childrenl: &sync.RWMutex{},
		parent:    p,
	}

	if p != nil {
		p.childrenl.Lock()
		defer p.childrenl.Unlock()
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
	caller := t.Lexical

	m := Car(t.Dump).(binding)

	t.NewFrame(m.ref().Scope())

	t.Code = m.ref().Body()

	c := toContext(t.Lexical)

	clabel := m.ref().CallerLabel()
	if clabel != Null {
		c.Define(Raw(clabel), caller)
	}

	olabel := m.ref().SelfLabel()
	if olabel != Null {
		c.Define(Raw(olabel), toContext(m.self()).Expose())
	}

	params := m.ref().Params()
	for args != Null && params != Null && IsAtom(Car(params)) {
		c.Define(Raw(Car(params)), Car(args))
		args, params = Cdr(args), Cdr(params)
	}
	if IsPair(Car(params)) {
		c.Define(Raw(Caar(params)), args)
	}

	c.Define("return", t.CurrentContinuation())

	return true
}

func (t *Task) Chdir(dir string) bool {
	rv := ExitSuccess

	c, _ := Resolve(t.Lexical, t.Frame, "PWD")
	oldwd := c.Get().String()

	err := os.Chdir(dir)
	if err != nil {
		rv = ExitFailure
	} else if wd, err := os.Getwd(); err == nil {
		c := toContext(t.Lexical)
		c.Public("PWD", NewSymbol(wd))
		c.Public("OLDPWD", NewSymbol(oldwd))
	}

	return t.Return(rv)
}

func (t *Task) Closure(n closurer) bool {
	olabel := Car(t.Code)
	t.Code = Cdr(t.Code)

	params := olabel
	if IsSymbol(olabel) {
		params = Car(t.Code)
		t.Code = Cdr(t.Code)
	} else {
		olabel = Null
	}

	equals := Car(t.Code)
	t.Code = Cdr(t.Code)

	clabel := Null
	if Raw(equals) != "=" {
		clabel = equals
		equals = Car(t.Code)
		t.Code = Cdr(t.Code)
	}

	if Raw(equals) != "=" {
		panic(ErrSyntax + "expected '='")
	}

	body := t.Code
	scope := toContext(t.Lexical)

	c := n((*Task).Apply, body, clabel, olabel, params, scope)
	if olabel == Null {
		SetCar(t.Dump, NewUnbound(c))
	} else {
		SetCar(t.Dump, NewBound(c, scope))
	}

	return false
}

func (t *Task) Continue() {
	if t.pid > 0 {
		system.ContinueProcess(t.pid)
	}

	t.childrenl.RLock()
	defer t.childrenl.RUnlock()
	for k, v := range t.children {
		if v {
			k.Continue()
		}
	}

	t.action.Continue()
}

func (t *Task) debug(s string) {
	fmt.Printf("%s: t.Code = %v, t.Dump = %v\n", s, t.Code, t.Dump)
}

func (t *Task) execute(arg0 string, argv []string, attr *os.ProcAttr) (*Status, error) {
	t.Lock()
	defer t.Unlock()

	attr.Sys = system.SysProcAttr(t.Job.assignedGroup())

	proc, err := os.StartProcess(arg0, argv, attr)
	if err != nil {
		return nil, err
	}

	t.Job.registerPid(proc.Pid)
	t.pid = proc.Pid

	rv := status(proc)

	t.Job.unregisterPid(proc.Pid)
	t.pid = 0

	return rv, err
}

func (t *Task) External(args Cell) bool {
	t.Dump = Cdr(t.Dump)

	name := tildeExpand(t.Lexical, t.Frame, Raw(Car(t.Dump)))

	pathenv := ""
	c, _ := Resolve(t.Lexical, t.Frame, "PATH")
	if c != nil {
		pathenv = Raw(c.Get())
	}
	arg0, exe, problem := adapted.LookPath(pathenv, name)

	SetCar(t.Dump, False)

	if problem != nil {
		panic(ErrNotFound + problem.Error())
	}

	if !exe {
		return t.Chdir(arg0)
	}

	argv := []string{name}

	for ; args != Null; args = Cdr(args) {
		argv = append(argv, Raw(Car(args)))
	}

	c, _ = Resolve(t.Lexical, t.Frame, "PWD")
	dir := c.Get().String()

	c, _ = Resolve(t.Lexical, t.Frame, "_stdin_")
	in := c.Get()

	c, _ = Resolve(t.Lexical, t.Frame, "_stdout_")
	out := c.Get()

	c, _ = Resolve(t.Lexical, t.Frame, "_stderr_")
	err := c.Get()

	files := []*os.File{rpipe(in), wpipe(out), wpipe(err)}

	attr := &os.ProcAttr{Dir: dir, Env: t.MakeEnv(), Files: files}

	rv, problem := t.execute(arg0, argv, attr)
	if problem != nil {
		panic(ErrNotExecutable + problem.Error())
	}

	return t.Return(rv)
}

func (t *Task) launch() {
	t.RunWithExceptionHandling(nil)
	close(t.Done)
}

func (t *Task) Listen() {
	t.Code = Cons(nil, Null)

	for c := range t.Eval {
		t.Dump = Cdr(t.Dump)

		saved := t.registers

		end := Cons(nil, Null)

		if t.Code == nil {
			break
		}

		SetCar(t.Code, c)
		SetCdr(t.Code, end)

		t.Code = end
		t.NewStates(svCode, psEvalCommand)

		t.Code = c

		var result Cell
		if !t.RunWithExceptionHandling(end) {
			t.registers = saved

			SetCar(t.Code, nil)
			SetCdr(t.Code, Null)
		} else {
			result = Car(t.Dump)
		}

		t.Done <- result
	}
}

func (t *Task) LexicalVar(state int64) bool {
	t.RemoveState()

	c := t.Lexical
	s := toContext(t.Self()).Expose()

	r := Raw(Car(t.Code))
	if strings.Contains(r, "=") {
		msg := "'" + r + "' is not a valid variable name"
		panic(msg)
	}

	if s != c {
		t.NewStates(svLexical)
		t.Lexical = s
	}

	t.NewStates(state)

	if s != c {
		t.NewStates(svCarCode | svLexical)
		t.Lexical = c
	} else {
		t.NewStates(svCarCode)
	}

	t.NewStates(psEvalElement)

	if Length(t.Code) == 3 {
		if Raw(Cadr(t.Code)) != "=" {
			msg := "expected '=' after " + r + "'"
			panic(ErrSyntax + msg)
		}
		t.Code = Caddr(t.Code)
	} else {
		t.Code = Cadr(t.Code)
	}

	t.Dump = Cdr(t.Dump)

	return true
}

func (t *Task) Lookup(sym *Symbol) string {
	prefix, r, _ := extractName(true, Raw(sym))

	s := t.GetState()
	if s == psEvalElement && prefix == "" {
		// Variable arguments must be prefixed with a $.
		t.Dump = Cons(sym, t.Dump)
		return ""
	}

	c, o := Resolve(t.Lexical, t.Frame, r)
	if c == nil {
		if s == psEvalHead && prefix == "" {
			// External command names are symbols
			// that are not the name of a variable.
			t.Dump = Cons(sym, t.Dump)
			return ""
		}
		return "'" + r + "' undefined"
	}

	v := c.Get()
	if a, ok := v.(binding); ok {
		t.Dump = Cons(a.bind(o), t.Dump)
	} else {
		t.Dump = Cons(v, t.Dump)
	}

	return ""
}

func (t *Task) RunWithExceptionHandling(end Cell) bool {
	for {
		ok, err := t.run(end)
		if err == nil {
			return ok
		}
		// Inject throw and restart.
		t.Throw(t.File, t.Line, fmt.Sprintf("%v", err))
	}
}

func (t *Task) run(end Cell) (_ bool, err interface{}) {
	defer func() {
		err = recover()
	}()

	for t.Runnable() {
		state := t.GetState()

		switch state {
		case psChangeContext:
			t.Lexical = Car(t.Dump)
			t.Dump = Cdr(t.Dump)

		case psExecBuiltin, psExecMethod:
			args := t.Arguments()

			if state == psExecBuiltin {
				args = t.expand(args)
			}

			t.Code = args

			fallthrough
		case psExecSyntax:
			m := Car(t.Dump).(binding)

			if m.ref().Applier()(t, t.Code) {
				continue
			}

		case psExecIf, psExecWhileBody:
			if !Car(t.Dump).Bool() {
				t.Code = Cdr(t.Code)

				for Car(t.Code) != Null &&
					!IsAtom(Car(t.Code)) {
					t.Code = Cdr(t.Code)
				}

				if Car(t.Code) != Null &&
					Raw(Car(t.Code)) != "else" {
					msg := "expected 'else'"
					panic(ErrSyntax + msg)
				}
			}

			if Cdr(t.Code) == Null {
				break
			}

			if t.RemoveState() == psExecWhileBody {
				t.NewStates(psExecWhileTest, svCode)
			}
			t.NewStates(psEvalBlock)

			t.Code = Cdr(t.Code)

			fallthrough
		case psEvalBlock:
			if t.Code == end {
				return true, ""
			}

			if t.Code == Null ||
				!IsPair(t.Code) || !IsPair(Car(t.Code)) {
				break
			}

			if Cdr(t.Code) == Null || !IsPair(Cadr(t.Code)) {
				t.ReplaceStates(psEvalCommand)
			} else {
				t.NewStates(svCdrCode, psEvalCommand)
			}

			t.Code = Car(t.Code)
			t.Dump = Cdr(t.Dump)

			fallthrough
		case psEvalCommand:
			if t.Code == Null {
				t.Dump = Cons(t.Code, t.Dump)
				break
			}

			if h, ok := t.Code.(*PairPlus); ok {
				t.File = h.File
				t.Line = h.Line
			}

			t.ReplaceStates(psExecCommand,
				svCdrCode,
				psEvalHead)
			t.Code = Car(t.Code)

			continue

		case psExecCommand:
			switch k := Car(t.Dump).(type) {
			case *String, *Symbol:
				t.Dump = Cons(external, t.Dump)

				t.ReplaceStates(psExecBuiltin,
					psEvalArguments)
			case binding:
				switch k.ref().(type) {
				case *builtin:
					t.ReplaceStates(psExecBuiltin,
						psEvalArguments)

				case *method:
					t.ReplaceStates(psExecMethod,
						psEvalArguments)
				case *syntax:
					t.ReplaceStates(psExecSyntax)
					continue
				}

			case *Continuation:
				t.ReplaceStates(psReturn, psEvalArguments)

			default:
				msg := fmt.Sprintf("can't evaluate: %v", Car(t.Dump))
				panic(msg)
			}

			t.Dump = Cons(nil, t.Dump)

			fallthrough
		case psEvalArguments:
			if t.Code == Null {
				break
			}

			t.NewStates(svCdrCode, psEvalElement)

			t.Code = Car(t.Code)

			fallthrough
		case psEvalElement, psEvalHead, psEvalMember:
			if t.Code == Null {
				t.Dump = Cons(t.Code, t.Dump)
				break
			} else if IsPair(t.Code) {
				if IsAtom(Cdr(t.Code)) {
					t.ReplaceStates(svLexical,
						psEvalMember,
						psChangeContext,
						svCdrCode,
						t.GetState())
					t.Code = Car(t.Code)
				} else {
					t.ReplaceStates(psEvalCommand)
				}
				continue
			} else if sym, ok := t.Code.(*Symbol); ok {
				msg := t.Lookup(sym)
				if msg != "" {
					panic(msg)
				}
				break
			} else {
				t.Dump = Cons(t.Code, t.Dump)
				break
			}

		case psExecDefine:
			toContext(t.Lexical).Define(Raw(t.Code), Car(t.Dump))

		case psExecPublic:
			toContext(t.Lexical).Public(Raw(t.Code), Car(t.Dump))

		case psExecSet:
			k := Raw(t.Code.(*Symbol))
			r, _ := Resolve(t.Lexical, t.Frame, k)
			if r == nil {
				msg := "'" + k + "' undefined"
				panic(msg)
			}

			r.Set(Car(t.Dump))

		case psExecSplice:
			l := Car(t.Dump)
			t.Dump = Cdr(t.Dump)

			if !IsPair(l) {
				t.Dump = Cons(l, t.Dump)
				break
			}

			for l != Null {
				t.Dump = Cons(Car(l), t.Dump)
				l = Cdr(l)
			}

		case psExecWhileTest:
			t.ReplaceStates(psExecWhileBody,
				svCode,
				psEvalElement)
			t.Code = Car(t.Code)
			t.Dump = Cdr(t.Dump)

			continue

		case psFatal:
			return false, ""

		case psNoOp:
			break

		case psReturn:
			args := t.Arguments()

			t.Continuation = *Car(t.Dump).(*Continuation)
			t.Dump = Cons(Car(args), t.Dump)

		default:
			if state >= svMax {
				msg := fmt.Sprintf("invalid state: %s",
					t.Code)
				panic(msg)
			} else {
				t.RestoreState()
				continue
			}
		}

		t.RemoveState()
	}

	return true, ""
}

func (t *Task) Runnable() bool {
	return t.action.Runnable() && t.Stack != Null
}

func (t *Task) Self() Cell {
	return Car(t.Dump).(binding).self()
}

func (t *Task) Stop() {
	t.action.Terminate()

	close(t.Eval)

	if t.pid > 0 {
		system.TerminateProcess(t.pid)
	}

	t.childrenl.RLock()
	defer t.childrenl.RUnlock()
	for k, v := range t.children {
		if v {
			k.Stop()
		}
	}
}

func (t *Task) Suspend() {
	if t.pid > 0 {
		system.SuspendProcess(t.pid)
	}

	t.childrenl.RLock()
	defer t.childrenl.RUnlock()
	for k, v := range t.children {
		if v {
			k.Suspend()
		}
	}

	t.action.Suspend()
}

func (t *Task) Throw(file string, line int, text string) {
	throw := NewSymbol("throw")

	var resolved Reference

	/* Unwind stack until we can resolve 'throw'. */
	for t.Lexical != scope0 {
		state := t.GetState()
		if state <= 0 {
			t.Lexical = scope0
			break
		}

		switch t.Lexical.(type) {
		case context:
			resolved, _ = Resolve(t.Lexical, t.Frame, "throw")
		}

		if resolved != nil {
			break
		}

		t.RemoveState()
	}

	kind := "error/runtime"
	code := "1"

	if strings.HasPrefix(text, "oh: ") {
		args := strings.SplitN(text, ": ", 4)
		code = args[1]
		kind = args[2]
		text = args[3]
	}
	c := List(
		throw, List(
			NewSymbol("_exception"),
			List(NewSymbol("quote"), NewSymbol(kind)),
			NewStatus(NewSymbol(code).Status()),
			List(NewSymbol("quote"), NewSymbol(text)),
			NewInteger(int64(line)),
			List(NewSymbol("quote"), NewSymbol(path.Base(file))),
		),
	)

	t.Code = c
	t.Dump = List(ExitSuccess)

	t.ReplaceStates(psNoOp, psEvalCommand, psNoOp)
}

func (t *Task) Validate(
	args Cell, minimum, maximum int64, validators ...validator,
) {
	given := Length(args)

	if given < minimum {
		panic(fmt.Sprintf(
			"expected %s (%d given)",
			namedCount(minimum, "argument", "s"),
			given,
		))
	}

	if -1 < maximum && maximum < given {
		panic(fmt.Sprintf(
			"expected %s (%d given)",
			namedCount(maximum, "argument", "s"),
			given,
		))
	}

	var i int64
	for _, validator := range validators {
		if args == Null {
			break
		}

		i++

		if !validator(Car(args)) {
			panic(fmt.Sprintf(
				"argument %d (of %d) is invalid",
				i, given,
			))
		}

		args = Cdr(args)
	}
}

func (t *Task) Wait() {
	t.childrenl.Lock()
	defer t.childrenl.Unlock()
	for k, v := range t.children {
		if v {
			<-k.Done
		}
		delete(t.children, k)
	}
}

/* Unbound cell definition. */

type unbound struct {
	r closure
}

func NewUnbound(c closure) *unbound {
	return &unbound{c}
}

func (u *unbound) Bool() bool {
	return true
}

func (u *unbound) Equal(c Cell) bool {
	if u, ok := c.(*unbound); ok {
		return u.r == u.ref()
	}
	return false
}

func (u *unbound) String() string {
	return fmt.Sprintf("%%unbound %p%%", u)
}

/* Unbound-specific functions */

func (u *unbound) bind(c Cell) binding {
	return u
}

func (u *unbound) ref() closure {
	return u.r
}

func (u *unbound) self() Cell {
	return nil
}

func Call(c Cell) Cell {
	task0l.Lock()
	defer task0l.Unlock()

	if taskc == nil {
		taskc = NewTask(nil, nil, nil)
	}

	taskc.registers = task0.registers

	taskc.Code = c
	taskc.Dump = List(ExitSuccess)
	taskc.Stack = List(NewInteger(psEvalCommand))
	taskc.Frame = task0.Frame

	taskc.RunWithExceptionHandling(nil)

	return Car(taskc.Dump)
}

func Exit() {
	exit(ExitSuccess)
}

func ForegroundTask() *Task {
	return task0
}

func IsContext(c Cell) bool {
	switch c.(type) {
	case context:
		return true
	}
	return false
}

func IsText(c Cell) bool {
	return IsSymbol(c) || IsString(c)
}

func MakeParser(input InputFunc) Parser {
	return parser.New(deref, input)
}

func Resolve(s Cell, f Cell, k string) (Reference, Cell) {
	if s != nil {
		c := toContext(s)
		if c != nil {
			if v := c.Access(k); v != nil {
				return v, s
			}
		}
	}

	if f != nil {
		for f != Null {
			o := toContext(Car(f))
			if v := o.Access(k); v != nil {
				return v, o
			}
			f = Cdr(f)
		}
	}

	return nil, nil
}

func StartFile(origin string, args []string) {
	bindSpecialVariables(origin, args)

	filename := args[0]
	eval(List(NewSymbol("source"), NewSymbol(filename)))
}

func StartInteractive(p Parser) {
	interactive = true
	bindSpecialVariables("", os.Args)
	system.BecomeForegroundProcessGroup()
	initSignalHandling()
	system.BecomeProcessGroupLeader()
	p.ParseCommands("oh", evaluate)
}

func StartNonInteractive() {
	if os.Args[1] == "-c" {
		if len(os.Args) == 2 {
			msg := "-c requires an argument"
			println(ErrSyntax + msg)
			os.Exit(1)
		}

		args := append([]string{os.Args[0]}, os.Args[3:]...)
		bindSpecialVariables("", args)

		b := bufio.NewReader(strings.NewReader(os.Args[2] + "\n"))
		MakeParser(b.ReadString).ParseBuffer("-c", eval)
	} else {
		StartFile(filepath.Dir(os.Args[1]), os.Args[1:])
	}
}

func bindSpecialVariables(origin string, args []string) {
	arglist := Null

	for i, s := range args {
		v := NewString(s)

		scope0.Define(strconv.Itoa(i), v)

		arglist = Cons(v, arglist)
	}

	scope0.Define("_args_", Cdr(Reverse(arglist)))
	if wd, err := os.Getwd(); err == nil {
		sys.Public("OLDPWD", NewSymbol(wd))
		sys.Public("PWD", NewSymbol(wd))
		if !filepath.IsAbs(origin) {
			origin = filepath.Join(wd, origin)
		}
	}
	scope0.Define("_origin_", NewSymbol(origin))
}

func braceExpand(arg string) []string {
	prefix := strings.SplitN(arg, "{", 2)
	if len(prefix) != 2 {
		return []string{arg}
	}

	suffix := strings.SplitN(prefix[1], "}", 2)
	if len(suffix) != 2 {
		return []string{arg}
	}

	middle := strings.Split(suffix[0], ",")
	if len(middle) <= 1 {
		return []string{arg}
	}

	expanded := make([]string, 0, len(middle))
	for _, v := range middle {
		v = prefix[0] + v + suffix[1]
		expanded = append(expanded, braceExpand(v)...)
	}

	return expanded
}

func conduitContext() context {
	if envc != nil {
		return envc
	}

	envc = NewScope(namespace, nil)
	envc.PublicMethod("_reader_close_", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		toConduit(t.Self()).ReaderClose()
		return t.Return(True)
	})
	envc.PublicMethod("_writer_close_", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		toConduit(t.Self()).WriterClose()
		return t.Return(True)
	})
	envc.PublicMethod("close", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		toConduit(t.Self()).Close()
		return t.Return(True)
	})
	envc.PublicMethod("keys", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(Null)
	})
	envc.PublicMethod("read", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(Car(toConduit(t.Self()).Read(MakeParser, t.Throw)))
	})
	envc.PublicMethod("readline", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(toConduit(t.Self()).ReadLine())
	})
	envc.PublicMethod("readlist", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(toConduit(t.Self()).Read(MakeParser, t.Throw))
	})
	envc.PublicMethod("write", func(t *Task, args Cell) bool {
		toConduit(t.Self()).Write(args)
		return t.Return(True)
	})

	return envc
}

func last() int {
	index := 0

	jobsl.RLock()
	defer jobsl.RUnlock()

	for k := range jobs {
		if k > index {
			index = k
		}
	}

	return index
}

func launchForegroundTask() {
	if task0 != nil {
		task0.Job.saveMode()
	}
	task0 = NewTask(nil, nil, nil)

	go task0.Listen()
}

func control(t *Task, args Cell) *Task {
	if !jobControlEnabled() || t != task0 {
		return nil
	}

	index := 0
	if args != Null {
		if a, ok := Car(args).(Atom); ok {
			index = int(a.Int())
		}
	} else {
		index = last()
	}

	jobsl.Lock()
	defer jobsl.Unlock()

	found, ok := jobs[index]
	if !ok {
		return nil
	}

	delete(jobs, index)

	return found
}

func eval(c Cell) (Cell, bool) {
	task0.Eval <- c
	return <-task0.Done, true
}

func (t *Task) expand(args Cell) Cell {
	list := Null

	for ; args != Null; args = Cdr(args) {
		c := Car(args)
		s := Raw(c)

		done := true

		switch c.(type) {
		case *Symbol:
			done = false
		}

		if done {
			list = AppendTo(list, NewSymbol(s))
			continue
		}

		for _, e := range braceExpand(s) {
			e = tildeExpand(t.Lexical, t.Frame, e)

			if !strings.ContainsAny(e, "*?[") {
				list = AppendTo(list, NewSymbol(e))
				continue
			}

			m, err := filepath.Glob(e)
			if err != nil || len(m) == 0 {
				panic("no matches found: " + e)
			}

			for _, v := range m {
				if v[0] != '.' || s[0] == '.' {
					list = AppendTo(list, NewString(v))
				}
			}
		}
	}

	return list
}

func extractName(complete bool, s string) (prefix, name, suffix string) {
	name = s
	prefix = ""
	suffix = ""

	if !strings.HasPrefix(name, "$") {
		return
	}

	name = s[1:]

	hasSuffix := true
	remove := 0
	if complete {
		hasSuffix = strings.HasSuffix(name, "}")
		remove = 1
	}

	if strings.HasPrefix(name, "{") && hasSuffix {
		name = name[1 : len(name)-remove]
		prefix = "${"
		suffix = "}"
	} else {
		prefix = "$"
	}

	return
}

func init() {
	rand.Seed(time.Now().UnixNano())

	CacheSymbols(symbols...)

	external = NewUnboundBuiltin((*Task).External, nil)

	namespace = NewScope(nil, nil)

	namespace.PublicMethod("_del_", func(t *Task, args Cell) bool {
		panic("public members cannot be removed from this type")
	})
	namespace.PublicMethod("_dir_", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		self := toContext(t.Self())
		l := Null
		for _, s := range self.Complete("") {
			l = Cons(NewSymbol(s), l)
		}
		return t.Return(l)
	})
	namespace.PublicMethod("_get_", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1)
		k := Raw(Car(args))

		c, _ := Resolve(toContext(t.Self()), nil, k)
		if c == nil {
			panic("'" + k + "' undefined")
		} else if a, ok := c.Get().(binding); ok {
			return t.Return(a.bind(t.Self()))
		} else {
			return t.Return(c.Get())
		}
	})
	namespace.PublicMethod("child", func(t *Task, args Cell) bool {
		panic("this type cannot be a parent")
	})
	namespace.PublicMethod("clone", func(t *Task, args Cell) bool {
		panic("this type cannot be cloned")
	})
	namespace.PublicMethod("define", func(t *Task, args Cell) bool {
		panic("private members cannot be added to this type")
	})
	namespace.PublicMethod("keys", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		self := toContext(t.Self())
		l := Null
		for _, s := range self.Faces().Prev().Complete(false, "") {
			l = Cons(NewSymbol(s), l)
		}
		return t.Return(l)
	})
	namespace.PublicMethod("export", func(t *Task, args Cell) bool {
		panic("public members cannot be added to this type")
	})

	object := NewScope(namespace, nil)

	/* Standard Methods. */
	object.PublicMethod("_del_", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1)
		self := toContext(t.Self())
		s := Raw(Car(args))

		ok := self.Faces().Prev().Remove(s)
		if !ok {
			panic("'" + s + "' undefined")
		}

		return t.Return(NewBoolean(ok))
	})
	object.PublicMethod("_set_", func(t *Task, args Cell) bool {
		t.Validate(args, 2, 2)
		k := Raw(Car(args))
		v := Cadr(args)

		toContext(t.Self()).Public(k, v)
		return t.Return(v)
	})
	object.PublicMethod("child", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		c := toContext(t.Self())
		return t.Return(NewObject(NewScope(c.Expose(), nil)))
	})
	object.PublicMethod("clone", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		c := toContext(t.Self())
		return t.Return(NewObject(c.Expose().Copy()))
	})
	object.PublicMethod("context", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		self := toContext(t.Self())
		bare := self.Expose()
		if self == bare {
			self = NewObject(bare)
		}
		return t.Return(self)
	})
	object.PublicMethod("eval", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1)
		scope := toContext(t.Self()).Expose()
		t.RemoveState()
		if t.Lexical != scope {
			t.NewStates(svLexical)
			t.Lexical = scope
		}
		t.NewStates(psEvalElement)
		t.Code = Car(args)
		t.Dump = Cdr(t.Dump)

		return true
	})
	object.PublicMethod("has", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1)
		c, _ := Resolve(t.Self(), nil, Raw(Car(args)))

		return t.Return(NewBoolean(c != nil))
	})
	object.PublicMethod("interpolate", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1, IsText)
		l := toContext(t.Self())
		if t.Lexical == l.Expose() {
			l = toContext(t.Lexical)
		}

		modified := interpolate(l, t.Frame, Raw(Car(args)))

		return t.Return(NewString(modified))
	})
	object.PublicSyntax("export", func(t *Task, args Cell) bool {
		return t.LexicalVar(psExecPublic)
	})

	/* Root Scope. */
	scope0 = NewScope(object, nil)

	/* Arithmetic. */
	bindArithmetic(scope0)

	/* Builtins. */
	scope0.DefineBuiltin("bg", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 1, IsNumber)
		SetCar(t.Dump, Null)

		found := control(t, args)
		if found == nil {
			return false
		}

		found.Continue()

		SetCar(t.Dump, found)

		return false
	})
	scope0.DefineBuiltin("cd", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 1, IsText)
		dir := ""
		if args == Null {
			c, _ := Resolve(t.Lexical, t.Frame, "HOME")
			dir = Raw(c.Get())
		} else {
			dir = Raw(Car(args))
		}

		if dir == "-" {
			c, _ := Resolve(t.Lexical, t.Frame, "OLDPWD")
			dir = c.Get().String()

		}

		return t.Chdir(dir)
	})
	scope0.DefineBuiltin("debug", func(t *Task, args Cell) bool {
		t.debug("debug")

		return false
	})
	scope0.DefineBuiltin("exists", func(t *Task, args Cell) bool {
		t.Validate(args, 1, -1)
		count := 0
		ignore := false
		for ; args != Null; args = Cdr(args) {
			count++
			path := Raw(Car(args))
			if path == "-i" {
				ignore = true
				continue
			}
			s, err := os.Stat(path)
			if err != nil {
				return t.Return(False)
			}
			if ignore && !s.Mode().IsRegular() {
				return t.Return(False)
			}
		}

		return t.Return(NewBoolean(count > 0))
	})
	scope0.DefineBuiltin("fg", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 1, IsNumber)
		found := control(t, args)
		if found == nil {
			return false
		}

		setForegroundTask(found)
		return true
	})
	scope0.DefineBuiltin("jobs", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		jobsl.RLock()
		defer jobsl.RUnlock()

		if !jobControlEnabled() || t != task0 || len(jobs) == 0 {
			return false
		}

		i := make([]int, 0, len(jobs))
		for k := range jobs {
			i = append(i, k)
		}
		sort.Ints(i)
		for k, v := range i {
			cmd, grp := jobs[v].Job.commandAndGroup()
			isdef := " "
			if k == len(jobs)-1 {
				isdef = "+"
			}
			fmt.Printf("[%d]%s\t%d\t%s\n", v, isdef, grp, cmd)
		}
		return false
	})
	scope0.DefineBuiltin("command", func(t *Task, args Cell) bool {
		t.Validate(args, 1, -1, IsText)
		if args == Null {
			SetCar(t.Dump, False)
			return false
		}
		SetCar(t.Dump, Car(args))
		t.Dump = Cons(external, t.Dump)
		t.Dump = Cons(nil, t.Dump)
		for args = Cdr(args); args != Null; args = Cdr(args) {
			t.Dump = Cons(Car(args), t.Dump)
		}
		t.ReplaceStates(psExecBuiltin)
		return true
	})

	/* Generators. */
	bindGenerators(scope0)

	scope0.DefineMethod("channel", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 1, IsNumber)
		capacity := 0
		if args != Null {
			capacity = int(Car(args).(Atom).Int())
		}

		return t.Return(NewChannel(capacity))
	})

	/* Predicates. */
	bindPredicates(scope0)

	/* Relational. */
	bindRelational(scope0)

	scope0.DefineMethod("match", func(t *Task, args Cell) bool {
		t.Validate(args, 2, 2, IsText, IsText)
		pattern := Raw(Car(args))
		text := Raw(Cadr(args))

		ok, err := path.Match(pattern, text)
		if err != nil {
			panic(err)
		}

		return t.Return(NewBoolean(ok))
	})
	scope0.DefineMethod("ne", func(t *Task, args Cell) bool {
		t.Validate(args, 2, -1)
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
	scope0.DefineMethod("resolves", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1, IsText)
		c, _ := Resolve(t.Lexical, t.Frame, Raw(Car(args)))

		return t.Return(NewBoolean(c != nil))
	})

	/* Standard Functions. */
	scope0.DefineMethod("exit", func(t *Task, args Cell) bool {
		t.Dump = List(Car(args))

		t.Stop()

		t.Stack = Null

		return true
	})
	scope0.DefineMethod("fatal", func(t *Task, args Cell) bool {
		t.Dump = List(Car(args))

		t.ReplaceStates(psFatal)

		return true
	})
	scope0.DefineMethod("get-line-number", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(NewInteger(int64(t.Line)))
	})
	scope0.DefineMethod("get-source-file", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(NewSymbol(t.File))
	})
	scope0.DefineMethod("open", func(t *Task, args Cell) bool {
		t.Validate(args, 2, 2, IsText, IsText)
		mode := Raw(Car(args))
		path := Raw(Cadr(args))
		flags := 0

		if !strings.ContainsAny(mode, "-") {
			flags = os.O_CREATE
		}

		read := false
		if strings.ContainsAny(mode, "r") {
			read = true
		}

		write := false
		if strings.ContainsAny(mode, "w") {
			write = true
			if !strings.ContainsAny(mode, "a") {
				flags |= os.O_TRUNC
			}
		}

		if strings.ContainsAny(mode, "a") {
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

		return t.Return(NewPipe(r, w))
	})
	scope0.DefineMethod("random", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(NewFloat(rand.Float64()))
	})
	scope0.DefineMethod("set-line-number", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1, IsNumber)
		t.Line = int(Car(args).(Atom).Int())

		return false
	})
	scope0.DefineMethod("set-source-file", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1, IsText)
		t.File = Raw(Car(args))

		return false
	})
	scope0.DefineMethod("temp-fifo", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
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
			SetCar(args, Car(child.Dump))
		}
		return t.Return(list)
	})

	/* Syntax. */
	scope0.DefineSyntax("_splice_", func(t *Task, args Cell) bool {
		t.ReplaceStates(psExecSplice, psEvalElement)

		t.Code = Car(t.Code)
		t.Dump = Cdr(t.Dump)

		return true
	})
	scope0.DefineSyntax("block", func(t *Task, args Cell) bool {
		t.ReplaceStates(svLexical, psEvalBlock)

		t.NewBlock(toContext(t.Lexical))

		return true
	})
	scope0.DefineSyntax("if", func(t *Task, args Cell) bool {
		t.ReplaceStates(svLexical,
			psExecIf, svCode, psEvalElement)

		t.NewBlock(toContext(t.Lexical))

		t.Code = Car(t.Code)
		t.Dump = Cdr(t.Dump)

		return true
	})
	scope0.DefineSyntax("set", func(t *Task, args Cell) bool {
		t.Dump = Cdr(t.Dump)

		s := Cadr(t.Code)
		if Length(t.Code) == 3 {
			if Raw(s) != "=" {
				panic(ErrSyntax + "expected '='")
			}
			s = Caddr(t.Code)
		}

		t.Code = Car(t.Code)
		if !IsPair(t.Code) {
			t.ReplaceStates(psExecSet, svCode)
		} else {
			t.ReplaceStates(svLexical,
				psExecSet, svCdrCode,
				psChangeContext, psEvalElement,
				svCarCode)
		}

		t.NewStates(psEvalElement)

		t.Code = s
		return true
	})
	scope0.DefineSyntax("spawn", func(t *Task, args Cell) bool {
		c := toContext(t.Lexical)
		child := NewTask(t.Code, NewScope(c, nil), t)

		go child.launch()

		SetCar(t.Dump, child)

		return false
	})
	scope0.DefineSyntax("while", func(t *Task, args Cell) bool {
		t.ReplaceStates(svLexical, psExecWhileTest)

		return true
	})

	/* The rest. */
	bindTheRest(scope0)

	env := NewObject(NewScope(object, nil))
	sys = NewObject(NewScope(object, nil))

	scope0.Define("false", False)
	scope0.Define("true", True)

	scope0.Define("_env_", env)
	scope0.Define("_pid_", NewInteger(int64(system.Pid())))
	scope0.Define("_platform_", NewSymbol(system.Platform))
	scope0.Define("_ppid_", NewInteger(int64(system.Ppid())))
	scope0.Define("_root_", scope0)
	scope0.Define("_sys_", sys)

	sys.Public("_stdin_", NewPipe(os.Stdin, nil))
	sys.Public("_stdout_", NewPipe(nil, os.Stdout))
	sys.Public("_stderr_", NewPipe(nil, os.Stderr))

	/* Environment variables. */
	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		env.Public(kv[0], NewSymbol(kv[1]))
	}

	frame0 = List(env, sys)

	initPlatformSpecific()

	launchForegroundTask()

	b := bufio.NewReader(strings.NewReader(boot.Script))
	MakeParser(b.ReadString).ParseBuffer("boot.oh", eval)
}

func interpolate(l context, f Cell, s string) string {
	cb := func(ref string) string {
		if ref == "$$" {
			return "$"
		}

		name := ref[1:]
		if name[0] == '{' {
			name = name[1 : len(name)-1]
		}

		c, _ := Resolve(l, f, name)
		if c == nil {
			panic("'" + name + "' undefined")
		}

		return Raw(c.Get())
	}

	r := regexp.MustCompile(`(?:\$\$)|(?:\${.+?})|(?:\$[0-9A-Z_a-z]+)`)
	return r.ReplaceAllStringFunc(s, cb)
}

func jobControlEnabled() bool {
	return interactive && system.JobControlSupported()
}

func namedCount(c int64, n string, p string) string {
	s := ""
	if c != 1 {
		s = p
	}

	return fmt.Sprintf("%d %s%s", c, n, s)
}

func pairContext() context {
	if envp != nil {
		return envp
	}

	envp = NewScope(namespace, nil)
	envp.PublicMethod("append", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1)
		var s Cell = ToPair(t.Self())

		n := Cons(Car(s), Null)
		l := n
		for s = Cdr(s); s != Null; s = Cdr(s) {
			SetCdr(n, Cons(Car(s), Null))
			n = Cdr(n)
		}
		SetCdr(n, args)

		return t.Return(l)
	})
	envp.PublicMethod("get", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 1, IsNumber)
		s := ToPair(t.Self())

		i := int64(0)
		if args != Null {
			i = Car(args).(Atom).Int()
			args = Cdr(args)
		}

		var dflt Cell
		if args != Null {
			dflt = args
		}

		return t.Return(Car(Tail(s, i, dflt)))
	})
	envp.PublicMethod("head", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		s := ToPair(t.Self())

		return t.Return(Car(s))
	})
	envp.PublicMethod("keys", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		var s Cell = ToPair(t.Self())
		l := Null

		i := int64(0)
		for s != Null {
			l = Cons(NewInteger(i), l)
			s = Cdr(s)
			i++
		}

		return t.Return(Reverse(l))
	})
	envp.PublicMethod("length", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(NewInteger(Length(t.Self())))
	})
	envp.PublicMethod("reverse", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(Reverse(t.Self()))
	})
	envp.PublicMethod("set", func(t *Task, args Cell) bool {
		t.Validate(args, 2, 2, IsNumber)
		s := ToPair(t.Self())

		i := Car(args).(Atom).Int()
		v := Cadr(args)

		SetCar(Tail(s, i, nil), v)
		return t.Return(v)
	})
	envp.PublicMethod("set-tail", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 2)
		s := ToPair(t.Self())

		i := int64(0)

		v := Car(args)
		args = Cdr(args)

		if args != Null {
			i = v.(Atom).Int()
			v = Car(args)
		}

		SetCdr(Tail(s, i, nil), v)
		return t.Return(v)
	})
	envp.PublicMethod("slice", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 2, IsNumber, IsNumber)
		s := ToPair(t.Self())
		i := Car(args).(Atom).Int()

		j := int64(0)

		args = Cdr(args)
		if args != Null {
			j = Car(args).(Atom).Int()
		}

		return t.Return(Slice(s, i, j))
	})
	envp.PublicMethod("tail", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		s := ToPair(t.Self())

		return t.Return(Cdr(s))
	})
	envp.DefineMethod("to-string", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		var s Cell

		v := ""
		for s = ToPair(t.Self()); s != Null; s = Cdr(s) {
			v = fmt.Sprintf("%s%c", v, int(Car(s).(Atom).Int()))
		}

		return t.Return(NewString(v))
	})

	return envp
}

func rpipe(c Cell) *os.File {
	return c.(*Pipe).ReadFd()

}

func setForegroundTask(t *Task) {
	t.Job.moveToForeground()

	task0, t = t, task0

	t.Stop()
	task0.Continue()
}

func stringContext() context {
	if envs != nil {
		return envs
	}

	envs = NewScope(namespace, nil)
	envs.PublicMethod("join", func(t *Task, args Cell) bool {
		sep := toString(t.Self())
		arr := make([]string, Length(args))

		for i := 0; args != Null; i++ {
			arr[i] = string(Raw(Car(args)))
			args = Cdr(args)
		}

		r := strings.Join(arr, string(Raw(sep)))

		return t.Return(NewString(r))
	})
	envs.PublicMethod("keys", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		return t.Return(Null)
	})
	envs.PublicMethod("length", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		s := Raw(toString(t.Self()))

		return t.Return(NewInteger(int64(len(s))))
	})
	envs.PublicMethod("slice", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 2, IsNumber, IsNumber)
		s := []rune(Raw(toString(t.Self())))

		start := int(Car(args).(Atom).Int())
		end := len(s)

		if Cdr(args) != Null {
			end = int(Cadr(args).(Atom).Int())
		}

		return t.Return(NewString(string(s[start:end])))
	})
	envs.PublicMethod("split", func(t *Task, args Cell) bool {
		t.Validate(args, 1, 1, IsText)
		r := Null

		sep := toString(t.Self())
		str := Car(args)

		l := strings.Split(string(Raw(str)), string(Raw(sep)))

		for i := len(l) - 1; i >= 0; i-- {
			r = Cons(NewString(l[i]), r)
		}

		return t.Return(r)
	})
	envs.PublicMethod("sprintf", func(t *Task, args Cell) bool {
		f := Raw(toString(t.Self()))

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
				argv = append(argv, Raw(t))
			}
		}

		s := fmt.Sprintf(f, argv...)

		return t.Return(NewString(s))
	})
	envs.PublicMethod("to-list", func(t *Task, args Cell) bool {
		t.Validate(args, 0, 0)
		s := Raw(toString(t.Self()))
		l := Null
		for _, char := range s {
			l = Cons(NewInteger(int64(char)), l)
		}

		return t.Return(Reverse(l))
	})

	bindStringPredicates(envs)

	return envs
}

func tildeExpand(l, f Cell, s string) string {
	if !strings.HasPrefix(s, "~") {
		return s
	}
	ref, _ := Resolve(l, f, "HOME")
	return filepath.Join(ref.Get().String(), s[1:])
}

/* Convert cell into a Conduit. */
func toConduit(c Cell) Conduit {
	conduit, ok := c.(Conduit)
	if !ok {
		panic("not a conduit")
	}

	return conduit
}

/* Convert Cell into a context. */
func toContext(c Cell) context {
	switch t := c.(type) {
	case context:
		return t
	case *Channel:
		return conduitContext()
	case *Pair, *PairPlus:
		return pairContext()
	case *Pipe:
		return conduitContext()
	case *String:
		return stringContext()
	}
	panic("not an object")
}

/* Convert Cell into a String. */
func toString(c Cell) *String {
	if s, ok := c.(*String); ok {
		return s
	}

	panic("not a string")
}

func wpipe(c Cell) *os.File {
	return c.(*Pipe).WriteFd()
}

//go:generate ./generate.oh
//go:generate go fmt generated.go
