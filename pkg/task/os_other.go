// Released under an MIT license. See LICENSE.

// +build !linux,!darwin,!dragonfly,!freebsd,!openbsd,!netbsd,!solaris

package task

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
)

func initPlatformSpecific() {}

func initSignalHandling(ui) {}

func evaluate(c Cell, f string, l int) (Cell, bool) {
	task0.Eval <- message{cmd: c, file: f, line: l}
	return <-task0.Done, task0.Stack != Null
}
