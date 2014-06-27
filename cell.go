/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"fmt"
	"strconv"
	"strings"
)

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

type Function func(t *Task, args Cell) bool

type NewClosure func(a Function, b, l, p Cell, s Context) Closure

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

var Null Cell
var False *Boolean
var True *Boolean

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

func (b *Boolean) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return bool(*b) == a.Bool()
	}
	return false
}

func (b *Boolean) String() string {
	if b == True {
		return "True"
	}
	return "False"
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
	Combiner
}

func NewBuiltin(a Function, b, l, p Cell, s Context) Closure {
	return &Builtin{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
	}
}

func (b *Builtin) Equal(c Cell) bool {
	return b == c
}

func (b *Builtin) String() string {
	return fmt.Sprintf("%%builtin %p%%", b)
}

/* Combiner cell definition. */

type Combiner struct {
	applier Function
	body    Cell
	label   Cell
	params  Cell
	scope   Context
}

func (c *Combiner) Bool() bool {
	return true
}

func (c *Combiner) Applier() Function {
	return c.applier
}

func (c *Combiner) Body() Cell {
	return c.body
}

func (c *Combiner) Params() Cell {
	return c.params
}

func (c *Combiner) Label() Cell {
	return c.label
}

func (c *Combiner) Scope() Context {
	return c.scope
}

/* Constant cell definition. */

type Constant struct {
	Variable
}

func NewConstant(v Cell) *Constant {
	return &Constant{Variable{v}}
}

func (ct *Constant) String() string {
	return fmt.Sprintf("%%ct %p%%", ct)
}

func (ct *Constant) Set(c Cell) {
	panic("constant cannot be set")
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

func (e *Env) Complete(line, prefix string) []string {
	cl := []string{}

	for k, _ := range e.hash {
		if strings.HasPrefix(k, prefix) {
			cl = append(cl, line+k)
		}
	}

	if e.prev != nil {
		cl = append(cl, e.prev.Complete(line, prefix)...)
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
		NewConstant(NewBound(NewMethod(m, Null, Null, Null, nil), nil))
}

func (e *Env) Prev() *Env {
	return e.prev
}

func (e *Env) Remove(key Cell) bool {
	_, ok := e.hash[key.String()]

	delete(e.hash, key.String())

	return ok
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

func (f *Float) String() string {
	return strconv.FormatFloat(float64(*f), 'g', -1, 64)
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

func (i *Integer) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return int64(*i) == a.Int()
	}
	return false
}

func (i *Integer) String() string {
	return strconv.FormatInt(int64(*i), 10)
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

/* Method cell definition. */

type Method struct {
	Combiner
}

func NewMethod(a Function, b, l, p Cell, s Context) Closure {
	return &Method{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
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
	panic("Private members cannot be added to an object.")
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

func (p *Pair) Equal(c Cell) bool {
	if p == Null && c == Null {
		return true
	}
	return p.car.Equal(Car(c)) && p.cdr.Equal(Cdr(c))
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

func (s *Scope) Complete(line, prefix string) []string {
	cl := []string{}

	var obj Context
	for obj = s; obj != nil; obj = obj.Prev() {
		cl = append(cl, obj.Faces().Complete(line, prefix)...)
	}

	return cl
}

func (s *Scope) Copy() Context {
	return &Scope{s.env.Copy(), s.prev}
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

func (s *Scope) Remove(key Cell) bool {
	if !s.env.prev.Remove(key) {
		return s.env.Remove(key)
	}

	return true
}

func (s *Scope) DefineBuiltin(k string, a Function) {
	s.Define(NewSymbol(k),
		NewUnbound(NewBuiltin(a, Null, Null, Null, s)))
}

func (s *Scope) DefineMethod(k string, a Function) {
	s.Define(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, s), s))
}

func (s *Scope) PublicMethod(k string, a Function) {
	s.Public(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, s), s))
}

func (s *Scope) DefineSyntax(k string, a Function) {
	s.Define(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, s), s))
}

func (s *Scope) PublicSyntax(k string, a Function) {
	s.Public(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, s), s))
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

func (s *Status) String() string {
	return strconv.FormatInt(int64(*s), 10)
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

func (s *String) String() string {
	return strconv.Quote(string(*s))
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

func (s *String) Status() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*s), 0, 64); err != nil {
		panic(err)
	}
	return i
}

/* String-specific functions. */

func (s *String) Raw() string {
	return string(*s)
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

func (s *Symbol) String() string {
	return string(*s)
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

func (s *Symbol) Greater(c Cell) bool {
	return string(*s) > c.(Atom).String()
}

func (s *Symbol) Less(c Cell) bool {
	return string(*s) < c.(Atom).String()
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

/* Symbol-specific functions. */

func (s *Symbol) isFloat() bool {
	_, err := strconv.ParseFloat(string(*s), 64)
	return err == nil
}

func (s *Symbol) isInt() bool {
	_, err := strconv.ParseInt(string(*s), 0, 64)
	return err == nil
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

func (m *Syntax) Equal(c Cell) bool {
	return m == c
}

func (m *Syntax) String() string {
	return fmt.Sprintf("%%syntax %p%%", m)
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

/* Variable cell definition. */

type Variable struct {
	v Cell
}

func NewVariable(v Cell) Reference {
	return &Variable{v}
}

func (vr *Variable) Bool() bool {
	return true
}

func (vr *Variable) Equal(c Cell) bool {
	return vr.v.Equal(c)
}

func (vr *Variable) String() string {
	return fmt.Sprintf("%%variable %p%%", vr)
}

/* Variable-specific functions */

func (vr *Variable) Copy() Reference {
	return NewVariable(vr.v)
}

func (vr *Variable) Get() Cell {
	return vr.v
}

func (vr *Variable) Set(c Cell) {
	vr.v = c
}

