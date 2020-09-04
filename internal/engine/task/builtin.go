// Released under an MIT license. See LICENSE.

package task

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// Builtin is oh's arguments-evaluated, globs-expanded, closure type.
type Builtin Closure

// The builtin type is a cell.

// Equal returns true if the cell c is the same builtin as m.
func (a *Builtin) Equal(c cell.I) bool {
	p, ok := c.(*Builtin)
	return ok && p == a
}

// Name returns the name of the builtin type.
func (*Builtin) Name() string {
	return "builtin"
}

// Methods specific to builtin.

// Closure returns the builtin a's underlying closure.
func (a *Builtin) Closure() *Closure {
	return (*Closure)(a)
}

// Execute sets up the operations required to execute the builtin a.
func (a *Builtin) Execute(t *T) Op {
	t.ReplaceOp(a.Op)
	t.PushOp(Action(execBuiltin))
	t.PushResult(nil)

	return t.PushOp(Action(evalArgs))
}
