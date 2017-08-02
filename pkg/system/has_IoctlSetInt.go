// Released under an MIT license. See LICENSE.

// +build solaris

package system

import (
	"golang.org/x/sys/unix"
)

func getForegroundGroup(fd int) int {
	group, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err != nil {
		return 0
	}
	return group
}

func setForegroundGroup(fd, group int) {
	unix.IoctlSetInt(fd, unix.TIOCSPGRP, group)
}
