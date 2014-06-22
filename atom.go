/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"strconv"
)

type Atom interface {
	Cell

	Float() float64
	Int() int64
	Status() int64
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

func IsAtom(c Cell) bool {
	switch c.(type) {
	case Atom:
		return true
	}

	return false
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

