// Released under an MIT license. See LICENSE.

package commands

import (
	"strings"

	"github.com/michaelmacinnis/adapted"
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/boolean"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/interface/scope"
	"github.com/michaelmacinnis/oh/internal/common/type/create"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/status"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func debug(args cell.I) cell.I {
	println(literal.String(args))

	return sym.True
}

func isObject(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return create.Bool(scope.Is(v[0]))
}

func makeBool(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return create.Bool(boolean.Value(v[0]))
}

func makeStatus(args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	return status.Rat(rational.Number(v[0]))
}

func match(args cell.I) cell.I {
	v := validate.Fixed(args, 2, 2)

	ok, err := adapted.Match(common.String(v[0]), common.String(v[1]))
	if err != nil {
		panic(err.Error())
	}

	return create.Bool(ok)
}

func mend(args cell.I) cell.I {
	v, rest := validate.Variadic(args, 2, 2)

	sep := common.String(v[0])
	c := v[1]

	var create func(string) cell.I = sym.New
	if str.Is(c) {
		create = str.New
	}

	var joined strings.Builder

	joined.WriteString(common.String(c))

	for rest != pair.Null {
		joined.WriteString(sep)

		c = pair.Car(rest)
		if str.Is(c) {
			create = str.New
		}

		joined.WriteString(common.String(c))

		rest = pair.Cdr(rest)
	}

	return create(joined.String())
}

func rend(args cell.I) cell.I {
	sep := pair.Car(args)
	s := pair.Cadr(args)

	create := sym.New
	if _, ok := s.(*str.T); ok {
		create = str.New
	}

	split := strings.Split(common.String(s), common.String(sep))

	res := make([]cell.I, len(split))
	for i, v := range split {
		res[i] = create(v)
	}

	return list.New(res...)
}
