// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/integer"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/channel"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func isChannel(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(channel.Is(v[0]))
}

func makeChannel(args cell.I) cell.I {
	v := validate.Fixed(args, 0, 1)

	n := int64(0)
	if len(v) > 0 {
		n = integer.Value(v[0])
	}

	return channel.New(n)
}
