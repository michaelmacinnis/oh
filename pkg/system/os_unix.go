// Released under an MIT license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd solaris

package system

import (
	"os"
	"path"
	"syscall"
	"unsafe"
)

var (
	Platform = "unix"
	history  = ""
)

func BecomeProcessGroupLeader() {
	if pid != pgid {
		syscall.Setpgid(0, 0)
	}
}

func ContinueProcess(pid int) {
	syscall.Kill(pid, syscall.SIGCONT)
}

func GetHistoryFilePath() (string, error) {
	if history == "" {
		history = path.Join(os.Getenv("HOME"), ".oh_history")
	}
	return history, nil
}

func JobControlSupported() bool {
	return true
}

func ResetForegroundGroup(f *os.File) bool {
	if f != os.Stdin {
		return false
	}

	if pgid <= 0 {
		return false
	}

	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&pgid)))

	return true
}

func SetForegroundGroup(group int) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&group)))
}

func SuspendProcess(pid int) {
	syscall.Kill(pid, syscall.SIGSTOP)
}

func SysProcAttr(group int) *syscall.SysProcAttr {
	sys := &syscall.SysProcAttr{}

	if group == 0 {
		sys.Ctty = syscall.Stdout
		sys.Foreground = true
	} else {
		sys.Setpgid = true
		sys.Pgid = group
	}

	return sys
}

func TerminateProcess(pid int) {
	syscall.Kill(pid, syscall.SIGTERM)
}

func getpgrp() int {
	return syscall.Getpgrp()
}
