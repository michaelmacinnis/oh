// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func isSymbol(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(sym.Is(v[0]))
}

func makeSymbol(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return sym.New(common.String(v[0]))
}
