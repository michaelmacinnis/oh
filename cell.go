/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Function func(t *Task, args Cell) bool

type NewCombiner func(a Function, b, l, p Cell, s Context) Closure

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
	SaveCode = SaveCarCode | SaveCdrCode
)

var Null Cell
var False *Boolean
var True *Boolean

var conduit_env *Env

var num [512]*Integer
var res [256]*Status
var str map[string]*String
var sym map[string]*Symbol

func init() {
	pair := new(Pair)
	pair.car = pair
	pair.cdr = pair

	Null = Cell(pair)

	F := Boolean(false)
	False = &F

	T := Boolean(true)
	True = &T

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

func (self *Boolean) Bool() bool {
	return self == True
}

func (self *Boolean) Float() float64 {
	if self == True {
		return 1.0
	}
	return 0.0
}

func (self *Boolean) Int() int64 {
	if self == True {
		return 1
	}
	return 0
}

func (self *Boolean) Status() int64 {
	if self == True {
		return 0
	}
	return 1
}

func (self *Boolean) String() string {
	if self == True {
		return "True"
	}
	return "False"
}

func (self *Boolean) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return bool(*self) == a.Bool()
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

func (self *Integer) Bool() bool {
	return *self != 0
}

func (self *Integer) Float() float64 {
	return float64(*self)
}

func (self *Integer) Int() int64 {
	return int64(*self)
}

func (self *Integer) Status() int64 {
	return int64(*self)
}

func (self *Integer) String() string {
	return strconv.FormatInt(int64(*self), 10)
}

func (self *Integer) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return int64(*self) == a.Int()
	}
	return false
}

func (self *Integer) Greater(c Cell) bool {
	return int64(*self) > c.(Atom).Int()
}

func (self *Integer) Less(c Cell) bool {
	return int64(*self) < c.(Atom).Int()
}

func (self *Integer) Add(c Cell) Number {
	return NewInteger(int64(*self) + c.(Atom).Int())
}

func (self *Integer) Divide(c Cell) Number {
	return NewInteger(int64(*self) / c.(Atom).Int())
}

func (self *Integer) Modulo(c Cell) Number {
	return NewInteger(int64(*self) % c.(Atom).Int())
}

func (self *Integer) Multiply(c Cell) Number {
	return NewInteger(int64(*self) * c.(Atom).Int())
}

func (self *Integer) Subtract(c Cell) Number {
	return NewInteger(int64(*self) - c.(Atom).Int())
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

func (self *Status) Bool() bool {
	return int64(*self) == 0
}

func (self *Status) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return int64(*self) == a.Status()
	}
	return false
}

func (self *Status) Float() float64 {
	return float64(*self)
}

func (self *Status) Int() int64 {
	return int64(*self)
}

func (self *Status) Status() int64 {
	return int64(*self)
}

func (self *Status) String() string {
	return strconv.FormatInt(int64(*self), 10)
}

func (self *Status) Greater(c Cell) bool {
	return int64(*self) > c.(Atom).Status()
}

func (self *Status) Less(c Cell) bool {
	return int64(*self) < c.(Atom).Status()
}

func (self *Status) Add(c Cell) Number {
	return NewStatus(int64(*self) + c.(Atom).Status())
}

func (self *Status) Divide(c Cell) Number {
	return NewStatus(int64(*self) / c.(Atom).Status())
}

func (self *Status) Modulo(c Cell) Number {
	return NewStatus(int64(*self) % c.(Atom).Status())
}

func (self *Status) Multiply(c Cell) Number {
	return NewStatus(int64(*self) * c.(Atom).Status())
}

func (self *Status) Subtract(c Cell) Number {
	return NewStatus(int64(*self) - c.(Atom).Status())
}

/* Float cell definition. */

type Float float64

func NewFloat(v float64) *Float {
	f := Float(v)
	return &f
}

func (self *Float) Bool() bool {
	return *self != 0
}

func (self *Float) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return float64(*self) == a.Float()
	}
	return false
}

func (self *Float) Float() float64 {
	return float64(*self)
}

func (self *Float) Int() int64 {
	return int64(*self)
}

func (self *Float) Status() int64 {
	return int64(*self)
}

func (self *Float) String() string {
	return strconv.FormatFloat(float64(*self), 'g', -1, 64)
}

func (self *Float) Greater(c Cell) bool {
	return float64(*self) > c.(Atom).Float()
}

func (self *Float) Less(c Cell) bool {
	return float64(*self) < c.(Atom).Float()
}

func (self *Float) Add(c Cell) Number {
	return NewFloat(float64(*self) + c.(Atom).Float())
}

func (self *Float) Divide(c Cell) Number {
	return NewFloat(float64(*self) / c.(Atom).Float())
}

func (self *Float) Modulo(c Cell) Number {
	panic("Type 'float' does not implement 'modulo'.")
}

func (self *Float) Multiply(c Cell) Number {
	return NewFloat(float64(*self) * c.(Atom).Float())
}

func (self *Float) Subtract(c Cell) Number {
	return NewFloat(float64(*self) - c.(Atom).Float())
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

func (self *Symbol) Bool() bool {
	if string(*self) == "False" {
		return false
	}

	return true
}

func (self *Symbol) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(*self) == a.String()
	}
	return false
}

