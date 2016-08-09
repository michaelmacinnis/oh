// Released under an MIT license. See LICENSE.

// +build solaris

package system

import (
	"golang.org/x/sys/unix"
)

func SetForegroundGroup(group int) {
	unix.IoctlSetInt(unix.Stdin, sys.TIOCSPGRP, group)
}

