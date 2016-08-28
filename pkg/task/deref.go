// Released under an MIT license. See LICENSE.

package task

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"unsafe"
)

func Deref(name string, address uintptr) Cell {
	switch {
	case name == "channel":
		return (*Channel)(unsafe.Pointer(address))
	case name == "pipe":
		return (*Pipe)(unsafe.Pointer(address))
	case name == "task":
		return (*Task)(unsafe.Pointer(address))

	case name == "bound":
		return (*bound)(unsafe.Pointer(address))
	case name == "builtin":
		return (*builtin)(unsafe.Pointer(address))
	case name == "continuation":
		return (*continuation)(unsafe.Pointer(address))
	case name == "method":
		return (*method)(unsafe.Pointer(address))
	case name == "object":
		return (*object)(unsafe.Pointer(address))
	case name == "scope":
		return (*scope)(unsafe.Pointer(address))
	case name == "syntax":
		return (*syntax)(unsafe.Pointer(address))
	case name == "unbound":
		return (*unbound)(unsafe.Pointer(address))
	}

	return Null
}
