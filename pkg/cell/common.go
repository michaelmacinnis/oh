// Released under an MIT license. See LICENSE.

package cell

import "errors"

type Cell interface {
	Bool() bool
	Equal(c Cell) bool
	String() string
}

type DerefFunc func(string, uintptr) Cell

type InputFunc func(byte) (string, error)

type Parser interface {
	ParseBuffer(bool, string, YieldFunc) bool
	ParseCommands(string, YieldFunc)
	ParsePipe(string, YieldFunc) (int, interface{})
	State(string) (string, string, string)
}

type MakeParserFunc func(InputFunc) Parser

type ThrowFunc func(filename string, lineno int, message string)

type YieldFunc func(Cell) (Cell, bool)

const (
	ErrNotExecutable = "oh: 126: error/runtime: "
	ErrNotFound      = "oh: 127: error/runtime: "
	ErrSyntax        = "oh: 1: error/syntax: "
)

var ErrCtrlCPressed = errors.New("ctrl-c pressed")
