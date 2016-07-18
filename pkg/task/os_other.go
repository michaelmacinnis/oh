// Released under an MIT license. See LICENSE.

// +build !linux,!darwin,!dragonfly,!freebsd,!openbsd,!netbsd,!solaris

package task

import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
)

func initPlatformSpecific() {}

func initSignalHandling() {}

func evaluate(c Cell, file string, line int, problem string) (Cell, bool) {
	task0.Eval <- Message{Cmd: c, File: file, Line: line, Problem: problem}
	return <-task0.Done, task0.Stack != Null
}
