// Released under an MIT license. See LICENSE.

package task

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// Method is oh's arguments-evaluated, closure type.
type Method Closure

// The method type is a cell.

// Equal returns true if the cell c is the same method as a.
func (a *Method) Equal(c cell.I) bool {
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

// Execute sets up the operations required to execute the method a.
func (a *Method) Execute(t *T) Op {
	t.ReplaceOp(a.Op)
	t.PushOp(Action(execMethod))
	t.PushResult(nil)

	return t.PushOp(Action(evalArgs))
}

var _ command = &Method{}
