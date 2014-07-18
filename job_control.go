// Released under an MIT-style license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd

package main

import "syscall"

func JobControlSupported() bool {
	return true
}

func SysProcAttr(group int) *syscall.SysProcAttr{
	sys := &syscall.SysProcAttr{}

	sys.Sigdfl = []syscall.Signal{syscall.SIGTTIN, syscall.SIGTTOU}

	if group == 0 {
		sys.Setpgid = true
		sys.Foreground = true
	} else {
		sys.Joinpgrp = group
	}

	return sys
}
