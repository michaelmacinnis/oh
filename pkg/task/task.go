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
	"math/big"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Binding interface {
	Cell

	Bind(c Context) Binding
	Ref() Closure
	Self() Context
}

type Closure interface {
	Cell

	Applier() Function
	Body() Cell
	CallerLabel() Cell
	Params() Cell
	Scope() Context
	SelfLabel() Cell
}

type ClosureGenerator func(a Function, b, c, l, p Cell, s Context) Closure

type Conduit interface {
	Context

	Close()
	ReaderClose()
	ReadLine(*Task) Cell
	Read(*Task) Cell
	WriterClose()
	Write(c Cell)
}

type Context interface {
	Cell

	Access(key Cell) Reference
	Copy() Context
	Complete(word string) []string
	Define(key, value Cell)
	Exported() map[string]Cell
	Expose() Context
	Faces() *Env
	Prev() Context
	Public(key, value Cell)
	Visibility() *Env

	DefineBuiltin(k string, f Function)
	DefineMethod(k string, f Function)
	DefineSyntax(k string, f Function)
	PublicMethod(k string, f Function)
	PublicSyntax(k string, f Function)
}

type Function func(t *Task, args Cell) bool

type Message struct {
	Cmd     Cell
	File    string
	Line    int
	Problem string
}

type ui interface {
	Close() error
	Exists() bool
	ReadString(delim byte) (line string, err error)
}

type reader func(*Task, common.ReadStringer, string,
	func(string, uintptr) Cell,
	func(Cell, string, int, string) Cell) bool

const (
	SaveCarCode = 1 << iota
	SaveCdrCode
	SaveDump
	SaveFrame
	SaveLexical
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
	psExecIf
	psExecMethod
	psExecPublic
	psExecSet
	psExecSplice
	psExecSyntax
	psExecWhileBody
	psExecWhileTest

	psFatal
	psReturn

	psMax
	SaveCode = SaveCarCode | SaveCdrCode
)

var (
	envc        *Env
	envs        *Env
	frame0      Cell
	external    Cell
	interactive bool
	jobs        = map[int]*Task{}
	parse       reader
	pgid        int
	pid         int
	runnable    chan bool
	scope0      *Scope
	str         = map[string]*String{}
	sys         Context
	task0       *Task
)

var next = map[int64][]int64{
	psEvalArguments:        {SaveCdrCode, psEvalElement},
	psEvalArgumentsBuiltin: {SaveCdrCode, psEvalElementBuiltin},
	psExecIf:               {psEvalBlock},
	psExecWhileBody:        {psExecWhileTest, SaveCode, psEvalBlock},
}

/* Bound cell definition. */

type Bound struct {
	ref     Closure
	context Context
}

func NewBound(ref Closure, context Context) *Bound {
	return &Bound{ref, context}
}

func (b *Bound) Bool() bool {
	return true
}

func (b *Bound) Equal(c Cell) bool {
	if m, ok := c.(*Bound); ok {
		return b.ref == m.Ref() && b.context == m.Self()
	}
	return false
}

func (b *Bound) String() string {
	return fmt.Sprintf("%%bound %p%%", b)
}

/* Bound-specific functions */

func (b *Bound) Bind(c Context) Binding {
	if c == b.context {
		return b
	}
	return NewBound(b.ref, c)
}

func (b *Bound) Ref() Closure {
	return b.ref
}

func (b *Bound) Self() Context {
	return b.context
}

/* Builtin cell definition. */

type Builtin struct {
	Command
}

func IsBuiltin(c Cell) bool {
	b, ok := c.(Binding)
	if !ok {
		return false
	}

	switch b.Ref().(type) {
	case *Builtin:
		return true
	}
	return false
}

func NewBuiltin(a Function, b, c, l, p Cell, s Context) Closure {
	return &Builtin{
		Command{
			applier: a,
			body:    b,
			clabel:  c,
			slabel:  l,
			params:  p,
			scope:   s,
		},
	}
}

func (b *Builtin) Equal(c Cell) bool {
	return b == c
}

