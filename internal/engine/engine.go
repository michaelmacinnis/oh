// Released under an MIT license. See LICENSE.

// Package engine provides an evaluator for parsed oh code.
package engine

import (
	"github.com/michaelmacinnis/oh/internal/engine/task"
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

// T (engine) is a facade in front of the machinery for evaluating oh code.
type T struct {
	cmd        chan cell.T
	done       chan struct{}
	foreground *task.T
}

// New creates a new T.
func New() *T {
	cmd := make(chan cell.T, 1)
	done := make(chan struct{}, 1)

	return &T{
		cmd:        cmd,
		done:       done,
		foreground: task.Foreground(cmd, done),
	}
}

// Evaluate sends the command c to the foreground process.
func (e *T) Evaluate(c cell.T) {
	e.cmd <- c
	<-e.done
}

// System sends the system command c to the foreground process.
func (e *T) System(c cell.T) {
}
