/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"fmt"
)

type Reference interface {
	Cell

	Copy() Reference
	Get() Cell
	Set(c Cell)
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

func (vr *Variable) String() string {
	return fmt.Sprintf("%%variable %p%%", vr)
}

func (vr *Variable) Equal(c Cell) bool {
	return vr.v.Equal(c)
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
	panic("ct cannot be set")
}

