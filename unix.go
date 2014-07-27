// Released under an MIT-style license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd

package main

// TODO: solaris should also be in the list above.

import "syscall"

//#include <signal.h>
//#include <unistd.h>
//void ignore(void) {
//      signal(SIGTTOU, SIG_IGN);
//      signal(SIGTTIN, SIG_IGN);
//}
import "C"

func InitSignalHandling() {
	C.ignore()
}

func InputIsTTY() bool {
	return C.isatty(C.int(0)) != 0
}

func JobControlSupported() bool {
	return true
}

func SetProcessGroup() int {
	pid := syscall.Getpid()
	pgid := syscall.Getpgrp()
	if pid != pgid {
		syscall.Setpgid(0, 0)
	}

	return pid
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
