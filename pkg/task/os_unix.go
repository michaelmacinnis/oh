// Released under an MIT-style license. See LICENSE.

// +build linux darwin dragonfly freebsd openbsd netbsd solaris

package task

import (
	"fmt"
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

var (
	Platform string = "unix"
	done0    chan Cell
	eval0    chan Cell
	incoming chan os.Signal
)

func BecomeProcessGroupLeader() int {
	pid := syscall.Getpid()
	pgid := syscall.Getpgrp()
	if pid != pgid {
		syscall.Setpgid(0, 0)
	}

	return pid
}

func ContinueProcess(pid int) {
	syscall.Kill(pid, syscall.SIGCONT)
}

func InitSignalHandling() {
	signal.Ignore(syscall.SIGTTOU, syscall.SIGTTIN)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTSTP}
	incoming = make(chan os.Signal, len(signals))

	signal.Notify(incoming, signals...)

	go broker()
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
				if pid == task0.Job.Group {
					incoming <- syscall.SIGTSTP
				}
				continue
			}

			if status.Signaled() {
				if status.Signal() == syscall.SIGINT &&
					pid == task0.Job.Group {
					incoming <- syscall.SIGINT
				}
				status += 128
			}

			notify <- Notification{pid, status}
			monitoring = <-active
		}
	}
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

func SetForegroundGroup(group int) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&group)))
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

func broker() {
	var c Cell
	for c == nil && task0.Stack != Null {
		for c == nil {
			select {
			case <-incoming:
			case c = <-eval0:
			}
		}
		task0.Eval <- c
		for c != nil {
			prev := task0
			select {
			case sig := <-incoming:
				// Handle signals.
				switch sig {
				case syscall.SIGTSTP:
					task0.Suspend()
					last := 0
					for k := range jobs {
						if k > last {
							last = k
						}
					}
					last++

					jobs[last] = task0

					fallthrough
				case syscall.SIGINT:
					if sig == syscall.SIGINT {
						task0.Stop()
					}
					fmt.Printf("\n")

					LaunchForegroundTask()
					c = nil
				}

			case c = <-task0.Done:
				if task0 != prev {
					c = Null
					continue
				}
			}
		}
		done0 <- c
	}
	os.Exit(status(Car(task0.Scratch)))
}

func evaluate(c Cell) {
	eval0 <- c
	<-done0

	task0.Job.Command = ""
	task0.Job.Group = 0
}

func init() {
	done0 = make(chan Cell)
	eval0 = make(chan Cell)
}
