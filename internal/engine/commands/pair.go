// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func cons(args cell.I) cell.I {
	v := validate.Fixed(args, 2, 2)

	return pair.Cons(v[0], v[1])
}

func isCons(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(pair.Is(v[0]))
}

func isNull(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(v[0] == pair.Null)
}
