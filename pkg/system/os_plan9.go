// Released under an MIT license. See LICENSE.

// +build plan9

package system

import (
	"golang.org/x/sys/plan9"
)

var Platform = "plan9"

func init() {
	pgid = -1
	pid = plan9.Getpid()
	ppid = plan9.Getppid()
}
