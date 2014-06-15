/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"bufio"
	"fmt"
	"github.com/peterh/liner"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Function func(t *Task, args Cell) bool

type NewClosure func(a Function, b, l, p Cell, s Context) Closure

type Atom interface {
	Cell

	Float() float64
	Int() int64
	Status() int64
}

type Binding interface {
	Cell

	Bind(c Context) Binding
	Ref() Closure
	Self() Context
}

type Cell interface {
	Bool() bool
	Equal(c Cell) bool
	String() string
}

type Closure interface {
	Cell

	Applier() Function
	Body() Cell
	Label() Cell
	Params() Cell
	Scope() Context
}

type Conduit interface {
	Context

	Close()
	ReaderClose()
	Read() Cell
	ReadLine() Cell
	WriterClose()
	Write(c Cell)
}

type Context interface {
	Cell

	Access(key Cell) Reference
	Copy() Context
	Complete(line, prefix string) []string
	Define(key, value Cell)
	Expose() Context
	Faces() *Env
	Prev() Context
	Public(key, value Cell)
	Remove(key Cell) bool

	DefineBuiltin(k string, f Function)
	DefineMethod(k string, f Function)
	DefineSyntax(k string, f Function)
	PublicMethod(k string, f Function)
	PublicSyntax(k string, f Function)
}

type Number interface {
	Atom

	Greater(c Cell) bool
	Less(c Cell) bool

	Add(c Cell) Number
	Divide(c Cell) Number
	Modulo(c Cell) Number
	Multiply(c Cell) Number
	Subtract(c Cell) Number
}

type Reference interface {
	Cell

	Copy() Reference
	Get() Cell
	Set(c Cell)
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

var Null Cell
var False *Boolean
var True *Boolean

var conduit_env *Env
var env0 *Env
var interactive bool
var runnable chan bool
var scope0 *Scope

var next = map[int64][]int64{
	psEvalArguments:	{SaveCdrCode, psEvalElement},
	psEvalArgumentsBuiltin: {SaveCdrCode, psEvalElementBuiltin},
	psExecIf:		{psEvalBlock},
	psExecWhileBody:	{psExecWhileTest, SaveCode, psEvalBlock},
}

var num [512]*Integer
var res [256]*Status
var str map[string]*String
var sym map[string]*Symbol

func init() {
	interactive = (len(os.Args) <= 1)

	pair := new(Pair)
	pair.car = pair
	pair.cdr = pair

	Null = Cell(pair)

	F := Boolean(false)
	False = &F

	T := Boolean(true)
	True = &T

	ext = NewUnbound(NewBuiltin((*Task).External, Null, Null, Null, nil))

	str = make(map[string]*String)
	sym = make(map[string]*Symbol)

	for _, v := range [...]string{
		"$redirect",
		"append-stderr",
		"append-stdout",
		"channel-stderr",
		"channel-stdout",
		"eval-list",
		"is-boolean",
		"is-builtin",
		"is-channel",
		"is-integer",
		"is-method",
		"is-number",
		"is-object",
		"is-status",
		"is-string",
		"is-symbol",
		"is-syntax",
		"pipe-stderr",
		"pipe-stdout",
		"reader-close",
		"redirect-stderr",
		"redirect-stdin",
		"redirect-stdout",
		"substring",
		"to-string",
		"to-symbol",
		"writer-close",
		"is-control",
		"is-graphic",
		"is-letter",
	} {
		sym[v] = NewSymbol(v)
	}

	conduit_env = NewEnv(nil)
	conduit_env.Method("close", func(t *Task, args Cell) bool {
		GetConduit(Car(t.Scratch).(Binding).Self()).Close()
		return t.Return(True)
	})
	conduit_env.Method("reader-close", func(t *Task, args Cell) bool {
		GetConduit(Car(t.Scratch).(Binding).Self()).ReaderClose()
		return t.Return(True)
	})
	conduit_env.Method("read", func(t *Task, args Cell) bool {
		r := GetConduit(Car(t.Scratch).(Binding).Self()).Read()
		return t.Return(r)
	})
	conduit_env.Method("readline", func(t *Task, args Cell) bool {
		r := GetConduit(Car(t.Scratch).(Binding).Self()).ReadLine()
		return t.Return(r)
	})
	conduit_env.Method("writer-close", func(t *Task, args Cell) bool {
		GetConduit(Car(t.Scratch).(Binding).Self()).WriterClose()
		return t.Return(True)
	})
	conduit_env.Method("write", func(t *Task, args Cell) bool {
		GetConduit(Car(t.Scratch).(Binding).Self()).Write(args)
		return t.Return(True)
	})

	runnable = make(chan bool)
	close(runnable)

	env0 = NewEnv(nil)
	scope0 = NewScope(nil, nil)
}

func AppendTo(list Cell, elements ...Cell) Cell {
	var pair, prev, start Cell

	index := 0

	start = Null

	if list == nil {
		panic("Cannot append to non-existent list.")
	}

	if list != Null {
		start = list

		for prev = list; Cdr(prev) != Null; prev = Cdr(prev) {
		}

	} else if len(elements) > 0 {
		start = Cons(elements[index], Null)
		prev = start
		index++
	}

	for ; index < len(elements); index++ {
		pair = Cons(elements[index], Null)
		SetCdr(prev, pair)
		prev = pair
	}

	return start
}

func Car(c Cell) Cell {
	return c.(*Pair).car
}

func Cdr(c Cell) Cell {
	return c.(*Pair).cdr
}

func Caar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).car
}

