// Released under an MIT license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd

package system

import (
	"syscall"
	"unsafe"
)

func SetForegroundGroup(group int) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&group)))
}
