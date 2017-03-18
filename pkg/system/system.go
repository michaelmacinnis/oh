// Released under an MIT license. See LICENSE.

package system

var (
	pgid     int
	pid      int
	ppid     int
	terminal = -1
)

func Pgid() int {
	return pgid
}

func Pid() int {
	return pid
}

func Ppid() int {
	return ppid
}
