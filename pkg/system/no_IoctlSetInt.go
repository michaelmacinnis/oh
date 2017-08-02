// Released under an MIT license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd

package system

import (
	"syscall"
	"unsafe"
)

func getForegroundGroup(fd int) (group int) {
	syscall.Syscall(
		syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGPGRP,
		uintptr(unsafe.Pointer(&group)),
	)
	return
}

func setForegroundGroup(fd, group int) {
	syscall.Syscall(
		syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSPGRP,
		uintptr(unsafe.Pointer(&group)),
	)
}