func Cadr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).car
}

func Cdar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).cdr
}

func Cddr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).cdr
}

func Caaar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).car.(*Pair).car
}

func Caadr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).car.(*Pair).car
}

func Cadar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).cdr.(*Pair).car
}

func Caddr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).cdr.(*Pair).car
}

func Cdaar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).car.(*Pair).cdr
}

func Cdadr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).car.(*Pair).cdr
}

func Cddar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).cdr.(*Pair).cdr
}

func Cdddr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).cdr.(*Pair).cdr
}

func IsAtom(c Cell) bool {
	switch c.(type) {
	case Atom:
		return true
	}

	return false
}

func IsCons(c Cell) bool {
	switch c.(type) {
	case *Pair:
		return true
	}

	return false
}

func IsSimple(c Cell) bool {
	return IsAtom(c) || IsCons(c)
}

func Join(list Cell, elements ...Cell) Cell {
	var pair, prev, start Cell

	if list != nil && list != Null {
		start = Cons(Car(list), Null)

		for list = Cdr(list); list != Null; list = Cdr(list) {
			pair = Cons(Car(list), Null)
			SetCdr(prev, pair)
			prev = pair
		}
	} else if len(elements) > 0 {
		return Join(elements[0], elements[1:]...)
	} else {
		return Null
	}

	for index := 0; index < len(elements); index++ {
		for list = elements[index]; list != Null; list = Cdr(list) {
			pair = Cons(Car(list), Null)
			SetCdr(prev, pair)
			prev = pair
		}
	}

	return start
}

func JoinTo(list Cell, elements ...Cell) Cell {
	var pair, prev, start Cell

	start = list

	if list == nil {
		panic("Cannot append to non-existent list.")
	} else if list == Null {
		panic("Cannot destructively modify nil value.")
	}

	for ; list != Null; list = Cdr(list) {
		prev = list
	}

	for index := 0; index < len(elements); index++ {
		for list = elements[index]; list != Null; list = Cdr(list) {
			pair = Cons(Car(list), Null)
			SetCdr(prev, pair)
			prev = pair
		}
	}

	return start
}

func Length(list Cell) int64 {
	var length int64 = 0

	for ; list != nil && list != Null; list = Cdr(list) {
		length++
	}

	return length
}

func List(elements ...Cell) Cell {
	if len(elements) <= 0 {
		return Null
	}

	start := Cons(elements[0], Null)
	prev := start

	for index := 1; index < len(elements); index++ {
		pair := Cons(elements[index], Null)
		SetCdr(prev, pair)
		prev = pair
	}

	return start
}

func Raw(c Cell) string {
	if s, ok := c.(*String); ok {
		return s.Raw()
	}

	return c.String()
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

func Reverse(list Cell) Cell {
	reversed := Null

	for ; list != nil && list != Null; list = Cdr(list) {
		reversed = Cons(Car(list), reversed)
	}

	return reversed
}

func SetCar(c, value Cell) {
	c.(*Pair).car = value
}

func SetCdr(c, value Cell) {
	c.(*Pair).cdr = value
}

/* Boolean cell definition. */

type Boolean bool

func NewBoolean(v bool) *Boolean {
	if v {
		return True
	}
	return False
}

func (b *Boolean) Bool() bool {
	return b == True
}

func (b *Boolean) Float() float64 {
	if b == True {
		return 1.0
	}
	return 0.0
}

func (b *Boolean) Int() int64 {
	if b == True {
		return 1
	}
	return 0
}

func (b *Boolean) Status() int64 {
	if b == True {
		return 0
	}
	return 1
}

func (b *Boolean) String() string {
	if b == True {
		return "True"
	}
	return "False"
}

func (b *Boolean) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return bool(*b) == a.Bool()
	}
	return false
}

/* Integer cell definition. */

type Integer int64

func NewInteger(v int64) *Integer {
	if -256 <= v && v <= 255 {
		n := v + 256
		p := num[n]

		if p == nil {
			i := Integer(v)
			p = &i

			num[n] = p
		}

		return p
	}

	i := Integer(v)
	return &i
}

func (i *Integer) Bool() bool {
	return *i != 0
}

func (i *Integer) Float() float64 {
	return float64(*i)
}

func (i *Integer) Int() int64 {
	return int64(*i)
}

func (i *Integer) Status() int64 {
	return int64(*i)
}

