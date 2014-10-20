// Released under an MIT-style license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd

package main

// TODO: solaris should also be in the list above.

import (
	"os"
	"syscall"
	"unsafe"
)

//#include <signal.h>
//#include <unistd.h>
//void ignore(void) {
//      signal(SIGTTOU, SIG_IGN);
//      signal(SIGTTIN, SIG_IGN);
//}
import "C"

var (
	InterruptRequest os.Signal = os.Interrupt
	StopRequest      os.Signal = syscall.SIGTSTP
)

func GrabTerminal(group *int) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(group)))
}

func InitSignalHandling() {
	C.ignore()
}

func InputIsTTY() bool {
	return C.isatty(C.int(0)) != 0
}

func JobControlSupported() bool {
	return true
}

func JoinProcess(proc *os.Process) int {
	response := make(chan Notification)
	register <- Registration{proc.Pid, response}

	return (<-response).status.ExitStatus()
}

func Monitor(active chan bool, notify chan Notification) {
	for {
		monitoring := <-active
		for monitoring {
			var rusage syscall.Rusage
			var status syscall.WaitStatus
			options := syscall.WUNTRACED
			pid, err := syscall.Wait4(-1, &status, options, &rusage)
			if err != nil {
				println("Wait4:", err.Error())
			}
			if pid <= 0 {
				break
			}

			if status.Stopped() {
				if pid == ForegroundTask().Job.group {
					incoming <- syscall.SIGTSTP
				}
				continue
			}

			if status.Signaled() {
				if status.Signal() == syscall.SIGINT &&
					pid == ForegroundTask().Job.group {
					incoming <- syscall.SIGINT
				}
				status += 128
			}

			notify <- Notification{pid, status}
			monitoring = <-active
		}
	}
	panic("unreachable")
}

func Registrar(active chan bool, notify chan Notification) {
	preregistered := make(map[int]Notification)
	registered := make(map[int]Registration)
	for {
		select {
		case n := <-notify:
			r, ok := registered[n.pid]
			if ok {
				r.cb <- n
				delete(registered, n.pid)
			} else {
				preregistered[n.pid] = n
			}
			active <- len(registered) != 0
		case r := <-register:
			if n, ok := preregistered[r.pid]; ok {
				r.cb <- n
				delete(preregistered, r.pid)
			} else {
				registered[r.pid] = r
				if len(registered) == 1 {
					active <- true
				}
			}
		}
	}
}

func SetProcessGroup() int {
	pid := syscall.Getpid()
	pgid := syscall.Getpgrp()
	if pid != pgid {
		syscall.Setpgid(0, 0)
	}

	return pid
}

func SysProcAttr(group int) *syscall.SysProcAttr {
	sys := &syscall.SysProcAttr{}

	if group == 0 {
		sys.Setpgid = true
		sys.Foreground = true
	} else {
		sys.Joinpgrp = group
	}

	return sys
}
