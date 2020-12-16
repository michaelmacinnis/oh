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

	// Umask sets and returns the current umask.
	Umask = unix.Umask

	id       = unix.Getpid()
	group, _ = unix.Getpgid(id)
	terminal = int(os.Stdin.Fd())
)

// BecomeForegroundGroup performs the Unix incantations necessary to put the current process in the foreground.
func BecomeForegroundGroup() (err error) {
	for group != ForegroundGroup() {
		err = unix.Kill(-group, unix.SIGTTIN)
		if err != nil {
			return
		}

		group, err = unix.Getpgid(id)
		if err != nil {
			return
		}
	}

	if id != group {
		err = unix.Setpgid(id, id)
		if err != nil {
			return
		}

		group = id
	}

	SetForegroundGroup(group)

	return
}

// Continue send a SIGCONT to the process ID pid.
func Continue(pid int) {
	_ = unix.Kill(pid, unix.SIGCONT)
}

// ForegroundGroup returns the current foreground group ID.
func ForegroundGroup() int {
	group, err := unix.IoctlGetInt(terminal, unix.TIOCGPGRP)
	if err != nil {
		return 0
	}

	return group
}

// Group returns the group ID for the current process.
func Group() int {
	return group
}

// ID returns the process ID for the current process.
func ID() int {
	return id
}

// Interrupt sends a SIGINT to the process ID pid.
func Interrupt(pid int) {
	_ = unix.Kill(pid, unix.SIGINT)
}

// RestoreForegroundGroup places the group for this process back in the foreground.
func RestoreForegroundGroup() {
	if group == ForegroundGroup() {
		return
	}

	SetForegroundGroup(group)
}

// SetForegroundGroup sets the terminal's foregeound group to g.
func SetForegroundGroup(g int) {
	err := unix.IoctlSetPointerInt(terminal, unix.TIOCSPGRP, g)
	if err != nil {
		println(err.Error())
	}
}

// Stop sends a SIGSTOP to the process ID pid.
func Stop(pid int) {
	_ = unix.Kill(pid, unix.SIGSTOP)
}

// SysProcAttr returns the appropriate *unix.SysProcAttr given the group ID
// and if this is for the foreground group.
func SysProcAttr(foreground bool, group int) *unix.SysProcAttr {
	sys := &unix.SysProcAttr{Foreground: foreground, Setpgid: true}

	if group == 0 {
		sys.Ctty = terminal
	} else {
		sys.Pgid = group
	}

	return sys
}

// Terminate sends a SIGTERM to the process ID pid.
func Terminate(pid int) {
	_ = unix.Kill(pid, unix.SIGTERM)
}
