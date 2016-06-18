// Released under an MIT license. See LICENSE.

package task

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"unsafe"
)

func deref(name string, address uintptr) Cell {
	switch {
	case name == "bound":
		return (*Bound)(unsafe.Pointer(address))
	case name == "builtin":
		return (*Builtin)(unsafe.Pointer(address))
	case name == "channel":
		return (*Channel)(unsafe.Pointer(address))
	case name == "constant":
		return (*Constant)(unsafe.Pointer(address))
	case name == "continuation":
		return (*Continuation)(unsafe.Pointer(address))
	case name == "method":
		return (*Method)(unsafe.Pointer(address))
	case name == "object":
		return (*Object)(unsafe.Pointer(address))
	case name == "pipe":
		return (*Pipe)(unsafe.Pointer(address))
	case name == "scope":
		return (*Scope)(unsafe.Pointer(address))
	case name == "syntax":
		return (*Syntax)(unsafe.Pointer(address))
	case name == "task":
		return (*Task)(unsafe.Pointer(address))
	case name == "unbound":
		return (*Unbound)(unsafe.Pointer(address))
	case name == "variable":
		return (*Variable)(unsafe.Pointer(address))
	}

	return Null
}
