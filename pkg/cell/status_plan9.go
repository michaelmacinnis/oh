// Released under an MIT license. See LICENSE.

// +build plan9

package cell

import (
	"fmt"
	"math/big"
	"strconv"
)

type Atom interface {
	Cell

	Float() float64
	Int() int64
	Rat() *big.Rat
	Status() string
}

var (
	ExitFailure *Status
	ExitSuccess *Status
	failure     = "failure"
	success     = ""
)

func init() {
	ExitFailure = NewStatus(failure)
	ExitSuccess = NewStatus(success)
}

func (b *Boolean) Status() string {
	if !b.Bool() {
		return success
	}
	return failure
}

func (f *Float) Status() string {
	if !f.Bool() {
		return success
	}
	return failure
}

func (i *Integer) Status() string {
	if !i.Bool() {
		return success
	}
	return failure
}

func (r Rational) Status() string {
	if !r.Bool() {
		return success
	}
	return failure
}

/* Status cell definition. */

type Status string

func IsStatus(c Cell) bool {
	switch c.(type) {
	case *Status:
		return true
	}
	return false
}

func NewStatus(v string) *Status {
	if v == success {
		return ExitSuccess
	}

	s := Status(v)
	return &s
}

func (s *Status) Bool() bool {
	return string(*s) == success
}

func (s *Status) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(*s) == a.String()
	}
	return false
}

func (s *Status) String() string {
	return string(*s)
}

func (s *Status) Float() float64 {
	f, err := strconv.ParseFloat(string(*s), 64)
	if err != nil {
		panic(err)
	}
	return f
}

func (s *Status) Int() int64 {
	i, err := strconv.ParseInt(string(*s), 0, 64)
	if err != nil {
		panic(err)
	}
	return i
}

func (s *Status) Rat() *big.Rat {
	r := new(big.Rat)
	if _, err := fmt.Sscan(string(*s), r); err != nil {
		panic(err)
	}
	return r
}

func (s *Status) Status() string {
	return string(*s)
}

func (s *String) Status() string {
	if Raw(s) == success {
		return success
	}
	return failure
}

func (s *Symbol) Status() string {
	if Raw(s) == success {
		return success
	}
	return failure
}
