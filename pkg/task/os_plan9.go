// Released under an MIT license. See LICENSE.

// +build plan9

package task

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"os"
	"syscall"
)

func exit(c Cell) {
	if c == Null {
		// syscall.Exits("")
		os.Exit(0)
	}

	a, ok := c.(Atom)
	if !ok {
		// syscall.Exits("unknown status")
		os.Exit(1)
	}

	// syscall.Exits(a.Status())
	os.Exit(int(NewStatus(a.Status()).Int()))
}

func status(proc *os.Process) *Status {
	status, err := proc.Wait()
	if err != nil {
		return ExitFailure
	}

	return NewStatus(status.Sys().(syscall.Waitmsg).Msg)
}
