// Released under an MIT license. See LICENSE.

package cell

import "errors"

const (
	ErrNotExecutable = "oh: 126: error/runtime: "
	ErrNotFound      = "oh: 127: error/runtime: "
	ErrSyntax        = "oh: 1: error/syntax: "
)

var CtrlCPressed = errors.New("ctrl-c pressed")
