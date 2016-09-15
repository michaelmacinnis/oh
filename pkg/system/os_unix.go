// Released under an MIT license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd solaris

package system

import (
	"golang.org/x/sys/unix"
	"os"
	"path"
	"syscall"
)

var (
	Platform = "unix"
	history  = ""
)

func BecomeProcessGroupLeader() {
	if pid != pgid {
		unix.Setpgid(0, 0)
	}
}

func ContinueProcess(pid int) {
	unix.Kill(pid, unix.SIGCONT)
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

	SetForegroundGroup(pgid)

	return true
}

func SuspendProcess(pid int) {
	unix.Kill(pid, unix.SIGSTOP)
}

func SysProcAttr(group int) *syscall.SysProcAttr {
	sys := &syscall.SysProcAttr{}

	if group == 0 {
		sys.Ctty = unix.Stdout
		sys.Foreground = true
	} else {
		sys.Setpgid = true
		sys.Pgid = group
	}

	return sys
}

func TerminateProcess(pid int) {
	unix.Kill(pid, unix.SIGTERM)
}

func init() {
	pid = unix.Getpid()
	pgid, _ = unix.Getpgid(pid)
	ppid = unix.Getppid()
}
