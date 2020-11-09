// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/pipe"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func isPipe(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(pipe.Is(v[0]))
}

func makePipe(args cell.I) cell.I {
	validate.Fixed(args, 0, 0)

	return pipe.New(nil, nil)
}
