// Released under an MIT license. See LICENSE.

// +build windows

package system

import (
	"golang.org/x/sys/windows"
)

var Platform = "windows"

func init() {
	pgid = -1
	pid = windows.Getpid()
	ppid = windows.Getppid()
}
