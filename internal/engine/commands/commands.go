// Released under an MIT license. See LICENSE.

package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// Functions returns a mapping of names to 'methods' that do not reference self.
func Functions() map[string]func(cell.I) cell.I {
	return map[string]func(cell.I) cell.I{
		"add":       add,
		"bool":      makeBool,
		"chan":      makeChan,
		"chan?":     isChan,
		"cons":      cons,
		"cons?":     isCons,
		"debug":     debug,
		"div":       div,
		"equal?":    equal,
		"ge?":       ge,
		"gt?":       gt,
		"le?":       le,
		"lt?":       lt,
		"match":     match,
		"mend":      mend,
		"mod":       mod,
		"mul":       mul,
		"not":       not,
		"null?":     isNull,
		"number":    number,
		"number?":   isNumber,
		"object?":   isObject,
		"open":      open,
		"pipe":      makePipe,
		"pipe?":     isPipe,
		"random":    random,
		"rend":      rend,
		"sprintf":   sprintf,
		"status":    makeStatus,
		"string":    makeString,
		"string?":   isString,
		"symbol":    makeSymbol,
		"symbol?":   isSymbol,
		"sub":       sub,
		"temp-fifo": tempfifo,
		"umask":     umask,
	}
}
