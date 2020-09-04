// Released under an MIT license. See LICENSE.

package commands

import (
	"strings"

	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
)

func _join_(args cell.I) cell.I { //nolint:golint
	var create func(string) cell.I = sym.New

	var joined strings.Builder

	for args != pair.Null {
		c := pair.Car(args)

		switch c := c.(type) {
		case *str.T:
			create = str.New

			joined.WriteString(c.String())
		case *sym.Plus:
			joined.WriteString(c.String())
		case *sym.T:
			joined.WriteString(c.String())
		default:
			panic("only strings and symbols can be joined")
		}

		args = pair.Cdr(args)
	}

	return create(joined.String())
}

func _split_(args cell.I) cell.I { //nolint:golint
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

func debug(args cell.I) cell.I {
	println(literal.String(args))

	return boolean.True
}
