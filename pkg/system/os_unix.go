// Released under an MIT license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd solaris

package system

import (
	"github.com/mattn/go-isatty"
	"golang.org/x/sys/unix"
	"os"
	"path"
	"syscall"
)

var (
	Platform = "unix"
	history  = ""
)

func BecomeForegroundProcessGroup() {
	for pgid != getForegroundGroup(terminal) {
		unix.Kill(-pgid, unix.SIGTTIN)
		pgid, _ = unix.Getpgid(pid)
	}
}

func BecomeProcessGroupLeader() {
	if pid != pgid {
		unix.Setpgid(pid, pid)
	}
	pgid = pid
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

func ResetForegroundGroup(err error) bool {
	perr, ok := err.(*os.PathError)
	if !ok || perr.Err != unix.EIO || pgid <= 0 {
		return false
	}

	SetForegroundGroup(pgid)

	return true
}

func SetForegroundGroup(pgid int) {
	if terminal < 0 {
		return
	}
	setForegroundGroup(terminal, pgid)
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
	for _, fd := range []int{unix.Stdin, unix.Stdout, unix.Stderr} {
		if isatty.IsTerminal(uintptr(fd)) {
			terminal = fd
		}
	}
}
