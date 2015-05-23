// Released under an MIT-style license. See LICENSE.

// +build windows

package task

import (
	"os"
	"syscall"
)

var Platform string = "windows"

func BecomeProcessGroupLeader() int {
	// TODO: Not sure what to do on non-Unix platforms.
	return 0
}

func ContinueProcess(pid int) {}

func InitSignalHandling() {}

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

func SysProcAttr(group int) *syscall.SysProcAttr {
	return nil
}

func TerminateProcess(pid int) {}

func evaluate(c Cell) {
	task0.Eval <- c
	<-task0.Done
}
