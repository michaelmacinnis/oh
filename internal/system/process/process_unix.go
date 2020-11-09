// Released under an MIT license. See LICENSE.

// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package process

import (
	"os"

	"golang.org/x/sys/unix"
)

//nolint:gochecknoglobals
var (
	Platform = "unix"

	id       = unix.Getpid()
	group, _ = unix.Getpgid(id)
	terminal = int(os.Stdin.Fd())
)

func BecomeForegroundGroup() {
	for group != ForegroundGroup() {
		unix.Kill(-group, unix.SIGTTIN)
		group, _ = unix.Getpgid(id)
	}

	// signal.Ignore(unix.SIGTTIN, unix.SIGTTOU)

	if id != group {
		unix.Setpgid(id, id)
		group = id
	}

	SetForegroundGroup(group)
}

func Continue(pid int) {
	unix.Kill(pid, unix.SIGCONT)
}

func ForegroundGroup() int {
	group, err := unix.IoctlGetInt(terminal, unix.TIOCGPGRP)
	if err != nil {
		return 0
	}

	return group
}

func Group() int {
	return group
}

func ID() int {
	return id
}

func Interrupt(pid int) {
	unix.Kill(pid, unix.SIGINT)
}

func RestoreForegroundGroup() {
	if group == ForegroundGroup() {
		return
	}

	SetForegroundGroup(group)
}

func SetForegroundGroup(g int) {
	err := unix.IoctlSetPointerInt(terminal, unix.TIOCSPGRP, g)
	if err != nil {
		println(err.Error())
	}
}

func Stop(pid int) {
	unix.Kill(pid, unix.SIGSTOP)
}

func SysProcAttr(foreground bool, group int) *unix.SysProcAttr {
	sys := &unix.SysProcAttr{Foreground: foreground, Setpgid: true}

	if group == 0 {
		sys.Ctty = terminal
	} else {
		sys.Pgid = group
	}

	return sys
}

func Terminate(pid int) {
	unix.Kill(pid, unix.SIGTERM)
}
