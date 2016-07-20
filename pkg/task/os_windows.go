// Released under an MIT license. See LICENSE.

// +build windows

package task

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"os"
	"syscall"
)

func exitStatus(proc *os.Process) *Status {
	status, err := proc.Wait()
	if err != nil {
		return ExitFailure
	}

	return NewStatus(int64(status.Sys().(syscall.WaitStatus).ExitStatus()))
}
