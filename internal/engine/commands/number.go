// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/num"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func isNumber(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(num.Is(v[0]))
}

func number(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return num.Rat(rational.Number(v[0]))
}
