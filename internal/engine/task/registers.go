// Released under an MIT license. See LICENSE.

package task

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/truth"
	"github.com/michaelmacinnis/oh/internal/common/struct/frame"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

// The registers type holds the state of oh's stack-based abstract machine.
type registers struct {
	*stack
	frame *frame.T
	code  cell.I
	dump  cell.I
}

// Perform copies non-nil fields from m to target.
func (m *registers) Perform(target *T) Op {
	m.restoreOver(target.registers)

	return target.PreviousOp()
}

func (m *registers) Code() cell.I {
	return m.code
}

func (m *registers) Completed() bool {
	return m.stack == done
}

func (m *registers) Equal(c cell.I) bool {
	switch c := c.(type) {
	case *registers:
		return *m == *c
	default:
		return false
	}
}

func (m *registers) Name() string {
	return "continuation"
}

// Op returns the abstract machine's current operation.
func (m *registers) Op() Op {
	return m.stack.op
}

// PopResult removes the top result from dump.
func (m *registers) PopResult() cell.I {
	//println("pop result")
	r := pair.Car(m.dump)
	m.dump = pair.Cdr(m.dump)

	return r
}

// PushOp pushes a new operation onto the stack.
func (m *registers) PushOp(s Op) Op {
	/*
		if a, ok := s.(Action); ok {
			println("PushOp(" + funcName(a) + ")")
		} else {
			println("PushOp(SaveOp)")
		}
	*/
	current := toRegisters(s)
	previous := toRegisters(m.stack.op)

	if current != nil && previous != nil {
		// Condense restore operations.
		previous.restoreOver(current)
		m.stack.op = current
	} else {
		m.stack = &stack{m.stack, s}
	}

	return s
}

// PushResult adds the result r to dump.
func (m *registers) PushResult(r cell.I) {
	//println("push result")
	m.dump = pair.Cons(r, m.dump)
}

// PreviousOp pops the current operation and returns the previous operation.
func (m *registers) PreviousOp() Op {
	m.RemoveOp()
	return m.Op()
}

// RemoveOp pops the current operation off the stack.
func (m *registers) RemoveOp() {
	/*
		if a, ok := m.stack.op.(Action); ok {
			println("RemoveOp("+funcName(a)+")")
		} else {
			println("RemoveOp(SaveOp)")
		}
	*/
	m.stack = m.stack.stack
}

// ReplaceOp replaces the operation at the top of the stack.
func (m *registers) ReplaceOp(s Op) Op {
	m.RemoveOp()
	return m.PushOp(s)
}

// ReplaceResult replaced the current result.
func (m *registers) ReplaceResult(r cell.I) {
	//println("replace result")
	m.dump = pair.Cons(r, pair.Cdr(m.dump))
}

// Result returns the current result.
func (m *registers) Result() cell.I {
	return pair.Car(m.dump)
}

// The stack type is a machine's execution stack.
type stack struct {
	*stack
	op Op
}

//nolint:gochecknoglobals
var (
	done = &stack{}
)

func (m *registers) arguments() cell.I {
	e := m.PopResult()
	l := pair.Null

	for e != nil && m.dump != pair.Null {
		l = pair.Cons(e, l)

		e = m.PopResult()
	}

	return l
}

func (m *registers) expand(l cell.I) cell.I {
	// TODO: Actually do expansion.
	return l
}

func (m *registers) restoreOver(target *registers) {
	if m.frame != nil {
		target.frame = m.frame
	}

	if m.code != nil {
		target.code = m.code
	}

	if m.dump != nil {
		target.dump = m.dump
	}

	if m.stack != nil {
		target.stack = m.stack
	}
}

func (m *registers) selectBranch() bool {
	if !truth.Value(m.Result()) {
		m.code = pair.Cdr(m.code)

		c := pair.Car(m.code)
		for pair.Is(c) && c != pair.Null {
			m.code = pair.Cdr(m.code)
			c = pair.Car(m.code)
		}

		if c != pair.Null && literal.String(c) != "else" {
			panic("expected else")
		}
	}

	return pair.Cdr(m.code) != pair.Null
}

func init() { //nolint:gochecknoinits
	done.stack = done
}

func toRegisters(s Op) *registers {
	if r, ok := s.(*registers); ok {
		return r
	}

	return nil
}
