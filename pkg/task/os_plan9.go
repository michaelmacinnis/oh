// Released under an MIT license. See LICENSE.

// +build plan9

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

	return NewStatus(status.Sys().(syscall.Waitmsg).Msg)
}