func (i *Integer) String() string {
	return strconv.FormatInt(int64(*i), 10)
}

func (i *Integer) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return int64(*i) == a.Int()
	}
	return false
}

func (i *Integer) Greater(c Cell) bool {
	return int64(*i) > c.(Atom).Int()
}

func (i *Integer) Less(c Cell) bool {
	return int64(*i) < c.(Atom).Int()
}

func (i *Integer) Add(c Cell) Number {
	return NewInteger(int64(*i) + c.(Atom).Int())
}

func (i *Integer) Divide(c Cell) Number {
	return NewInteger(int64(*i) / c.(Atom).Int())
}

func (i *Integer) Modulo(c Cell) Number {
	return NewInteger(int64(*i) % c.(Atom).Int())
}

func (i *Integer) Multiply(c Cell) Number {
	return NewInteger(int64(*i) * c.(Atom).Int())
}

func (i *Integer) Subtract(c Cell) Number {
	return NewInteger(int64(*i) - c.(Atom).Int())
}

/* Status cell definition. */

type Status int64

func NewStatus(v int64) *Status {
	if 0 <= v && v <= 255 {
		p := res[v]

		if p == nil {
			s := Status(v)
			p = &s

			res[v] = p
		}

		return p
	}

	s := Status(v)
	return &s
}

func (s *Status) Bool() bool {
	return int64(*s) == 0
}

func (s *Status) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return int64(*s) == a.Status()
	}
	return false
}

func (s *Status) Float() float64 {
	return float64(*s)
}

func (s *Status) Int() int64 {
	return int64(*s)
}

func (s *Status) Status() int64 {
	return int64(*s)
}

func (s *Status) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

func (s *Status) Greater(c Cell) bool {
	return int64(*s) > c.(Atom).Status()
}

func (s *Status) Less(c Cell) bool {
	return int64(*s) < c.(Atom).Status()
}

func (s *Status) Add(c Cell) Number {
	return NewStatus(int64(*s) + c.(Atom).Status())
}

func (s *Status) Divide(c Cell) Number {
	return NewStatus(int64(*s) / c.(Atom).Status())
}

func (s *Status) Modulo(c Cell) Number {
	return NewStatus(int64(*s) % c.(Atom).Status())
}

func (s *Status) Multiply(c Cell) Number {
	return NewStatus(int64(*s) * c.(Atom).Status())
}

func (s *Status) Subtract(c Cell) Number {
	return NewStatus(int64(*s) - c.(Atom).Status())
}

/* Float cell definition. */

type Float float64

func NewFloat(v float64) *Float {
	f := Float(v)
	return &f
}

func (f *Float) Bool() bool {
	return *f != 0
}

func (f *Float) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return float64(*f) == a.Float()
	}
	return false
}

func (f *Float) Float() float64 {
	return float64(*f)
}

func (f *Float) Int() int64 {
	return int64(*f)
}

func (f *Float) Status() int64 {
	return int64(*f)
}

func (f *Float) String() string {
	return strconv.FormatFloat(float64(*f), 'g', -1, 64)
}

func (f *Float) Greater(c Cell) bool {
	return float64(*f) > c.(Atom).Float()
}

func (f *Float) Less(c Cell) bool {
	return float64(*f) < c.(Atom).Float()
}

func (f *Float) Add(c Cell) Number {
	return NewFloat(float64(*f) + c.(Atom).Float())
}

func (f *Float) Divide(c Cell) Number {
	return NewFloat(float64(*f) / c.(Atom).Float())
}

func (f *Float) Modulo(c Cell) Number {
	panic("Type 'float' does not implement 'modulo'.")
}

func (f *Float) Multiply(c Cell) Number {
	return NewFloat(float64(*f) * c.(Atom).Float())
}

func (f *Float) Subtract(c Cell) Number {
	return NewFloat(float64(*f) - c.(Atom).Float())
}

/* Symbol cell definition. */

type Symbol string

func NewSymbol(v string) *Symbol {
	p, ok := sym[v]

	if ok {
		return p
	}

	s := Symbol(v)
	p = &s

	if len(v) <= 8 {
		sym[v] = p
	}

	return p
}

func (s *Symbol) Bool() bool {
	if string(*s) == "False" {
		return false
	}

	return true
}

func (s *Symbol) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(*s) == a.String()
	}
	return false
}

func (s *Symbol) Float() (f float64) {
	var err error
	if f, err = strconv.ParseFloat(string(*s), 64); err != nil {
		panic(err)
	}
	return f
}

func (s *Symbol) Int() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*s), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *Symbol) Status() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*s), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *Symbol) String() string {
	return string(*s)
}

func (s *Symbol) Greater(c Cell) bool {
	return string(*s) > c.(Atom).String()
}

func (s *Symbol) Less(c Cell) bool {
	return string(*s) < c.(Atom).String()
}

func (s *Symbol) isFloat() bool {
	_, err := strconv.ParseFloat(string(*s), 64)
	return err == nil
}

