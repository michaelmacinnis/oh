// Released under an MIT license. See LICENSE.

package task

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// Syntax is oh's arguments-not-evaluated, closure type.
type Syntax Closure

// The syntax type is a cell.

// Equal returns true if the cell c is the same syntax as a.
func (a *Syntax) Equal(c cell.I) bool {
	p, ok := c.(*Syntax)
	return ok && p == a
}

// Name returns the name of the syntax type.
func (a *Syntax) Name() string {
	return "syntax"
}

// Methods specific to syntax.

// Closure returns the syntax a's underlying closure.
func (a *Syntax) Closure() *Closure {
	return (*Closure)(a)
}

// Execute sets up the operations required to execute the syntax a.
func (a *Syntax) Execute(t *T) Op {
	return t.ReplaceOp(a.Op)
}
