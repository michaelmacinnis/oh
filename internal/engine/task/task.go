// Released under an MIT license. See LICENSE.

// Package task provides the Task object which encapsulate a thread of execution for the oh language.
package task

import (
	"github.com/michaelmacinnis/oh/internal/engine/secd"
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/type/pair"
)

type T struct {
	ancestor *T
	children map[*T]struct{}

	irq chan secd.State

	//job *job.Job
	//pid int
	registers
}

type registers = *secd.Machine

// Global block allows continuations to work at the top level.
var (
	block0 = nilBlock()
)

// Foreground launches and returns a new foreground task.
//
// It creates a pair of actions that replace each other on the stack.
// The first waits for a command and executes it. The second signals
// that the command has been executed.
//
// The nil command serves as a sentinal value in EvalBlock.
//
// The head (nil) will be replaced with the command received and the
// tail will be modified to point to a new nil block.
func Foreground(cmd chan cell.T, done chan struct{}) *T {
	t := New(block0)

	var expect, notify secd.Action

	expect = func(m *secd.Machine) secd.State {
		c := <-cmd

		pair.SetCar(block0, c)
		pair.SetCdr(block0, nilBlock())

		m.ReplaceState(notify)

		return m.NewState(secd.EvalBlock)
	}

	notify = func(m *secd.Machine) secd.State {
		// TODO: If there was an error we could skip this step.
		// In addition to overwriting the instruction this would
		// overwrite the nil block with another nil block.
		block0 = pair.Cdr(block0)

		done <- struct{}{}

		return m.ReplaceState(expect)
	}

	go t.Run(expect)

	return t
}

func New(code cell.T) *T {
	t := &T{
		registers: secd.New(code),
		irq:       make(chan secd.State),
	}

	return t
}

func (t *T) Run(s secd.State) {
	t.NewState(s)

	for s != nil {
		select {
		case s = <-t.irq:
			t.NewState(s)
		default:
		}

		s = t.Step(s)
	}
}

func nilBlock() cell.T {
	return pair.Cons(nil, pair.Null)
}
