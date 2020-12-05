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
		"add":                add,
		"boolean":            makeBoolean,
		"boolean?":           isBoolean,
		"channel":            makeChannel,
		"channel?":           isChannel,
		"cons":               cons,
		"cons?":              isCons,
		"debug":              debug,
		"div":                div,
		"equal?":             equal,
		"ge?":                ge,
		"gt?":                gt,
		"le?":                le,
		"lt?":                lt,
		"match":              match,
		"mend":               mend,
		"mod":                mod,
		"mul":                mul,
		"not":                not,
		"null?":              isNull,
		"number":             number,
		"number?":            isNumber,
		"object?":            isObject,
		"open":               open,
		"pipe":               makePipe,
		"pipe?":              isPipe,
		"rend":               rend,
		"sprintf":            sprintf,
		"string":             makeString,
		"string?":            isString,
		"string-replace":     sreplace,
		"string-trim-prefix": trimPrefix,
		"string-trim-suffix": trimSuffix,
		"symbol":             makeSymbol,
		"symbol?":            isSymbol,
		"sub":                sub,
		"temp-fifo":          tempfifo,
		"umask":              umask,
	}
}