func (self *Symbol) Float() (f float64) {
	var err error
	if f, err = strconv.ParseFloat(string(*self), 64); err != nil {
		panic(err)
	}
	return f
}

func (self *Symbol) Int() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*self), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (self *Symbol) Status() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*self), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (self *Symbol) String() string {
	return string(*self)
}

func (self *Symbol) Greater(c Cell) bool {
	return string(*self) > c.(Atom).String()
}

func (self *Symbol) Less(c Cell) bool {
	return string(*self) < c.(Atom).String()
}

func (self *Symbol) isFloat() bool {
	_, err := strconv.ParseFloat(string(*self), 64)
	return err == nil
}

func (self *Symbol) isInt() bool {
	_, err := strconv.ParseInt(string(*self), 0, 64)
	return err == nil
}

func (self *Symbol) Add(c Cell) Number {
	if self.isInt() {
		return NewInteger(self.Int() + c.(Atom).Int())
	} else if self.isFloat() {
		return NewFloat(self.Float() + c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'add'.")
}

func (self *Symbol) Divide(c Cell) Number {
	if self.isInt() {
		return NewInteger(self.Int() / c.(Atom).Int())
	} else if self.isFloat() {
		return NewFloat(self.Float() / c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'divide'.")
}

func (self *Symbol) Modulo(c Cell) Number {
	if self.isInt() {
		return NewInteger(self.Int() % c.(Atom).Int())
	}

	panic("Type 'symbol' does not implement 'modulo'.")
}

func (self *Symbol) Multiply(c Cell) Number {
	if self.isInt() {
		return NewInteger(self.Int() * c.(Atom).Int())
	} else if self.isFloat() {
		return NewFloat(self.Float() * c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'multiply'.")
}

func (self *Symbol) Subtract(c Cell) Number {
	if self.isInt() {
		return NewInteger(self.Int() - c.(Atom).Int())
	} else if self.isFloat() {
		return NewFloat(self.Float() - c.(Atom).Float())
	}

	panic("Type 'symbol' does not implement 'subtract'.")
}

/* String cell definition. */

type String string

func NewString(q string) *String {
	v, _ := strconv.Unquote("\"" + q + "\"")

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

func (self *String) Bool() bool {
	return true
}

func (self *String) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(*self) == a.String()
	}
	return false
}

func (self *String) Float() (f float64) {
	var err error
	if f, err = strconv.ParseFloat(string(*self), 64); err != nil {
		panic(err)
	}
	return f
}

func (self *String) Int() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*self), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (self *String) Raw() string {
	return string(*self)
}

func (self *String) Status() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*self), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (self *String) String() string {
	return strconv.Quote(string(*self))
}

/* Pair cell definition. */

type Pair struct {
	car Cell
	cdr Cell
}

func Cons(h, t Cell) Cell {
	return &Pair{car: h, cdr: t}
}

func (self *Pair) Bool() bool {
	return self != Null
}

func (self *Pair) String() (s string) {
	s = ""

	if IsCons(self.car) && IsCons(Cdr(self.car)) {
		s += "("
	}

	if self.car != Null {
		s += self.car.String()
	}

	if IsCons(self.car) && IsCons(Cdr(self.car)) {
		s += ")"
	}

	if IsCons(self.cdr) {
		if self.cdr == Null {
			return s
		}

		s += " "
	} else {
		s += "::"
	}

	s += self.cdr.String()

	return s
}

func (self *Pair) Equal(c Cell) bool {
	if self == Null && c == Null {
		return true
	}
	return self.car.Equal(Car(c)) && self.cdr.Equal(Cdr(c))
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
			if cli != nil && self.r == os.Stdin {
				cli.Reset()
			}
			Parse(self.reader(), func(c Cell) {
				self.c <- c
				if cli != nil && self.r == os.Stdin {
					cli.Set()
				}
				<-self.d
				if cli != nil && self.r == os.Stdin {
					cli.Reset()
				}
			})
			self.c <- Null
		}()
	} else {
		self.d <- true
	}

	return <-self.c
}

func (self *Pipe) ReadLine() Cell {
	if cli != nil && self.r == os.Stdin {
		cli.Reset()
	}
	s, err := self.reader().ReadString('\n')
	if cli != nil && self.r == os.Stdin {
		cli.Set()
	}
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

/* Task cell definition. */

type Task struct {
	*Registers
	Done  chan Cell
	Eval  chan Cell
	Child []*Task
}

func NewTask(state int64, code Cell, dynamic *Env, lexical Context) *Task {
	t := &Task{
		Registers: &Registers{
			Continuation: Continuation{
				Scratch: List(NewStatus(0)),
				Stack:   List(NewInteger(state)),
			},
			Code:    code,
			Dynamic: dynamic,
			Lexical: lexical,
		},
		Done:  make(chan Cell, 1),
		Eval:  make(chan Cell, 1),
		Child: nil,
	}

	return t
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

func (self *Task) Running() bool {
	select {
	case <-self.Done:
		return false
	default:
	}
	return true
}

func (self *Task) Start() {
	if self.Done == nil {
		self.Done = make(chan Cell, 1)
	}
}

func (self *Task) Stop() {
	close(self.Done)
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
