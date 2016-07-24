// Released under an MIT license. See LICENSE.

// +build windows

package task

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"os"
	"syscall"
)

func exit(c Cell) {
	if c == Null {
		os.Exit(0)
	}

	a, ok := c.(Atom)
	if !ok {
		os.Exit(1)
	}

	os.Exit(int(a.Status()))
}

func status(proc *os.Process) *Status {
	status, err := proc.Wait()
	if err != nil {
		return ExitFailure
	}

	return NewStatus(int64(status.Sys().(syscall.WaitStatus).ExitStatus()))
}
