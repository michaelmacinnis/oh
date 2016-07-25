// Released under an MIT license. See LICENSE.

package common

import "errors"

type ReadStringer interface {
	ReadString(delim byte) (line string, err error)
}

type Thrower interface {
	Throw(filename string, lineno int, message string)
	SetFile(filename string)
	SetLine(lineno int)
}

const (
	ErrNotExecutable = "oh: 126: error/runtime: "
	ErrNotFound      = "oh: 127: error/runtime: "
	ErrSyntax        = "oh: 1: error/syntax: "
)

var CtrlCPressed = errors.New("ctrl-c pressed")