func (s *Symbol) isInt() bool {
	_, err := strconv.ParseInt(string(*s), 0, 64)
	return err == nil
}

func (s *Symbol) Add(c Cell) Number {
	if s.isInt() {
		return NewInteger(s.Int() + c.(Atom).Int())
	} else if s.isFloat() {
		return NewFloat(s.Float() + c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'add'.")
}

func (s *Symbol) Divide(c Cell) Number {
	if s.isInt() {
		return NewInteger(s.Int() / c.(Atom).Int())
	} else if s.isFloat() {
		return NewFloat(s.Float() / c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'divide'.")
}

func (s *Symbol) Modulo(c Cell) Number {
	if s.isInt() {
		return NewInteger(s.Int() % c.(Atom).Int())
	}

	panic("Type 'symbol' does not implement 'modulo'.")
}

func (s *Symbol) Multiply(c Cell) Number {
	if s.isInt() {
		return NewInteger(s.Int() * c.(Atom).Int())
	} else if s.isFloat() {
		return NewFloat(s.Float() * c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'multiply'.")
}

func (s *Symbol) Subtract(c Cell) Number {
	if s.isInt() {
		return NewInteger(s.Int() - c.(Atom).Int())
	} else if s.isFloat() {
		return NewFloat(s.Float() - c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'subtract'.")
}

/* String cell definition. */

type String string

func NewRawString(v string) *String {
	p, ok := str[v]

	if ok {
		return p
	}

	s := String(v)
	p = &s

	if len(v) <= 8 {
		str[v] = p
	}

	return p
}

func NewString(q string) *String {
	v, _ := strconv.Unquote("\"" + q + "\"")

	return NewRawString(v)
}

func (s *String) Bool() bool {
	return true
}

func (s *String) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(*s) == a.String()
	}
	return false
}

func (s *String) Float() (f float64) {
	var err error
	if f, err = strconv.ParseFloat(string(*s), 64); err != nil {
		panic(err)
	}
	return f
}

func (s *String) Int() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*s), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *String) Raw() string {
	return string(*s)
}

func (s *String) Status() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*s), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *String) String() string {
	return strconv.Quote(string(*s))
}

/* Pair cell definition. */

type Pair struct {
	car Cell
	cdr Cell
}

func Cons(h, t Cell) Cell {
	return &Pair{car: h, cdr: t}
}

func (p *Pair) Bool() bool {
	return p != Null
}

func (p *Pair) String() (s string) {
	s = ""

	if IsCons(p.car) && IsCons(Cdr(p.car)) {
		s += "("
	}

	if p.car != Null {
		s += p.car.String()
	}

	if IsCons(p.car) && IsCons(Cdr(p.car)) {
		s += ")"
	}

	if IsCons(p.cdr) {
		if p.cdr == Null {
			return s
		}

		s += " "
	} else {
		s += "::"
	}

	s += p.cdr.String()

	return s
}

func (p *Pair) Equal(c Cell) bool {
	if p == Null && c == Null {
		return true
	}
	return p.car.Equal(Car(c)) && p.cdr.Equal(Cdr(c))
}

/* Convert Channel/Pipe Context (or child Context) into a Conduit. */

func GetConduit(o Context) Conduit {
	for o != nil {
		if c, ok := o.Expose().(Conduit); ok {
			return c
		}
		o = o.Prev()
	}

	panic("Not a conduit")
	return nil
}

/* Channel cell definition. */

type Channel struct {
	*Scope
	v chan Cell
}

func NewChannel(t *Task, cap int) Context {
	return NewObject(&Channel{
		NewScope(t.Lexical.Expose(), conduit_env),
		make(chan Cell, cap),
	})
}

func (self *Channel) String() string {
	return fmt.Sprintf("%%channel %p%%", self)
}

func (self *Channel) Equal(c Cell) bool {
	return self == c
}

func (self *Channel) Close() {
	self.WriterClose()
}

func (self *Channel) Expose() Context {
	return self
}

func (self *Channel) ReaderClose() {
	return
}

func (self *Channel) Read() Cell {
	v := <-self.v
	if v == nil {
		return Null
	}
	return v
}

func (self *Channel) ReadLine() Cell {
	v := <-self.v
	if v == nil {
		return False
	}
	return NewString(v.String())
}

func (self *Channel) WriterClose() {
	close(self.v)
}

func (self *Channel) Write(c Cell) {
	self.v <- c
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

func NewPipe(t *Task, r *os.File, w *os.File) Context {
	p := &Pipe{
		Scope: NewScope(t.Lexical.Expose(), conduit_env),
		b:     nil, c: nil, d: nil, r: r, w: w,
	}

	if r == nil && w == nil {
		var err error

		if p.r, p.w, err = os.Pipe(); err != nil {
			p.r, p.w = nil, nil
		}
	}

	runtime.SetFinalizer(p, (*Pipe).Close)

	return NewObject(p)
}

func (self *Pipe) String() string {
	return fmt.Sprintf("%%pipe %p%%", self)
}

func (self *Pipe) Equal(c Cell) bool {
	return self == c
}

func (self *Pipe) Close() {
	if self.r != nil && len(self.r.Name()) > 0 {
		self.ReaderClose()
	}

	if self.w != nil && len(self.w.Name()) > 0 {
		self.WriterClose()
	}
}

func (self *Pipe) Expose() Context {
	return self
}

func (self *Pipe) reader() *bufio.Reader {
	if self.b == nil {
		self.b = bufio.NewReader(self.r)
	}

	return self.b
}

func (self *Pipe) ReaderClose() {
	if self.r != nil {
		self.r.Close()
		self.r = nil
	}
}

/*
 * TODO: Rather than using cli to set and reset the terminal mode when the
 * reading from stdin, Read and ReadLine should save the previous terminal
 * mode set it to cooked and then reset it when they are done.
 */

func (self *Pipe) Read() Cell {
	if self.r == nil {
		return Null
	}

	if self.c == nil {
		self.c = make(chan Cell)
		self.d = make(chan bool)
		go func() {
			Parse(self.reader(), func(c Cell) {
				self.c <- c
				<-self.d
			})
			self.c <- Null
		}()
	} else {
		self.d <- true
	}

	return <-self.c
}

func (self *Pipe) ReadLine() Cell {
	s, err := self.reader().ReadString('\n')
	if err != nil && len(s) == 0 {
		self.b = nil
		return Null
	}

	return NewString(strings.TrimRight(s, "\n"))
}

func (self *Pipe) WriterClose() {
	if self.w != nil {
		self.w.Close()
		self.w = nil
	}
}

func (self *Pipe) Write(c Cell) {
	if self.w == nil {
		panic("write to closed pipe")
	}

	fmt.Fprintln(self.w, c)
}

/* Pipe-specific functions */

func (self *Pipe) ReadFd() *os.File {
	return self.r
}

func (self *Pipe) WriteFd() *os.File {
	return self.w
}

/* Combiner cell definition. */

type Combiner struct {
	applier Function
	body    Cell
	label   Cell
	params  Cell
	scope   Context
}

func (self *Combiner) Bool() bool {
	return true
}

func (self *Combiner) Applier() Function {
	return self.applier
}

func (self *Combiner) Body() Cell {
	return self.body
}

func (self *Combiner) Params() Cell {
	return self.params
}

func (self *Combiner) Label() Cell {
	return self.label
}

func (self *Combiner) Scope() Context {
	return self.scope
}

/* Builtin cell definition. */

type Builtin struct {
	Combiner
}

func NewBuiltin(a Function, b, l, p Cell, s Context) Closure {
	return &Builtin{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
	}
}

func (self *Builtin) String() string {
	return fmt.Sprintf("%%builtin %p%%", self)
}

func (self *Builtin) Equal(c Cell) bool {
	return self == c
}

/* Method cell definition. */

type Method struct {
	Combiner
}

func NewMethod(a Function, b, l, p Cell, s Context) Closure {
	return &Method{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
	}
}

func (self *Method) String() string {
	return fmt.Sprintf("%%method %p%%", self)
}

func (self *Method) Equal(c Cell) bool {
	return self == c
}

/* Syntax cell definition. */

type Syntax struct {
	Combiner
}

func NewSyntax(a Function, b, l, p Cell, s Context) Closure {
	return &Syntax{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
	}
}

func (self *Syntax) String() string {
	return fmt.Sprintf("%%syntax %p%%", self)
}

func (self *Syntax) Equal(c Cell) bool {
	return self == c
}

/* Env cell definition. */

type Env struct {
	hash map[string]Reference
	prev *Env
}

func NewEnv(prev *Env) *Env {
	return &Env{make(map[string]Reference), prev}
}

func (self *Env) Bool() bool {
	return true
}

func (self *Env) Equal(c Cell) bool {
	return self == c
}

func (self *Env) String() string {
	return fmt.Sprintf("%%env %p%%", self)
}

/* Env-specific functions */

func (self *Env) Access(key Cell) Reference {
	for env := self; env != nil; env = env.prev {
		if value, ok := env.hash[key.String()]; ok {
			return value
		}
	}

	return nil
}

func (self *Env) Add(key Cell, value Cell) {
	self.hash[key.String()] = NewVariable(value)
}

func (self *Env) Complete(line, prefix string) []string {
	cl := []string{}

	for k, _ := range self.hash {
		if strings.HasPrefix(k, prefix) {
			cl = append(cl, line+k)
		}
	}

	if self.prev != nil {
		cl = append(cl, self.prev.Complete(line, prefix)...)
	}

	return cl
}

func (self *Env) Copy() *Env {
	if self == nil {
		return nil
	}

	fresh := NewEnv(self.prev.Copy())

	for k, v := range self.hash {
		fresh.hash[k] = v.Copy()
	}

	return fresh
}

func (self *Env) Method(name string, m Function) {
	self.hash[name] =
		NewConstant(NewBound(NewMethod(m, Null, Null, Null, nil), nil))
}

func (self *Env) Prev() *Env {
	return self.prev
}

func (self *Env) Remove(key Cell) bool {
	_, ok := self.hash[key.String()]

	delete(self.hash, key.String())

	return ok
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

func (self *Object) Equal(c Cell) bool {
	if self == c {
		return true
	}
	if o, ok := c.(*Object); ok {
		return self.Context == o.Expose()
	}
	return false
}

func (self *Object) String() string {
	return fmt.Sprintf("%%object %p%%", self)
}

/* Object-specific functions */

func (self *Object) Access(key Cell) Reference {
	var obj Context
	for obj = self; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().prev.Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (self *Object) Copy() Context {
	return &Object{
		&Scope{self.Expose().Faces().Copy(), self.Context.Prev()},
	}
}

func (self *Object) Expose() Context {
	return self.Context
}

func (self *Object) Define(key Cell, value Cell) {
	panic("Private members cannot be added to an object.")
}

/* Continuation cell definition. */

type Continuation struct {
	Scratch Cell
	Stack   Cell
}

func NewContinuation(scratch Cell, stack Cell) *Continuation {
	return &Continuation{Scratch: scratch, Stack: stack}
}

func (self *Continuation) Bool() bool {
	return true
}

func (self *Continuation) Equal(c Cell) bool {
	return self == c
}

func (self *Continuation) String() string {
	return fmt.Sprintf("%%continuation %p%%", self)
}

/* Registers cell definition. */

type Registers struct {
	Continuation // Stack and Dump

	Code    Cell // Control
	Dynamic *Env
	Lexical Context
}

/* Registers-specific functions. */

func (self *Registers) Arguments() Cell {
	e := Car(self.Scratch)
	l := Null

	for e != nil {
		l = Cons(e, l)

		self.Scratch = Cdr(self.Scratch)
		e = Car(self.Scratch)
	}

	self.Scratch = Cdr(self.Scratch)

	return l
}

func (self *Registers) Complete(line, prefix string) []string {
	completions := self.Lexical.Complete(line, prefix)
	return append(completions, self.Dynamic.Complete(line, prefix)...)
}

func (self *Registers) GetState() int64 {
	if self.Stack == Null {
		return 0
	}
	return Car(self.Stack).(Atom).Int()
}

func (self *Registers) NewBlock(dynamic *Env, lexical Context) {
	self.Dynamic = NewEnv(dynamic)
	self.Lexical = NewScope(lexical, nil)
}

func (self *Registers) NewStates(l ...int64) {
	for _, f := range l {
		if f >= SaveMax {
			self.Stack = Cons(NewInteger(f), self.Stack)
			continue
		}

		if s := self.GetState(); s < SaveMax && f&s == f {
			continue
		}

		if f&SaveCode > 0 {
			if f&SaveCode == SaveCode {
				self.Stack = Cons(self.Code, self.Stack)
			} else if f&SaveCarCode > 0 {
				self.Stack = Cons(Car(self.Code), self.Stack)
			} else if f&SaveCdrCode > 0 {
				self.Stack = Cons(Cdr(self.Code), self.Stack)
			}
		}

		if f&SaveDynamic > 0 {
			self.Stack = Cons(self.Dynamic, self.Stack)
		}

		if f&SaveLexical > 0 {
			self.Stack = Cons(self.Lexical, self.Stack)
		}

		if f&SaveScratch > 0 {
			self.Stack = Cons(self.Scratch, self.Stack)
		}

		self.Stack = Cons(NewInteger(f), self.Stack)
	}
}

func (self *Registers) RemoveState() {
	f := self.GetState()

	self.Stack = Cdr(self.Stack)
	if f >= SaveMax {
		return
	}

	if f&SaveScratch > 0 {
		self.Stack = Cdr(self.Stack)
	}

	if f&SaveLexical > 0 {
		self.Stack = Cdr(self.Stack)
	}

	if f&SaveDynamic > 0 {
		self.Stack = Cdr(self.Stack)
	}

	if f&SaveCode > 0 {
		self.Stack = Cdr(self.Stack)
	}
}

func (self *Registers) ReplaceStates(l ...int64) {
	self.RemoveState()
	self.NewStates(l...)
}

func (self *Registers) RestoreState() {
	f := self.GetState()

	if f == 0 || f >= SaveMax {
		return
	}

	if f&SaveScratch > 0 {
		self.Stack = Cdr(self.Stack)
		self.Scratch = Car(self.Stack)
	}

	if f&SaveLexical > 0 {
		self.Stack = Cdr(self.Stack)
		self.Lexical = Car(self.Stack).(Context)
	}

	if f&SaveDynamic > 0 {
		self.Stack = Cdr(self.Stack)
		self.Dynamic = Car(self.Stack).(*Env)
	}

	if f&SaveCode > 0 {
		self.Stack = Cdr(self.Stack)
		self.Code = Car(self.Stack)
	}

	self.Stack = Cdr(self.Stack)
}

func (self *Registers) Return(rv Cell) bool {
	SetCar(self.Scratch, rv)

	return false
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

/* Task cell definition. */

type Task struct {
	*Job
	*Registers
	Done      chan Cell
	Eval      chan Cell
	children  map[*Task]bool
	parent    *Task
	pid	int
	suspended chan bool
}

func NewTask(s int64, c Cell, d *Env, l Context, p *Task) *Task {
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
				Stack:   List(NewInteger(s)),
			},
			Code:    c,
			Dynamic: d,
			Lexical: l,
		},
		Done:      make(chan Cell, 1),
		Eval:      make(chan Cell, 1),
		children:  make(map[*Task]bool),
		parent:    p,
		pid:	0,
		suspended: runnable,
	}

	if p != nil {
		p.children[t] = true
	}

	return t
}

func NewTask0() *Task {
	return NewTask(psEvalBlock, Cons(nil, Null), env0, scope0, nil)
}

func (self *Task) Bool() bool {
	return true
}

func (self *Task) String() string {
	return fmt.Sprintf("%%task %p%%", self)
}

func (self *Task) Equal(c Cell) bool {
	return self == c
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

func (t *Task) Closure(n NewClosure) bool {
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

	c := n((*Task).Apply, body, label, params, scope)
	if label == Null {
		SetCar(t.Scratch, NewUnbound(c))
	} else {
		SetCar(t.Scratch, NewBound(c, scope))
	}

	return false
}

func (self *Task) Continue() {
	if self.pid > 0 {
		syscall.Kill(self.pid, syscall.SIGCONT)
	}

	for k, v := range self.children {
		if v {
			k.Continue()
		}
	}

	close(self.suspended)
}

func (t *Task) Debug(s string) {
	fmt.Printf("%s: t.Code = %v, t.Scratch = %v\n", s, t.Code, t.Scratch)
}

func (t *Task) DynamicVar(state int64) bool {
	r := Raw(Car(t.Code))
	if t.Strict() && number(r) {
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

func (self *Task) Execute(arg0 string, argv []string, attr *os.ProcAttr) (*Status, error) {

	self.Lock()
	defer self.Unlock()

	attr.Sys = &syscall.SysProcAttr{
		Sigdfl: []syscall.Signal{syscall.SIGTTIN, syscall.SIGTTOU},
	}
	if self.group == 0 {
		attr.Sys.Setpgid = true
		attr.Sys.Foreground = true
	} else {
		attr.Sys.Joinpgrp = self.group
	}

	proc, err := os.StartProcess(arg0, argv, attr)
	if err != nil {
		return nil, err
	}

	if self.group == 0 {
		self.group = proc.Pid
	}

	self.pid = proc.Pid

	status := JoinProcess(proc.Pid)

	self.pid = 0

	return NewStatus(int64(status.ExitStatus())), err
}

func (t *Task) External(args Cell) bool {
	t.Scratch = Cdr(t.Scratch)

	arg0, problem := exec.LookPath(Raw(Car(t.Scratch)))

	SetCar(t.Scratch, False)

	if problem != nil {
		panic(problem)
	}

	argv := []string{arg0}

	for ; args != Null; args = Cdr(args) {
		argv = append(argv, Car(args).String())
	}

	c := Resolve(t.Lexical, t.Dynamic, NewSymbol("$cwd"))
	dir := c.Get().String()

	in := rpipe(Resolve(t.Lexical, t.Dynamic, NewSymbol("$stdin")).Get())
	out := wpipe(Resolve(t.Lexical, t.Dynamic, NewSymbol("$stdout")).Get())
	err := wpipe(Resolve(t.Lexical, t.Dynamic, NewSymbol("$stderr")).Get())

	files := []*os.File{in, out, err}

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

func (t *Task) LexicalVar(state int64) bool {
	t.RemoveState()

	l := Car(t.Scratch).(Binding).Self().Expose()
	if t.Lexical != l {
		t.NewStates(SaveLexical)
		t.Lexical = l
	}

	t.NewStates(state)

	r := Raw(Car(t.Code))
	if t.Strict() && number(r) {
		panic(r + " cannot be used as a variable name")
	}

	t.NewStates(SaveCarCode|SaveLexical, psEvalElement)

	t.Code = Cadr(t.Code)
	t.Scratch = Cdr(t.Scratch)

	return true
}

func (t *Task) Lookup(sym *Symbol, simple bool) (bool, string) {
	c := Resolve(t.Lexical, t.Dynamic, sym)
	if c == nil {
		r := Raw(sym)
		if t.Strict() && !number(r) {
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

func (self *Task) Runnable() bool {
	return !<-self.suspended
}

func (self *Task) Stop() {
	self.Stack = Null
	close(self.Eval)

	select {
	case <-self.suspended:
	default:
		close(self.suspended)
	}

	if self.pid > 0 {
		syscall.Kill(self.pid, syscall.SIGTERM)
	}

	for k, v := range self.children {
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

func (self *Task) Suspend() {
	//	if self.pid > 0 {
	//		syscall.Kill(self.pid, syscall.SIGSTOP)
	//	}

	for k, v := range self.children {
		if v {
			k.Suspend()
		}
	}

	self.suspended = make(chan bool)
}

func (self *Task) Wait() {
	for k, v := range self.children {
		if v {
			<-k.Done
		}
		delete(self.children, k)
	}
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

func (self *Scope) Bool() bool {
	return true
}

func (self *Scope) String() string {
	return fmt.Sprintf("%%scope %p%%", self)
}

func (self *Scope) Equal(c Cell) bool {
	return self == c
}

/* Scope-specific functions */

func (self *Scope) Access(key Cell) Reference {
	var obj Context
	for obj = self; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (self *Scope) Complete(line, prefix string) []string {
	cl := []string{}

	var obj Context
	for obj = self; obj != nil; obj = obj.Prev() {
		cl = append(cl, obj.Faces().Complete(line, prefix)...)
	}

	return cl
}

func (self *Scope) Copy() Context {
	return &Scope{self.env.Copy(), self.prev}
}

func (self *Scope) Expose() Context {
	return self
}

func (self *Scope) Faces() *Env {
	return self.env
}

func (self *Scope) Prev() Context {
	return self.prev
}

func (self *Scope) Define(key Cell, value Cell) {
	self.env.Add(key, value)
}

func (self *Scope) Public(key Cell, value Cell) {
	self.env.prev.Add(key, value)
}

func (self *Scope) Remove(key Cell) bool {
	if !self.env.prev.Remove(key) {
		return self.env.Remove(key)
	}

	return true
}

func (self *Scope) DefineBuiltin(k string, a Function) {
	self.Define(NewSymbol(k),
		NewUnbound(NewBuiltin(a, Null, Null, Null, self)))
}

func (self *Scope) DefineMethod(k string, a Function) {
	self.Define(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, self), self))
}

func (self *Scope) PublicMethod(k string, a Function) {
	self.Public(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, self), self))
}

func (self *Scope) DefineSyntax(k string, a Function) {
	self.Define(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, self), self))
}

func (self *Scope) PublicSyntax(k string, a Function) {
	self.Public(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, self), self))
}

/* Bound cell definition. */

type Bound struct {
	ref     Closure
	context Context
}

func NewBound(ref Closure, context Context) *Bound {
	return &Bound{ref, context}
}

func (self *Bound) Bool() bool {
	return true
}

func (self *Bound) String() string {
	return fmt.Sprintf("%%bound %p%%", self)
}

func (self *Bound) Equal(c Cell) bool {
	if m, ok := c.(*Bound); ok {
		return self.ref == m.Ref() && self.context == m.Self()
	}
	return false
}

/* Bound-specific functions */

func (self *Bound) Bind(c Context) Binding {
	if c == self.context {
		return self
	}
	return NewBound(self.ref, c)
}

func (self *Bound) Ref() Closure {
	return self.ref
}

func (self *Bound) Self() Context {
	return self.context
}

/* Unbound cell definition. */

type Unbound struct {
	ref Closure
}

func NewUnbound(Ref Closure) *Unbound {
	return &Unbound{Ref}
}

func (self *Unbound) Bool() bool {
	return true
}

func (self *Unbound) String() string {
	return fmt.Sprintf("%%unbound %p%%", self)
}

func (self *Unbound) Equal(c Cell) bool {
	if u, ok := c.(*Unbound); ok {
		return self.ref == u.Ref()
	}
	return false
}

/* Unbound-specific functions */

func (self *Unbound) Bind(c Context) Binding {
	return self
}

func (self *Unbound) Ref() Closure {
	return self.ref
}

func (self *Unbound) Self() Context {
	return nil
}

/* Variable cell definition. */

type Variable struct {
	v Cell
}

func NewVariable(v Cell) Reference {
	return &Variable{v}
}

func (self *Variable) Bool() bool {
	return true
}

func (self *Variable) String() string {
	return fmt.Sprintf("%%variable %p%%", self)
}

func (self *Variable) Equal(c Cell) bool {
	return self.v.Equal(c)
}

/* Variable-specific functions */

func (self *Variable) Copy() Reference {
	return NewVariable(self.v)
}

func (self *Variable) Get() Cell {
	return self.v
}

func (self *Variable) Set(c Cell) {
	self.v = c
}

/* Constant cell definition. */

type Constant struct {
	Variable
}

func NewConstant(v Cell) *Constant {
	return &Constant{Variable{v}}
}

func (self *Constant) String() string {
	return fmt.Sprintf("%%constant %p%%", self)
}

func (self *Constant) Set(c Cell) {
	panic("constant cannot be set")
}
