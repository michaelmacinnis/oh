// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func eq(args cell.I) cell.I {
	v, rest := validate.Variadic(args, 2, 2)

	if !v[0].Equal(v[1]) {
		return boolean.False
	}

	for rest != pair.Null {
		if !v[0].Equal(pair.Car(rest)) {
			return boolean.False
		}

		rest = pair.Cdr(rest)
	}

	return boolean.True
}

func lt(args cell.I) cell.I {
	prev := rational.Number(pair.Car(args))

	for args := pair.Cdr(args); args != pair.Null; args = pair.Cdr(args) {
		curr := rational.Number(pair.Car(args))

		if prev.Cmp(curr) >= 0 {
			return boolean.False
		}
	}

	return boolean.True
}
