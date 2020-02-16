// Released under an MIT license. See LICENSE.

package secd

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

// Builtin is oh's arguments-evaluated, globs-expanded, closure type.
type Builtin Closure

// The builtin type is a cell.

// Equal returns true if the cell c is the same builtin as m.
func (m *Builtin) Equal(c cell.T) bool {
	p, ok := c.(*Builtin)
	return ok && p == m
}

// Name returns the name of the builtin type.
func (m *Builtin) Name() string {
	return "builtin"
}

// Methods specific to builtin.

// Closure returns the builtin a's underlying closure.
func (a *Builtin) Closure() *Closure {
        return (*Closure)(a)
}

// Execute sets up the states required to execute the builtin a.
func (a *Builtin) Execute(m *Machine) State {
        m.ReplaceState(a.State)
        m.NewState(ExecBuiltin)
	m.PushResult(nil)
        return m.NewState(EvalArgs)
}
