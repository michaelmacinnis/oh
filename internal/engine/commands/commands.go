// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

func Builtins() map[string]func(cell.I) cell.I {
	return map[string]func(cell.I) cell.I{
		"exists": exists,
	}
}

func Functions() map[string]func(cell.I) cell.I {
	return map[string]func(cell.I) cell.I{
		"_join_":  _join_,
		"_split_": _split_,
		"add":     add,
		"channel": makeChannel,
		"cons":    cons,
		"debug":   debug,
		"eq":      eq,
		"is-cons": isCons,
		"is-null": isNull,
		"lt":      lt,
		"not":     not,
		"open":    open,
		"pipe":    makePipe,
	}
}
