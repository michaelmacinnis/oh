// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/truth"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func not(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(!truth.Value(v[0]))
}
