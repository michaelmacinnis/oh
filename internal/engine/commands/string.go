// Released under an MIT license. See LICENSE.

package commands

import (
	"fmt"
	"strings"

	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/integer"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func isString(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return boolean.Bool(str.Is(v[0]))
}

func makeString(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return str.New(common.String(v[0]))
}

func sreplace(args cell.I) cell.I {
	v := validate.Fixed(args, 3, 4)

	s := common.String(v[0])
	old := common.String(v[1])
	replacement := common.String(v[2])

	n := -1
	if len(v) == 4 {
		n = int(integer.Value(v[3]))
	}

	return str.New(strings.Replace(s, old, replacement, n))
}

// TODO: Extend oh types to play nicer with fmt and Sprintf.
func sprintf(args cell.I) cell.I {
	v, args := validate.Variadic(args, 1, 1)

	argv := []interface{}{}

	for args != pair.Null {
		argv = append(argv, pair.Car(args))
		args = pair.Cdr(args)
	}

	return str.New(fmt.Sprintf(common.String(v[0]), argv...))
}
