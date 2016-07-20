// Released under an MIT license. See LICENSE.

// +build !plan9

package cell

import (
	"math/big"
	"strconv"
	"sync"
)

type Atom interface {
	Cell

	Float() float64
	Int() int64
	Rat() *big.Rat
	Status() int64
}

var (
	ExitFailure *Status
	ExitSuccess *Status
	res         [256]*Status
	resl        = &sync.RWMutex{}
)

func init() {
	ExitFailure = NewStatus(1)
	ExitSuccess = NewStatus(0)
}

func (b *Boolean) Status() int64 {
	if b == True {
		return 0
	}
	return 1
}

func (f *Float) Status() int64 {
	return f.Int()
}

func (i *Integer) Status() int64 {
	return i.Int()
}

func (r Rational) Status() int64 {
	return r.Int()
}

/* Status cell definition. */

type Status int64

func IsStatus(c Cell) bool {
	switch c.(type) {
	case *Status:
		return true
	}
	return false
}

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

func (s *Status) Rat() *big.Rat {
	return big.NewRat(int64(*s), 1)
}

func (s *Status) Status() int64 {
	return s.Int()
}

func (s *Status) Greater(c Cell) bool {
	return s.Rat().Cmp(c.(Atom).Rat()) > 0
}

func (s *Status) Less(c Cell) bool {
	return s.Rat().Cmp(c.(Atom).Rat()) < 0
}

func (s *Status) Add(c Cell) Number {
	return NewRational(new(big.Rat).Add(s.Rat(), c.(Atom).Rat()))
}

func (s *Status) Divide(c Cell) Number {
	return NewRational(new(big.Rat).Quo(s.Rat(), c.(Atom).Rat()))
}

func (s *Status) Modulo(c Cell) Number {
	return NewRational(ratmod(s.Rat(), c.(Atom).Rat()))
}

func (s *Status) Multiply(c Cell) Number {
	return NewRational(new(big.Rat).Mul(s.Rat(), c.(Atom).Rat()))
}

func (s *Status) Subtract(c Cell) Number {
	return NewRational(new(big.Rat).Sub(s.Rat(), c.(Atom).Rat()))
}

func (s *String) Status() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(s.v), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *Symbol) Status() (i int64) {
	return s.Int()
}
