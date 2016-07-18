// Released under an MIT license. See LICENSE.

// +build plan9

package task

import (
	"os"
	"syscall"
)

func exitStatus(proc *os.Process) int {
	status, err := proc.Wait()
	if err != nil {
		return -1
	}

	return status.Sys().(syscall.Waitmsg).ExitStatus()
}
