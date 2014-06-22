/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"os"
	"os/signal"
	"syscall"
)

type Notification struct {
	pid    int
	status syscall.WaitStatus
}

type Registration struct {
	pid int
	cb  chan Notification
}

var incoming chan os.Signal
var register chan Registration

func init() {
	active := make(chan bool)
	notify := make(chan Notification)
	register = make(chan Registration)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTSTP}
	incoming = make(chan os.Signal, len(signals))
	signal.Notify(incoming, signals...)

	go monitor(active, notify)
	go registrar(active, notify)
}

func monitor(active chan bool, notify chan Notification) {
	for <-active {
		for {
			var rusage syscall.Rusage
			var status syscall.WaitStatus
			options := syscall.WUNTRACED
			pid, e := syscall.Wait4(-1, &status, options, &rusage)
			if e != nil || pid <= 0 {
				continue
			}

			if status.Stopped() {
				if pid == ForegroundTask().Job.group {
					incoming <- syscall.SIGTSTP
				}
			} else if status.Signaled() &&
				status.Signal() == syscall.SIGINT {
				if pid == ForegroundTask().Job.group {
					incoming <- syscall.SIGINT
				}
			} else {
				notify <- Notification{pid, status}
				if !<-active {
					break
				}
			}
		}
	}
	panic("This should never happen.")
}

func registrar(active chan bool, notify chan Notification) {
	registered := make(map[int]Registration)
	for {
		select {
		case n := <-notify:
			r, ok := registered[n.pid]
			if ok {
				if n.status.Exited() {
					r.cb <- n
					delete(registered, n.pid)
				}
			}
			active <- len(registered) != 0
		case r := <-register:
			registered[r.pid] = r
			if len(registered) == 1 {
				active <- true
			}
		}
	}
}

func Incoming() chan os.Signal {
	return incoming
}

func JoinProcess(pid int) syscall.WaitStatus {
	cb := make(chan Notification)
	register <- Registration{pid, cb}
	n := <-cb

	return n.status
}
