// Released under an MIT license. See LICENSE.

// +build solaris

package system

import (
	"golang.org/x/sys/unix"
)

func setForegroundGroup(fd, group int) {
	unix.IoctlSetInt(fd, unix.TIOCSPGRP, group)
}
