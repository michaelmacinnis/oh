// Released under an MIT-style license. See LICENSE.

// +build windows

package main

import (
	"os"
	"syscall"
)

var (
	InterruptRequest os.Signal = os.Interrupt
	StopRequest      os.Signal = os.Kill
)

func ContinueProcess(pid int) {}

func InitSignalHandling() {}

func InputIsTTY() bool {
	// TODO: Not sure what to do on non-Unix platforms.
	return true
}

func JobControlSupported() bool {
	return false
}

func JoinProcess(proc *os.Process) int {
	status, err := proc.Wait()
	if err != nil {
		return -1
	}

	return status.Sys().(syscall.WaitStatus).ExitStatus()
}

func Monitor(active chan bool, notify chan Notification) {}

func Registrar(active chan bool, notify chan Notification) {}

func SetForegroundGroup(group int) {}

func SetProcessGroup() int {
	// TODO: Not sure what to do on non-Unix platforms.
	return 0
}

func SysProcAttr(group int) *syscall.SysProcAttr {
	return nil
}

func TerminateProcess(pid int) {}
