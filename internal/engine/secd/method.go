// Released under an MIT license. See LICENSE.

package secd

import (
	"github.com/michaelmacinnis/oh/internal/interface/cell"
)

// Method is oh's arguments-evaluated, closure type.
type Method Closure

// The method type is a cell.

// Equal returns true if the cell c is the same method as a.
func (a *Method) Equal(c cell.T) bool {
	p, ok := c.(*Method)
	return ok && p == a
}

// Name returns the name of the method type.
func (a *Method) Name() string {
	return "method"
}

// Methods specific to method.

// Closure returns the method a's underlying closure.
func (a *Method) Closure() *Closure {
	return (*Closure)(a)
}

// Execute sets up the states required to execute the method a.
func (a *Method) Execute(m *Machine) State {
	m.ReplaceState(a.State)
	m.NewState(ExecMethod)
	m.PushResult(nil)
	return m.NewState(EvalArgs)
}

var _ command = &Method{}
