// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/boolean"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/create"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func not(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return create.Bool(!boolean.Value(v[0]))
}