func (b *Builtin) String() string {
	return fmt.Sprintf("%%builtin %p%%", b)
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
		NewScope(t.Lexical.Expose(), conduitEnv()),
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

/* Command cell definition. */

type Command struct {
	applier Function
	body    Cell
	clabel  Cell
	slabel  Cell
	params  Cell
	scope   Context
}

func (c *Command) Bool() bool {
	return true
}

func (c *Command) Applier() Function {
	return c.applier
}

func (c *Command) Body() Cell {
	return c.body
}

func (c *Command) CallerLabel() Cell {
	return c.clabel
}

func (c *Command) Params() Cell {
	return c.params
}

func (c *Command) Scope() Context {
	return c.scope
}

func (c *Command) SelfLabel() Cell {
	return c.slabel
}

/* Continuation cell definition. */

type Continuation struct {
	Dump  Cell
	Stack Cell
	Frame Cell
}

func IsContinuation(c Cell) bool {
	switch c.(type) {
	case *Continuation:
		return true
	}
	return false
}

func NewContinuation(dump, stack, frame Cell) *Continuation {
	return &Continuation{
		Dump:  dump,
		Stack: stack,
		Frame: frame,
	}
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

/* Env cell definition. */

type Env struct {
	hash map[string]Reference
	prev *Env
}

func NewEnv(prev *Env) *Env {
	return &Env{make(map[string]Reference), prev}
}

func (e *Env) Bool() bool {
	return true
}

func (e *Env) Equal(c Cell) bool {
	return e == c
}

func (e *Env) String() string {
	return fmt.Sprintf("%%env %p%%", e)
}

/* Env-specific functions */

func (e *Env) Access(key Cell) Reference {
	for env := e; env != nil; env = env.prev {
		if value, ok := env.hash[key.String()]; ok {
			return value
		}
	}

	return nil
}

func (e *Env) Add(key Cell, value Cell) {
	e.hash[key.String()] = NewVariable(value)
}

func (e *Env) Complete(word string) []string {
	p := e.Prefixed(word)

	cl := make([]string, 0, len(p))

	for k := range p {
		cl = append(cl, k)
	}

	if e.prev != nil {
		cl = append(cl, e.prev.Complete(word)...)
	}

	return cl
}

func (e *Env) Copy() *Env {
	if e == nil {
		return nil
	}

	fresh := NewEnv(e.prev.Copy())

	for k, v := range e.hash {
		fresh.hash[k] = v.Copy()
	}

	return fresh
}

func (e *Env) Method(name string, m Function) {
	e.hash[name] =
		NewConstant(NewBound(NewMethod(
			m, Null, Null, Null, Null, nil), nil))
}

func (e *Env) Prefixed(prefix string) map[string]Cell {
	r := map[string]Cell{}

	for k, v := range e.hash {
		if strings.HasPrefix(k, prefix) {
			r[k] = v.Get()
		}
	}

	return r
}

func (e *Env) Prev() *Env {
	return e.prev
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

/* Method cell definition. */

type Method struct {
	Command
}

func IsMethod(c Cell) bool {
	b, ok := c.(Binding)
	if !ok {
		return false
	}

	switch b.Ref().(type) {
	case *Method:
		return true
	}
	return false
}

func NewMethod(a Function, b, c, l, p Cell, s Context) Closure {
	return &Method{
		Command{
			applier: a,
			body:    b,
			clabel:  c,
			slabel:  l,
			params:  p,
			scope:   s,
		},
	}
}

func (m *Method) Equal(c Cell) bool {
	return m == c
}

func (m *Method) String() string {
	return fmt.Sprintf("%%method %p%%", m)
}

/*
 * Object cell definition.
 * (An object cell only allows access to a context's public members).
 */

type Object struct {
	Context
}

func NewObject(v Context) *Object {
	return &Object{v.Expose()}
}

func (o *Object) Equal(c Cell) bool {
	if o == c {
		return true
	}
	if o, ok := c.(*Object); ok {
		return o.Context == o.Expose()
	}
	return false
}

func (o *Object) String() string {
	return fmt.Sprintf("%%object %p%%", o)
}

/* Object-specific functions */

func (o *Object) Access(key Cell) Reference {
	var obj Context
	for obj = o; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().prev.Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (o *Object) Copy() Context {
	return &Object{
		&Scope{o.Expose().Faces().Copy(), o.Context.Prev()},
	}
}

func (o *Object) Expose() Context {
	return o.Context
}

func (o *Object) Define(key Cell, value Cell) {
	panic("private members cannot be added to an object")
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
		Scope: NewScope(l.Expose(), conduitEnv()),
		b:     nil,
		c:     nil,
		d:     nil,
		r:     r,
		w:     w,
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
			parse(
				t, p.reader(), p.r.Name(), deref,
				func(c Cell, f string, l int, u string) Cell {
					t.Line = l
					p.c <- c
					<-p.d
					return nil
				},
			)
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
	Lexical Context
}

/* Registers-specific functions. */

func (r *Registers) Arguments() Cell {
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

func (r *Registers) Complete(word string) []string {
	cl := r.Lexical.Complete(word)

	for f:= r.Frame; f != Null; f = Cdr(f) {
		o := Car(f).(Context)
		cl = append(cl, o.Complete(word)...)
	}

	return cl
}

func (r *Registers) GetState() int64 {
	if r.Stack == Null {
		return 0
	}
	return Car(r.Stack).(Atom).Int()
}

func (r *Registers) MakeEnv() []string {
	e := r.Lexical.Exported()

	for f:= r.Frame; f != Null; f = Cdr(f) {
		o := Car(f).(Context)
		for k, v := range o.Exported() {
			if _, ok := e[k]; !ok {
				e[k] = v
			}
		}
	}

	l := make([]string, 0, len(e))

	for k, v := range(e) {
		l = append(l, k[1:] + "=" + raw(v))
	}

	return l
}

func (r *Registers) NewBlock(lexical Context) {
	r.Lexical = NewScope(lexical, nil)
}

func (r *Registers) NewFrame(lexical Context) {
	var state int64 = SaveLexical

	v := r.Lexical.Visibility()
	if v != nil && v != Car(r.Frame).(Context).Visibility() {
		state |= SaveFrame
	}

	r.ReplaceStates(state, psEvalBlock)

	if state&SaveFrame > 0 {
		r.Frame = Cons(NewObject(r.Lexical), r.Frame)
	}

	r.Lexical = NewScope(lexical, nil)
}

func (r *Registers) NewStates(l ...int64) {
	for _, f := range l {
		if f >= SaveMax {
			r.Stack = Cons(NewInteger(f), r.Stack)
			continue
		}

		p := *r

		s := r.GetState()
		if s < SaveMax && f < SaveMax {
			// Previous and current states are save states.
			c := f & s
			if f&SaveCode > 0 || s&SaveCode > 0 {
				c |= SaveCode
			}
			if c&f == f {
				// Nothing new to save.
				continue
			} else if c&s == s {
				// Previous save state is a subset.
				p.RestoreState()
				r.Stack = p.Stack
				if c&SaveCode > 0 {
					f |= SaveCode
				}
			}
		}

		if f&SaveCode > 0 {
			if f&SaveCode == SaveCode {
				r.Stack = Cons(p.Code, r.Stack)
			} else if f&SaveCarCode > 0 {
				r.Stack = Cons(Car(p.Code), r.Stack)
			} else if f&SaveCdrCode > 0 {
				r.Stack = Cons(Cdr(p.Code), r.Stack)
			}
		}

		if f&SaveDump > 0 {
			r.Stack = Cons(p.Dump, r.Stack)
		}

		if f&SaveFrame > 0 {
			r.Stack = Cons(p.Frame, r.Stack)
		}

		if f&SaveLexical > 0 {
			r.Stack = Cons(p.Lexical, r.Stack)
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

	if f&SaveLexical > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if f&SaveFrame > 0 {
		r.Stack = Cdr(r.Stack)
	}

	if f&SaveDump > 0 {
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

	if f&SaveLexical > 0 {
		r.Stack = Cdr(r.Stack)
		r.Lexical = Car(r.Stack).(Context)
	}

	if f&SaveFrame > 0 {
		r.Stack = Cdr(r.Stack)
		r.Frame = Car(r.Stack)
	}

	if f&SaveDump > 0 {
		r.Stack = Cdr(r.Stack)
		r.Dump = Car(r.Stack)
	}

	if f&SaveCode > 0 {
		r.Stack = Cdr(r.Stack)
		r.Code = Car(r.Stack)
	}

	r.Stack = Cdr(r.Stack)
}

func (r *Registers) Return(rv Cell) bool {
	SetCar(r.Dump, rv)

	return false
}

/*
 * Scope cell definition.
 * (A scope cell allows access to a context's public and private members).
 */

type Scope struct {
	env  *Env
	prev Context
}

func NewScope(prev Context, fixed *Env) *Scope {
	return &Scope{NewEnv(NewEnv(fixed)), prev}
}

func (s *Scope) Bool() bool {
	return true
}

func (s *Scope) Equal(c Cell) bool {
	return s == c
}

func (s *Scope) String() string {
	return fmt.Sprintf("%%scope %p%%", s)
}

/* Scope-specific functions */

func (s *Scope) Access(key Cell) Reference {
	var obj Context
	for obj = s; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (s *Scope) Complete(word string) []string {
	cl := []string{}

	var obj Context
	for obj = s; obj != nil; obj = obj.Prev() {
		cl = append(cl, obj.Faces().Complete(word)...)
	}

	return cl
}

func (s *Scope) Copy() Context {
	return &Scope{s.env.Copy(), s.prev}
}

func (s *Scope) Exported() map[string]Cell {
	return s.env.prev.Prefixed("$")
}

func (s *Scope) Expose() Context {
	return s
}

func (s *Scope) Faces() *Env {
	return s.env
}

func (s *Scope) Prev() Context {
	return s.prev
}

func (s *Scope) Define(key Cell, value Cell) {
	s.env.Add(key, value)
}

func (s *Scope) Public(key Cell, value Cell) {
	s.env.prev.Add(key, value)
}

func (s *Scope) Visibility() *Env {
	var obj Context
	for obj = s; obj != nil; obj = obj.Prev() {
		env := obj.Faces().prev
		if len(env.hash) != 0 {
			return env
		}
	}

	return nil
}

func (s *Scope) DefineBuiltin(k string, a Function) {
	s.Define(NewSymbol(k),
		NewUnbound(NewBuiltin(a, Null, Null, Null, Null, s)))
}

func (s *Scope) DefineMethod(k string, a Function) {
	s.Define(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, Null, s), s))
}

func (s *Scope) PublicMethod(k string, a Function) {
	s.Public(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, Null, s), s))
}

func (s *Scope) DefineSyntax(k string, a Function) {
	s.Define(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, Null, s), s))
}

func (s *Scope) PublicSyntax(k string, a Function) {
	s.Public(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, Null, s), s))
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

	e := stringEnv()
	l := scope0
	if t != nil {
		l = NewScope(t.Lexical.Expose(), e)
	} else if task0 != nil {
		l = NewScope(task0.Lexical.Expose(), e)
	} else {
		l = NewScope(l, e)
	}

	s := String{l, v}
	p = &s

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

/* Syntax cell definition. */

type Syntax struct {
	Command
}

func IsSyntax(c Cell) bool {
	b, ok := c.(Binding)
	if !ok {
		return false
	}

	switch b.Ref().(type) {
	case *Syntax:
		return true
	}
	return false
}

func NewSyntax(a Function, b, c, l, p Cell, s Context) Closure {
	return &Syntax{
		Command{
			applier: a,
			body:    b,
			clabel:  c,
			slabel:  l,
			params:  p,
			scope:   s,
		},
	}
}

func (m *Syntax) Equal(c Cell) bool {
	return m == c
}

func (m *Syntax) String() string {
	return fmt.Sprintf("%%syntax %p%%", m)
}

/* Task cell definition. */

type Task struct {
	*Job
	*Registers
	Done      chan Cell
	Eval      chan Message
	File      string
	Line      int
	children  map[*Task]bool
	parent    *Task
	pid       int
	suspended chan bool
}

func NewTask(c Cell, l Context, p *Task) *Task {
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
		Job: j,
		Registers: &Registers{
			Continuation: Continuation{
				Dump:  List(NewStatus(0)),
				Stack: List(NewInteger(psEvalBlock)),
				Frame: frame,
			},
			Code:    c,
			Lexical: l,
		},
		Done:      make(chan Cell, 1),
		Eval:      make(chan Message, 1),
		File:      "oh",
		Line:      0,
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
	caller := t.Lexical

	m := Car(t.Dump).(Binding)

	t.NewFrame(m.Ref().Scope())

	t.Code = m.Ref().Body()

	clabel := m.Ref().CallerLabel()
	if clabel != Null {
		t.Lexical.Define(clabel, caller)
	}

	slabel := m.Ref().SelfLabel()
	if slabel != Null {
		t.Lexical.Define(slabel, m.Self().Expose())
	}

	params := m.Ref().Params()
	for args != Null && params != Null && IsAtom(Car(params)) {
		t.Lexical.Define(Car(params), Car(args))
		args, params = Cdr(args), Cdr(params)
	}
	if IsCons(Car(params)) {
		t.Lexical.Define(Caar(params), args)
	}

	cc := NewContinuation(Cdr(t.Dump), t.Stack, t.Frame)
	t.Lexical.Define(NewSymbol("return"), cc)

	return true
}

func (t *Task) Closure(n ClosureGenerator) bool {
	slabel := Car(t.Code)
	t.Code = Cdr(t.Code)

	params := Null
	if IsSymbol(slabel) {
		params = Car(t.Code)
		t.Code = Cdr(t.Code)
	} else {
		params = slabel
		slabel = Null
	}

	equals := Car(t.Code)
	t.Code = Cdr(t.Code)

	clabel := Null
	if raw(equals) != "=" {
		clabel = equals
		equals = Car(t.Code)
		t.Code = Cdr(t.Code)
	}

	if raw(equals) != "=" {
		panic(common.ErrSyntax + "expected '='")
	}

	body := t.Code
	scope := t.Lexical

	c := n((*Task).Apply, body, clabel, slabel, params, scope)
	if slabel == Null {
		SetCar(t.Dump, NewUnbound(c))
	} else {
		SetCar(t.Dump, NewBound(c, scope))
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
	fmt.Printf("%s: t.Code = %v, t.Dump = %v\n", s, t.Code, t.Dump)
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
	t.Dump = Cdr(t.Dump)

	arg0, problem := adapted.LookPath(raw(Car(t.Dump)))

	SetCar(t.Dump, False)

	if problem != nil {
		panic(common.ErrNotFound + problem.Error())
	}

	argv := []string{arg0}

	for ; args != Null; args = Cdr(args) {
		argv = append(argv, raw(Car(args)))
	}

	c, _ := Resolve(t.Lexical, t.Frame, NewSymbol("$CWD"))
	dir := c.Get().String()

	c, _ = Resolve(t.Lexical, t.Frame, NewSymbol("_stdin_"))
	in := c.Get()

	c, _ = Resolve(t.Lexical, t.Frame, NewSymbol("_stdout_"))
	out := c.Get()

	c, _ = Resolve(t.Lexical, t.Frame, NewSymbol("_stderr_"))
	err := c.Get()

	files := []*os.File{rpipe(in), wpipe(out), wpipe(err)}

	attr := &os.ProcAttr{Dir: dir, Env: t.MakeEnv(), Files: files}

	status, problem := t.Execute(arg0, argv, attr)
	if problem != nil {
		panic(common.ErrNotExecutable + problem.Error())
	}

	return t.Return(status)
}

func (t *Task) Launch() {
	t.Run(nil, "")
	close(t.Done)
}

func (t *Task) Listen() {
	t.Code = Cons(nil, Null)

	for m := range t.Eval {
		saved := *(t.Registers)

		end := Cons(nil, Null)

		if t.Code == nil {
			break
		}

		if m.File != "" {
			t.File = m.File
		}
		if m.Line != -1 {
			t.Line = m.Line
		}
		SetCar(t.Code, m.Cmd)
		SetCdr(t.Code, end)

		t.Code = end
		t.NewStates(SaveCode, psEvalCommand)

		t.Code = m.Cmd
		status := t.Run(end, m.Problem)
		var result Cell = nil
		if status < 0 {
			break
		} else if status > 0 {
			*(t.Registers) = saved

			SetCar(t.Code, nil)
			SetCdr(t.Code, Null)
		} else {
			result = Car(t.Dump)
			t.Dump = Cdr(t.Dump)
		}

		t.Done <- result
	}
}

func (t *Task) LexicalVar(state int64) bool {
	t.RemoveState()

	c := t.Lexical
	s := t.Self().Expose()

	r := raw(Car(t.Code))
	if t.Strict() && number(r) {
		msg := r + " cannot be used as a variable name"
		panic(msg)
	}

	if s != c {
		t.NewStates(SaveLexical)
		t.Lexical = s
	}

	t.NewStates(state)

	if s != c {
		t.NewStates(SaveCarCode | SaveLexical)
		t.Lexical = c
	} else {
		t.NewStates(SaveCarCode)
	}

	t.NewStates(psEvalElement)

	if Length(t.Code) == 3 {
		if raw(Cadr(t.Code)) != "=" {
			msg := "expected '=' after " + r + "'"
			panic(common.ErrSyntax + msg)
		}
		t.Code = Caddr(t.Code)
	} else {
		t.Code = Cadr(t.Code)
	}

	t.Dump = Cdr(t.Dump)

	return true
}

func (t *Task) Lookup(sym *Symbol, simple bool) (bool, string) {
	c, s := Resolve(t.Lexical, t.Frame, sym)
	if c == nil {
		r := raw(sym)
		if t.GetState() == psEvalMember || (t.Strict() && !number(r)) {
			return false, "'" + r + "' undefined"
		}
		t.Dump = Cons(sym, t.Dump)
	} else if simple && !isSimple(c.Get()) {
		t.Dump = Cons(sym, t.Dump)
	} else if a, ok := c.Get().(Binding); ok {
		t.Dump = Cons(a.Bind(s), t.Dump)
	} else {
		t.Dump = Cons(c.Get(), t.Dump)
	}

	return true, ""
}

func (t *Task) Run(end Cell, problem string) (status int) {
	status = 0

	defer func() {
		r := recover()
		if r == nil {
			return
		}

		if problem == "" {
			t.Throw(t.File, t.Line, fmt.Sprintf("%v", r))
		} else {
			println(problem)
		}

		status = 1
	}()

	for t.Runnable() && t.Stack != Null {
		state := t.GetState()

		switch state {
		case psChangeContext:
			t.Lexical = Car(t.Dump).(Context)
			t.Dump = Cdr(t.Dump)

		case psExecBuiltin, psExecMethod:
			args := t.Arguments()

			if state == psExecBuiltin {
				args = expand(t, args)
			}

			t.Code = args

			fallthrough
		case psExecSyntax:
			m := Car(t.Dump).(Binding)

			if m.Ref().Applier()(t, t.Code) {
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
					raw(Car(t.Code)) != "else" {
					msg := "expected 'else'"
					panic(common.ErrSyntax + msg)
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
				//t.Dump = Cdr(t.Dump)
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
			t.Dump = Cdr(t.Dump)

			fallthrough
		case psEvalCommand:
			if t.Code == Null {
				t.Dump = Cons(t.Code, t.Dump)
				break
			}

			t.ReplaceStates(psExecCommand,
				SaveCdrCode,
				psEvalElement)
			t.Code = Car(t.Code)

			continue

		case psExecCommand:
			switch k := Car(t.Dump).(type) {
			case *String, *Symbol:
				t.Dump = Cons(external, t.Dump)

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
				msg := fmt.Sprintf("can't evaluate: %v", t)
				panic(msg)
			}

			t.Dump = Cons(nil, t.Dump)

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
				t.Dump = Cons(t.Code, t.Dump)
				break
			} else if IsCons(t.Code) {
				if IsAtom(Cdr(t.Code)) {
					t.ReplaceStates(SaveLexical,
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
				t.Dump = Cons(t.Code, t.Dump)
				break
			}

		case psExecDefine:
			t.Lexical.Define(t.Code, Car(t.Dump))

		case psExecPublic:
			t.Lexical.Public(t.Code, Car(t.Dump))

		case psExecSet:
			k := t.Code.(*Symbol)
			r, _ := Resolve(t.Lexical, t.Frame, k)
			if r == nil {
				msg := "'" + k.String() + "' undefined"
				panic(msg)
			}

			r.Set(Car(t.Dump))

		case psExecSplice:
			l := Car(t.Dump)
			t.Dump = Cdr(t.Dump)

			if !IsCons(l) {
				t.Dump = Cons(l, t.Dump)
				break
			}

			for l != Null {
				t.Dump = Cons(Car(l), t.Dump)
				l = Cdr(l)
			}

		case psExecWhileTest:
			t.ReplaceStates(psExecWhileBody,
				SaveCode,
				psEvalElement)
			t.Code = Car(t.Code)
			t.Dump = Cdr(t.Dump)

			continue

		case psFatal:
			return -1

		case psReturn:
			args := t.Arguments()

			t.Continuation = *Car(t.Dump).(*Continuation)
			t.Dump = Cons(Car(args), t.Dump)

			break

		default:
			if state >= SaveMax {
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

	return
}

func (t *Task) Runnable() bool {
	return !<-t.suspended
}

func (t *Task) Self() Context {
	return Car(t.Dump).(Binding).Self()
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

	c, _ := Resolve(t.Lexical, nil, NewSymbol("strict"))
	if c == nil {
		return false
	}

	return c.Get().(Cell).Bool()
}

func (t *Task) Suspend() {
	if t.pid > 0 {
		SuspendProcess(t.pid)
	}

	for k, v := range t.children {
		if v {
			k.Suspend()
		}
	}

	t.suspended = make(chan bool)
}

func (t *Task) Throw(file string, line int, text string) {
	kind := "error/runtime"
	code := "1"

	if strings.HasPrefix(text, "oh: ") {
		args := strings.SplitN(text, ": ", 4)
		code = args[1]
		kind = args[2]
		text = args[3]
	}
	c := List(
		NewSymbol("throw"), List(
			NewSymbol("exception"),
			NewSymbol(kind),
			NewSymbol(text),
			NewStatus(NewSymbol(code).Int()),
			NewSymbol(path.Base(file)),
			NewInteger(int64(line)),
		),
	)
	Call(t, c, text)
}

func (t *Task) Wait() {
	for k, v := range t.children {
		if v {
			<-k.Done
		}
		delete(t.children, k)
	}
}

/* Unbound cell definition. */

type Unbound struct {
	ref Closure
}

func NewUnbound(Ref Closure) *Unbound {
	return &Unbound{Ref}
}

func (u *Unbound) Bool() bool {
	return true
}

func (u *Unbound) Equal(c Cell) bool {
	if u, ok := c.(*Unbound); ok {
		return u.ref == u.Ref()
	}
	return false
}

func (u *Unbound) String() string {
	return fmt.Sprintf("%%unbound %p%%", u)
}

/* Unbound-specific functions */

func (u *Unbound) Bind(c Context) Binding {
	return u
}

func (u *Unbound) Ref() Closure {
	return u.ref
}

func (u *Unbound) Self() Context {
	return nil
}

func Call(t *Task, c Cell, problem string) string {
	if t == nil {
		return raw(evaluate(c, "", -1, problem))
	}

	saved := *(t.Registers)

	t.Code = c
	t.Dump = List(NewStatus(0))
	t.Stack = List(NewInteger(psEvalCommand))

	t.Run(nil, problem)

	status := Car(t.Dump)

	*(t.Registers) = saved

	return raw(status)
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
	task0 = NewTask(nil, nil, nil)

	go task0.Listen()
}

func PrintError(file string, line int, msg string) {
	file = path.Base(file)
	fmt.Fprintf(os.Stderr, "%s: %d: %v\n", file, line, msg)
}

func Pgid() int {
	return pgid
}

func Resolve(s Context, f Cell, k *Symbol) (Reference, Context) {
	if s != nil {
		if v := s.Access(k); v != nil {
			return v, s
		}
	}

	if f != nil {
		for f != Null {
			o := Car(f).(Context)
			if v := o.Access(k); v != nil {
				return v, o
			}
			f = Cdr(f)
		}
	}

	return nil, nil
}

func Start(parser reader, cli ui) {
	LaunchForegroundTask()

	parse = parser
	eval := func(c Cell, f string, l int, p string) Cell {
		task0.Eval <- Message{Cmd: c, File: f, Line: l, Problem: p}
		<-task0.Done
		return nil
	}

	b := bufio.NewReader(strings.NewReader(boot.Script))
	parse(task0, b, "boot.oh", deref, eval)

	/* Command-line arguments */
	argc := len(os.Args)
	args := Null
	origin := ""
	if argc > 1 && os.Args[1] != "-c" {
		origin = filepath.Dir(os.Args[1])
		scope0.Define(NewSymbol("_0_"), NewSymbol(os.Args[1]))

		for i, v := range os.Args[2:] {
			k := "_" + strconv.Itoa(i+1) + "_"
			scope0.Define(NewSymbol(k), NewSymbol(v))
		}

		for i := argc - 1; i > 1; i-- {
			args = Cons(NewSymbol(os.Args[i]), args)
		}
	} else {
		scope0.Define(NewSymbol("_0_"), NewSymbol(os.Args[0]))
	}
	scope0.Define(NewSymbol("_args_"), args)

	if wd, err := os.Getwd(); err == nil {
		sys.Public(NewSymbol("$CWD"), NewSymbol(wd))
		if !filepath.IsAbs(origin) {
			origin = filepath.Join(wd, origin)
		}
	}
	scope0.Define(NewSymbol("_origin_"), NewSymbol(origin))

	interactive = false
	if argc > 1 {
		if os.Args[1] == "-c" {
			if argc == 2 {
				msg := "-c requires an argument"
				println(common.ErrSyntax + msg)
				os.Exit(1)
			}
			s := os.Args[2] + "\n"
			b := bufio.NewReader(strings.NewReader(s))
			parse(task0, b, "-c", deref, eval)
		} else {
			cmd := List(NewSymbol("source"), NewSymbol(os.Args[1]))
			eval(cmd, os.Args[1], 0, "")
		}
	} else if cli.Exists() {
		interactive = true

		InitSignalHandling()

		pgid = BecomeProcessGroupLeader()

		if parse(task0, cli, "oh", deref, evaluate) {
			fmt.Printf("\n")
		}
		cli.Close()
	} else {
		eval(
			List(NewSymbol("source"), NewSymbol("/dev/stdin")),
			"/dev/stdin", 0, "")
	}

	os.Exit(status(Car(task0.Dump)))
}

/* Convert Context into a Conduit. (Return nil if not possible). */
func asConduit(o Context) Conduit {
	if c, ok := o.(Conduit); ok {
		return c
	}

	return nil
}

func conduitEnv() *Env {
	if envc != nil {
		return envc
	}

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
		toConduit(t.Self()).Close()
		return t.Return(True)
	})
	envc.Method("_reader_close_", func(t *Task, args Cell) bool {
		toConduit(t.Self()).ReaderClose()
		return t.Return(True)
	})
	envc.Method("read", func(t *Task, args Cell) bool {
		return t.Return(toConduit(t.Self()).Read(t))
	})
	envc.Method("readline", func(t *Task, args Cell) bool {
		return t.Return(toConduit(t.Self()).ReadLine(t))
	})
	envc.Method("_writer_close_", func(t *Task, args Cell) bool {
		toConduit(t.Self()).WriterClose()
		return t.Return(True)
	})
	envc.Method("write", func(t *Task, args Cell) bool {
		toConduit(t.Self()).Write(args)
		return t.Return(True)
	})

	return envc
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
		for k := range jobs {
			if k > index {
				index = k
			}
		}
	}

	found, ok := jobs[index]
	if !ok {
		return nil
	}

	delete(jobs, index)

	return found
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

	builtin := NewBuiltin((*Task).External, Null, Null, Null, Null, nil)
	external = NewUnbound(builtin)

	public := func(t *Task, args Cell) bool {
		return t.LexicalVar(psExecPublic)
	}

	object := NewScope(nil, nil)

	object.PublicSyntax("public", public)

	/* Standard Methods. */
	object.PublicMethod("child", func(t *Task, args Cell) bool {
		return t.Return(NewObject(NewScope(t.Self().Expose(), nil)))
	})
	object.PublicMethod("clone", func(t *Task, args Cell) bool {
		return t.Return(NewObject(t.Self().Expose().Copy()))
	})
	object.PublicMethod("context", func(t *Task, args Cell) bool {
		self := t.Self()
		bare := self.Expose()
		if self == bare {
			self = NewObject(bare)
		}
		return t.Return(self)
	})
	object.PublicMethod("eval", func(t *Task, args Cell) bool {
		scope := t.Self().Expose()
		t.RemoveState()
		if t.Lexical != scope {
			t.NewStates(SaveLexical)
			t.Lexical = scope
		}
		t.NewStates(psEvalElement)
		t.Code = Car(args)
		t.Dump = Cdr(t.Dump)

		return true
	})
	object.PublicMethod("get-slot", func(t *Task, args Cell) bool {
		s := raw(Car(args))
		k := NewSymbol(s)

		c, o := Resolve(t.Self(), nil, k)
		if c == nil {
			panic("'" + s + "' undefined")
		} else if a, ok := c.Get().(Binding); ok {
			return t.Return(a.Bind(o))
		} else {
			return t.Return(c.Get())
		}
	})
	object.PublicMethod("has", func(t *Task, args Cell) bool {
		c, _ := Resolve(t.Self(), nil, NewSymbol(raw(Car(args))))

		return t.Return(NewBoolean(c != nil))
	})
	object.PublicMethod("interpolate", func(t *Task, args Cell) bool {
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

			c, _ := Resolve(l, t.Frame, sym)
			if c == nil {
				sym := NewSymbol("$" + name)
				c, _ = Resolve(l, t.Frame, sym)
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

	object.PublicMethod("set-slot", func(t *Task, args Cell) bool {
		s := raw(Car(args))
		v := Cadr(args)

		k := NewSymbol(s)

		t.Self().Public(k, v)
		return t.Return(v)
	})

	/* Root Scope. */
	scope0 = NewScope(object, nil)

	/* Arithmetic. */
	bindArithmetic(scope0)

	/* Builtins. */
	scope0.DefineBuiltin("bg", func(t *Task, args Cell) bool {
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
		err := os.Chdir(raw(Car(args)))
		status := 0
		if err != nil {
			status = 1
		}

		if wd, err := os.Getwd(); err == nil {
			t.Lexical.Public(NewSymbol("$CWD"), NewSymbol(wd))
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
		found := control(t, args)
		if found == nil {
			return false
		}

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
		c, _ := Resolve(t.Lexical, t.Frame, sym)

		if c == nil {
			return t.Return(sym)
		}

		return t.Return(c.Get())
	})
	scope0.DefineBuiltin("run", func(t *Task, args Cell) bool {
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
	scope0.DefineMethod("resolves", func(t *Task, args Cell) bool {
		c, _ := Resolve(t.Lexical, t.Frame, NewSymbol(raw(Car(args))))

		return t.Return(NewBoolean(c != nil))
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
		t.Dump = List(Car(args))

		t.Stop()

		return true
	})
	scope0.DefineMethod("fatal", func(t *Task, args Cell) bool {
		t.Dump = List(Car(args))

		t.ReplaceStates(psFatal)

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
	scope0.DefineMethod("get-line-number", func(t *Task, args Cell) bool {
		return t.Return(NewInteger(int64(t.Line)))
	})
	scope0.DefineMethod("get-source-file", func(t *Task, args Cell) bool {
		return t.Return(NewSymbol(t.File))
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
	scope0.DefineMethod("set-line-number", func(t *Task, args Cell) bool {
		t.Line = int(Car(args).(Atom).Int())

		return false
	})
	scope0.DefineMethod("set-source-file", func(t *Task, args Cell) bool {
		t.File = raw(Car(args))

		return false
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
			SetCar(args, Car(child.Dump))
		}
		return t.Return(list)
	})

	/* Syntax. */
	scope0.DefineSyntax("block", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveLexical, psEvalBlock)

		t.NewBlock(t.Lexical)

		return true
	})
	scope0.DefineSyntax("if", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveLexical,
			psExecIf, SaveCode, psEvalElement)

		t.NewBlock(t.Lexical)

		t.Code = Car(t.Code)
		t.Dump = Cdr(t.Dump)

		return true
	})
	scope0.DefineSyntax("set", func(t *Task, args Cell) bool {
		t.Dump = Cdr(t.Dump)

		s := Null
		if Length(t.Code) == 3 {
			if raw(Cadr(t.Code)) != "=" {
				panic(common.ErrSyntax + "expected '='")
			}
			s = Caddr(t.Code)
		} else {
			s = Cadr(t.Code)
		}

		t.Code = Car(t.Code)
		if !IsCons(t.Code) {
			t.ReplaceStates(psExecSet, SaveCode)
		} else {
			t.ReplaceStates(SaveLexical,
				psExecSet, SaveCdrCode,
				psChangeContext, psEvalElement,
				SaveCarCode)
		}

		t.NewStates(psEvalElement)

		t.Code = s
		return true
	})
	scope0.DefineSyntax("spawn", func(t *Task, args Cell) bool {
		child := NewTask(t.Code, NewScope(t.Lexical, nil), t)

		go child.Launch()

		SetCar(t.Dump, child)

		return false
	})
	scope0.DefineSyntax("_splice_", func(t *Task, args Cell) bool {
		t.ReplaceStates(psExecSplice, psEvalElement)

		t.Code = Car(t.Code)
		t.Dump = Cdr(t.Dump)

		return true
	})
	scope0.DefineSyntax("while", func(t *Task, args Cell) bool {
		t.ReplaceStates(SaveLexical, psExecWhileTest)

		return true
	})

	/* The rest. */
	bindTheRest(scope0)

	env := NewObject(NewScope(object, nil))
	sys = NewObject(NewScope(object, nil))

	scope0.Define(NewSymbol("false"), False)
	scope0.Define(NewSymbol("true"), True)

	scope0.Define(NewSymbol("$$"), NewInteger(int64(os.Getpid())))
	scope0.Define(NewSymbol("_env_"), env)
	scope0.Define(NewSymbol("_platform_"), NewSymbol(Platform))
	scope0.Define(NewSymbol("_root_"), scope0)
	scope0.Define(NewSymbol("_sys_"), sys)

	sys.Public(NewSymbol("_stdin_"), NewPipe(scope0, os.Stdin, nil))
	sys.Public(NewSymbol("_stdout_"), NewPipe(scope0, nil, os.Stdout))
	sys.Public(NewSymbol("_stderr_"), NewPipe(scope0, nil, os.Stderr))

	/* Environment variables. */
	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		env.Public(NewSymbol("$"+kv[0]), NewSymbol(kv[1]))
	}

	frame0 = List(env, sys)
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

func stringEnv() *Env {
	if envs != nil {
		return envs
	}

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
		sep := toString(t.Self())
		arr := make([]string, Length(args))

		for i := 0; args != Null; i++ {
			arr[i] = string(raw(Car(args)))
			args = Cdr(args)
		}

		r := strings.Join(arr, string(raw(sep)))

		return t.Return(NewString(t, r))
	})
	envs.Method("slice", func(t *Task, args Cell) bool {
		s := []rune(raw(toString(t.Self())))

		start := int(Car(args).(Atom).Int())
		end := len(s)

		if Cdr(args) != Null {
			end = int(Cadr(args).(Atom).Int())
		}

		return t.Return(NewString(t, string(s[start:end])))
	})
	envs.Method("split", func(t *Task, args Cell) bool {
		r := Null

		sep := Car(args)
		str := toString(t.Self())

		l := strings.Split(string(raw(str)), string(raw(sep)))

		for i := len(l) - 1; i >= 0; i-- {
			r = Cons(NewString(t, l[i]), r)
		}

		return t.Return(r)
	})
	envs.Method("sprintf", func(t *Task, args Cell) bool {
		f := raw(toString(t.Self()))

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

		return t.Return(NewString(t, s))
	})
	envs.Method("to-list", func(t *Task, args Cell) bool {
		s := raw(toString(t.Self()))
		l := Null
		for _, char := range s {
			l = Cons(NewInteger(int64(char)), l)
		}

		return t.Return(Reverse(l))
	})

	bindStringPredicates(envs)

	return envs
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

//go:generate ./generate.oh
//go:generate go fmt generated.go
